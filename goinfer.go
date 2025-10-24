// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/LM4eu/garcon/vv"
	"github.com/LM4eu/llama-swap/proxy"
	"github.com/labstack/echo/v4"

	"github.com/LM4eu/goinfer/conf"
	"github.com/LM4eu/goinfer/infer"
)

const templateJinja = "template.jinja"

func main() {
	cfg := getCfg()
	startServers(cfg)
}

// Config is created from lower to higher priority: (1) config files, (2) env. Vars. And (3) flags.
// Depending on the flags, this function also creates config files and exits.
func getCfg() *conf.Cfg {
	quiet := flag.Bool("q", false, "quiet mode (disable verbose output)")
	debug := flag.Bool("debug", false, "debug mode (with -write: set debug ABI keys)")
	write := flag.Bool("write", false, "write config files: "+conf.GoinferINI+" "+templateJinja+" "+conf.LlamaSwapYML)
	run := flag.Bool("run", false, "run the server, can be combined with -write")
	extra := flag.String("hf", "", "configure the given extra_models and load the first one (start llama-server)")
	start := flag.String("start", "", "set the default_model and load it (start llama-server)")
	noAPIKey := flag.Bool("no-api-key", false, "disable API key check (with -write: set a warning in place of the API key)")
	vv.SetVersionFlag()
	flag.Parse()

	verbose := !*quiet

	if *extra != "" || *start != "" { // -hf and -start implies -run
		*run = true
	}

	switch {
	case *debug:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case verbose:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	default:
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}
	slog.Debug("debug mode")

	cfg := doGoinferINI(*debug, *write, *run, *noAPIKey, *extra, *start)

	if *write || verbose {
		cfg.Print()
	}

	// if -write without -run => stop here, just successfully generated "goinfer.ini"
	if *write && !*run {
		os.Exit(0)
	}

	doLlamaSwapYML(cfg, *write, verbose, *debug)

	return cfg
}

func doGoinferINI(debug, write, run, noAPIKey bool, extra, start string) *conf.Cfg {
	cfg, err := conf.ReadGoinferINI(noAPIKey, extra, start)
	if err != nil {
		switch {
		case write:
			slog.Info("Write a fresh new config file, may contain issues: "+
				"please verify the config.", "file", conf.GoinferINI, "error", err)
		case run:
			slog.Error("Error config. Flags -run -start -hf without -write prevent to write it. "+
				"Add flag -write to write the config", "file", conf.GoinferINI, "error", err)
			os.Exit(1)
		default:
			slog.Info("Error config => Write a new one using default values, env vars and flags. "+
				"May contain issues: please verify the config.", "file", conf.GoinferINI, "error", err)
		}
		write = true
	}

	if write {
		err = os.WriteFile(templateJinja, []byte("{{- messages[0].content -}}"), 0o600)
		if err != nil {
			slog.Error("Cannot write", "file", templateJinja, "error", err)
		}

		er := cfg.WriteGoinferINI(debug, noAPIKey)
		if er != nil {
			slog.Info(er.Error())
			err = er
		}

		// read "goinfer.ini" to verify it can be successfully loaded
		// Pass empty extra and start to keep the eventual fixes.
		cfg, er = conf.ReadGoinferINI(noAPIKey, "", "")
		if er != nil {
			slog.Warn("Please review "+conf.GoinferINI, "error", er)
			os.Exit(1)
		}

		// stop if any error from WriteFile(templateJinja) or WriteGoinferINI
		if err != nil {
			slog.Info("Please review the env. vars and " + conf.GoinferINI)
			os.Exit(1)
		}

		slog.Info("Successfully wrote " + conf.GoinferINI)
	}

	// command line precedes config file
	if noAPIKey {
		cfg.APIKey = ""
	}

	return cfg
}

func doLlamaSwapYML(cfg *conf.Cfg, write, verbose, debug bool) {
	yml, err := cfg.GenSwapYAMLData(verbose, debug)
	if err != nil {
		slog.Error("Failed creating a valid llama-swap config", "file", conf.LlamaSwapYML, "error", err)
		os.Exit(1)
	}

	if write {
		err = conf.WriteLlamaSwapYML(yml)
		if err != nil {
			slog.Warn("Failed writing the llama-swap config", "file", conf.LlamaSwapYML, "error", err)
		}
		slog.Info("Generated llama-swap config", "file", conf.LlamaSwapYML, "models", len(cfg.Swap.Models))
	}

	reader := bytes.NewReader(yml)
	err = cfg.ReadSwapFromReader(reader)
	if err != nil {
		slog.Error("Invalid llama-swap config (use flag -write to check "+conf.LlamaSwapYML+")", "error", err)
		os.Exit(1)
	}
}

