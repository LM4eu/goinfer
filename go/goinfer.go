// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package main

import (
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
	"github.com/LM4eu/llama-swap/proxy/config"
	"github.com/labstack/echo/v4"

	"github.com/LM4eu/goinfer/conf"
	"github.com/LM4eu/goinfer/infer"
)

const (
	goinferYML    = "goinfer.yml"
	llamaSwapYML  = "llama-swap.yml"
	templateJinja = "template.jinja"
)

func main() {
	cfg := getCfg()
	startServers(cfg)
}

// Config is created from lower to higher priority: (1) config files, (2) env. Vars. And (3) flags.
// Depending on the flags, this function also creates config files and exits.
func getCfg() *conf.Cfg {
	quiet := flag.Bool("q", false, "quiet mode (disable verbose output)")
	debug := flag.Bool("debug", false, "debug mode (set debug ABI keys with -gen)")
	noAPIKey := flag.Bool("no-api-key", false, "disable API key check/generation (with -gen)")
	genCfg := flag.Bool("gen", false, "generate "+mainCfg)
	vv.SetVersionFlag()
	flag.Parse()

	cfg := conf.DefaultCfg

	cfg.SetLogLevel(!*quiet, *debug)

	// generate "goinfer.yml"
	if *genCfg {
		err := cfg.WriteMainCfg(mainCfg, *debug, *noAPIKey)
		if err != nil {
			slog.Error("Cannot create main config", "file", mainCfg, "error", err)
			os.Exit(1)
		}
	}

	// verify "goinfer.yml" can be successfully loaded
	err := cfg.ReadMainCfg(mainCfg, *noAPIKey)
	if err != nil {
		slog.Error("Cannot load main config", "file", mainCfg, "error", err)
		os.Exit(1)
	}

	// successfully generated "goinfer.yml"
	if *genCfg {
		slog.Info("Generated main", "config", mainCfg)
		if !*quiet {
			cfg.Print()
		}
		os.Exit(0)
	}

	// generate "template.jinja"
	err = os.WriteFile("template.jinja", []byte("{{- messages[0].content -}}"), 0o600)
	if err != nil {
		slog.Error("Cannot write", "file", "template.jinja", "error", err)
		os.Exit(1)
	}

	// generate "llama-swap.yml"
	err := cfg.WriteSwapCfg(llamaSwapYML, verbose, debug)
	if err != nil {
		slog.Error("Failed creating a valid llama-swap config", "file", llamaSwapYML, "error", err)
		os.Exit(1)
	}

	// verify "llama-swap.yml" can be successfully loaded
	cfg.Swap, err = config.LoadConfig(llamaSwapYML)
	if err != nil {
		slog.Error("Cannot load llama-swap config", "file", llamaSwapYML, "error", err)
		os.Exit(1)
	}
	err = cfg.ValidateSwap()
	if err != nil {
		slog.Error("llama-swap config ", "file", llamaSwapYML, "error", err)
		os.Exit(1)
	}

	slog.Info("Generated Swap config", "file", swapCfg, "models", len(cfg.Swap.Models))

	// command line precedes config file
	if *noAPIKey {
		cfg.APIKey = ""
	}

	if *debug {
		cfg.Print()
	}

	return &cfg
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
		if !strings.Contains(services, "infer") {
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
