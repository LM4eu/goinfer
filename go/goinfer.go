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

	"github.com/labstack/echo/v4"
	"github.com/mostlygeek/llama-swap/proxy"
	"github.com/teal-finance/garcon"

	"github.com/LM4eu/goinfer/conf"
	"github.com/LM4eu/goinfer/infer"
)

const (
	giCfg = "goinfer.yml"
	pxCfg = "llama-swap.yml"
)

func main() {
	cfg := getCfg()
	startServers(cfg)
}

// Config is created from lower to higher priority: (1) config files, (2) env. Vars. And (3) flags.
// Depending on the flags, this function also creates config files and exits.
func getCfg() *conf.GoInferCfg {
	quiet := flag.Bool("q", false, "quiet mode (disable verbose output)")
	debug := flag.Bool("debug", false, "debug mode")
	genGiCfg := flag.Bool("gen-gi-cfg", false, "generate "+giCfg+" (main config file)")
	genPxCfg := flag.Bool("gen-px-cfg", false, "generate "+pxCfg+" (proxy config file)")
	noAPIKey := flag.Bool("no-api-key", false, "disable API key check")
	garcon.SetVersionFlag()
	flag.Parse()

	var cfg conf.GoInferCfg

	if *debug {
		slog.Debug("Debug mode is on")
		cfg.Debug = true
	}

	cfg.Verbose = !*quiet

	// Generate config
	if *genGiCfg {
		err := cfg.Write(giCfg, *noAPIKey)
		if err != nil {
			slog.Error("Cannot create main config", "file", giCfg, "error", err)
			os.Exit(1)
		}
	}

	// Verify we can upload the config
	err := cfg.Read(giCfg, *noAPIKey)
	if err != nil {
		slog.Error("Cannot load main config", "file", giCfg, "error", err)
		os.Exit(1)
	}

	if cfg.Verbose {
		cfg.Print()
	}

	if *genGiCfg {
		slog.Info("Generated main", "config", giCfg)
		os.Exit(0)
	}

	// Load the llama-swap config
	cfg.Proxy, err = proxy.LoadConfig(pxCfg)
	// even if err!=nil => generate the config file
	if *genPxCfg {
		if err != nil {
			slog.Warn("Cannot load proxy config", "file", pxCfg, "error", err)
		}
		err = cfg.CreateProxyCfg(pxCfg)
		if err != nil {
			slog.Error("Cannot create proxy config", "file", pxCfg, "error", err)
			os.Exit(1)
		}
		slog.Info("Generated proxy config", "file", pxCfg, "models", len(cfg.Proxy.Models))
		os.Exit(0)
	}
	if err != nil {
		slog.Error("Cannot load proxy config", "file", pxCfg, "error", err)
		os.Exit(1)
	}

	if *noAPIKey {
		cfg.Server.APIKeys = nil
	}

	if cfg.Debug {
		cfg.Print()
	}

	return &cfg
}

// startServers creates and runs the HTTP servers.
func startServers(cfg *conf.GoInferCfg) {
	// Create context with cancel for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling and start shutdown handler
	go handleShutdown(ctx, cancel)

	// Use errgroup to coordinate the servers shutdown
	var grp errgroup.Group

	// Start Echo and proxy servers if configured
	startEchoServers(ctx, cfg, &grp)
	startProxyServer(ctx, cfg, &grp)

	// prints a startup message when all servers are running.
	if cfg.Verbose {
		slog.Info("-----------------------------")
		slog.Info("All HTTP servers started. Press CTRL+C to stop.")
	}

	// Wait for all servers to complete
	err := grp.Wait()
	if err != nil {
		slog.Error("Server error", "error", err)
	} else {
		slog.Info("All HTTP servers stopped gracefully")
	}
}