// startServers creates and runs the HTTP servers.
func startServers(cfg *conf.Cfg) {
	// Create context with cancel for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling and start shutdown handler
	go handleShutdown(ctx, cancel)

	// Use errgroup to coordinate the servers shutdown
	var grp errgroup.Group

	// Start llama-swap (Gin) and Echo servers (if configured)
	proxyMan := startSwapServer(ctx, cfg, &grp)
	startEchoServers(ctx, cfg, &grp, proxyMan)

	// prints a startup message when all servers are running.
	slog.Info("-----------------------------")
	slog.Info("All HTTP servers started. Press CTRL+C to stop.")

	// Wait for all servers to complete
	err := grp.Wait()
	if err != nil {
		slog.Error("Server error", "error", err)
	} else {
		slog.Info("All HTTP servers stopped gracefully")
	}
}

// startEchoServers starts all HTTP Echo servers configured in the config.
func startEchoServers(ctx context.Context, cfg *conf.Cfg, grp *errgroup.Group, proxyMan *proxy.ProxyManager) {
	inf := &infer.Infer{Cfg: cfg, ProxyMan: proxyMan}
	for addr, services := range cfg.Listen {
		if !strings.Contains(services, "goinfer") {
			continue
		}

		e := inf.NewEcho()
		infer.PrintRoutes(e, addr)
		grp.Go(func() error {
			slog.InfoContext(ctx, "start Echo", "url", url(addr), "origins", cfg.Origins)
			return startEcho(ctx, e, addr)
		})
	}
}

func url(addr string) string {
	if addr != "" && addr[0] == ':' {
		return "http://localhost" + addr
	}
	return "http://" + addr
}

// startSwapServer starts the llama-swap if configured in the config.
func startSwapServer(ctx context.Context, cfg *conf.Cfg, grp *errgroup.Group) *proxy.ProxyManager {
	var proxyMan *proxy.ProxyManager

	for addr, services := range cfg.Listen {
		if !strings.Contains(services, "swap") {
			continue
		}

		proxyMan = proxy.New(cfg.Swap)
		swapServer := &http.Server{
			Addr:    addr,
			Handler: proxyMan,
		}

		grp.Go(func() error {
			slog.DebugContext(ctx, "start llama-swap (Gin)", "url", url(swapServer.Addr))
			return startSwap(ctx, swapServer, proxyMan)
		})
	}

	return proxyMan
}

// startEcho starts a HTTP server with graceful shutdown handling.
func startEcho(ctx context.Context, e *echo.Echo, addr string) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- e.Start(addr)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return stopEcho(ctx, e, addr)
	}
}

// startSwap starts a llama-swap server with graceful shutdown handling.
func startSwap(ctx context.Context, swapServer *http.Server, swapHandler http.Handler) error {
	err := make(chan error, 1)
	go func() {
		err <- swapServer.ListenAndServe()
	}()

	select {
	case er := <-err:
		return er
	case <-ctx.Done():
		return stopSwap(ctx, swapServer, swapHandler)
	}
}

// stopEcho performs graceful shutdown of an Echo server.
func stopEcho(ctx context.Context, e *echo.Echo, addr string) error {
	slog.InfoContext(ctx, "Shutting down Echo", "url", url(addr))

	err := e.Shutdown(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Echo shutdown", "url", url(addr), "error", err)
		return err
	}

	slog.InfoContext(ctx, "Echo stopped gracefully", "url", url(addr))
	return nil
}

// stopSwap performs graceful shutdown of a llama-swap server.
func stopSwap(ctx context.Context, swapServer *http.Server, swapHandler http.Handler) error {
	slog.InfoContext(ctx, "Shutting down llama-swap (Gin)", "url", url(swapServer.Addr))

	// check if swapHandler has a Shutdown method
	if shutdownHandler, ok := swapHandler.(interface{ Shutdown() }); ok {
		shutdownHandler.Shutdown()
	}

	err := swapServer.Shutdown(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "llama-swap shutdown", "error", err)
		return err
	}

	slog.InfoContext(ctx, "llama-swap stopped gracefully", "url", url(swapServer.Addr))
	return nil
}

// handleShutdown handles graceful shutdown upon receiving a signal.
func handleShutdown(ctx context.Context, cancel context.CancelFunc) {
	// sets up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	slog.InfoContext(ctx, "Initiating graceful shutdown, Received", "signal", sig)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for graceful shutdown completion or timeout
	select {
	case <-time.After(10 * time.Second):
		slog.WarnContext(ctx, "Graceful shutdown timed out, forcing exit")
		os.Exit(1)
	case <-ctx.Done():
		slog.InfoContext(ctx, "Graceful shutdown completed")
	}
}