// startEchoServers starts all HTTP Echo servers configured in the config.
func startEchoServers(ctx context.Context, cfg *conf.GoInferCfg, grp *errgroup.Group) {
	inf := &infer.Infer{Cfg: cfg}
	for addr, services := range cfg.Server.Listen {
		if strings.Contains(services, "swap") {
			continue // reserved for llama-swap proxy
		}

		enableAdminWebUI := strings.Contains(services, "admin")
		enableModelsEndpoint := strings.Contains(services, "model")
		enableGoinferEndpoint := strings.Contains(services, "goinfer")
		enableOpenAPIEndpoint := strings.Contains(services, "openai")

		if !enableAdminWebUI && !enableModelsEndpoint && !enableGoinferEndpoint && !enableOpenAPIEndpoint {
			slog.ErrorContext(ctx, "Unexpected", "service", services)
			os.Exit(1)
		}

		e := inf.NewEcho(cfg, addr, enableAdminWebUI, enableModelsEndpoint, enableGoinferEndpoint, enableOpenAPIEndpoint)
		if e != nil {
			if cfg.Verbose {
				slog.InfoContext(ctx, "-----------------------------")
				slog.InfoContext(ctx, "Echo listen", "addr", addr, "origins", cfg.Server.Origins)
				slog.InfoContext(ctx, "Echo endpoint", "WebUI", enableAdminWebUI)
				slog.InfoContext(ctx, "Echo endpoint", "/models", enableModelsEndpoint)
				slog.InfoContext(ctx, "Echo endpoint", "/goinfer", enableGoinferEndpoint)
				slog.InfoContext(ctx, "Echo endpoint", "OpenAI", enableOpenAPIEndpoint)
			}

			grp.Go(func() error {
				return startEcho(ctx, cfg, e, addr)
			})
		}
	}
}

// startProxyServer starts the llama-swap proxy if configured in the config.
func startProxyServer(ctx context.Context, cfg *conf.GoInferCfg, grp *errgroup.Group) {
	for addr, services := range cfg.Server.Listen {
		if !strings.Contains(services, "swap") {
			continue
		}

		proxyHandler := proxy.New(cfg.Proxy)
		proxyServer := &http.Server{
			Addr:    addr,
			Handler: proxyHandler,
		}

		if cfg.Verbose {
			slog.InfoContext(ctx, "-----------------------------")
			slog.InfoContext(ctx, "Gin server (llama-swap proxy) listen", "addr", proxyServer.Addr)
		}

		grp.Go(func() error {
			return startProxy(ctx, cfg, proxyServer, proxyHandler)
		})
	}
}

// startEcho starts a HTTP server with graceful shutdown handling.
func startEcho(ctx context.Context, cfg *conf.GoInferCfg, e *echo.Echo, addr string) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- e.Start(addr)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return stopEcho(ctx, cfg, e, addr)
	}
}

// startProxy starts a llama-swap proxy server with graceful shutdown handling.
func startProxy(ctx context.Context, cfg *conf.GoInferCfg, proxyServer *http.Server, proxyHandler http.Handler) error {
	err := make(chan error, 1)
	go func() {
		err <- proxyServer.ListenAndServe()
	}()

	select {
	case er := <-err:
		return er
	case <-ctx.Done():
		return stopProxy(ctx, cfg, proxyServer, proxyHandler)
	}
}

// stopEcho performs graceful shutdown of an Echo server.
func stopEcho(ctx context.Context, cfg *conf.GoInferCfg, e *echo.Echo, addr string) error {
	if cfg.Verbose {
		slog.InfoContext(ctx, "Shutting down Echo", "addr", addr)
	}

	err := e.Shutdown(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Echo shutdown", "addr", addr, "error", err)
		return err
	}

	if cfg.Verbose {
		slog.InfoContext(ctx, "Echo stopped gracefully", "addr", addr)
	}
	return nil
}

// stopProxy performs graceful shutdown of a llama-swap proxy server.
func stopProxy(ctx context.Context, cfg *conf.GoInferCfg, proxyServer *http.Server, proxyHandler http.Handler) error {
	if cfg.Verbose {
		slog.InfoContext(ctx, "Shutting down Proxy (Gin)", "addr", proxyServer.Addr)
	}

	// Check if proxyHandler has a Shutdown method
	if shutdownHandler, ok := proxyHandler.(interface{ Shutdown() }); ok {
		shutdownHandler.Shutdown()
	}

	err := proxyServer.Shutdown(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Proxy shutdown", "error", err)
		return err
	}

	if cfg.Verbose {
		slog.InfoContext(ctx, "Proxy stopped gracefully", "addr", proxyServer.Addr)
	}
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
