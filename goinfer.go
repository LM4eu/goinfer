// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"fmt"
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
	goInferCfgFile = "goinfer.yml"
	proxyCfgFile   = "llama-swap.yml"
)

func main() {
	cfg := getFlagsCfg()
	runHTTPServers(cfg)
}

func getFlagsCfg() *conf.GoInferCfg {
	quiet := flag.Bool("q", false, "quiet mode (disable verbose output)")
	debug := flag.Bool("debug", false, "debug mode")
	genGiCfg := flag.Bool("gen-gi-cfg", false, "generate "+goInferCfgFile)
	genPxCfg := flag.Bool("gen-px-cfg", false, "generate "+proxyCfgFile+" (proxy config file)")
	noAPIKey := flag.Bool("no-api-key", false, "disable API key check")
	garcon.SetVersionFlag()
	flag.Parse()

	var cfg conf.GoInferCfg

	if *debug {
		fmt.Println("DBG: Debug mode is on")
		cfg.Debug = true
	}

	cfg.Verbose = !*quiet

	// Generate config
	if *genGiCfg {
		err := cfg.Create(goInferCfgFile, *noAPIKey)
		if err != nil {
			fmt.Printf("ERROR creating config: %v\n", err)
			os.Exit(1)
		}
	}

	// Verify we can upload the config
	err := cfg.Load(goInferCfgFile, *noAPIKey)
	if err != nil {
		fmt.Printf("ERROR loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Verbose {
		cfg.Print()
	}

	if *genGiCfg {
		fmt.Printf("INF: Configuration file %s created successfully.\n", goInferCfgFile)
		os.Exit(0)
	}

	// Load the llama-swap config
	cfg.Proxy, err = proxy.LoadConfig(proxyCfgFile)
	// even if err!=nil => generate the config file,
	if *genPxCfg {
		err = cfg.GenProxyCfg(proxyCfgFile)
		if err != nil {
			fmt.Printf("ERROR generating proxy config: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if err != nil {
		fmt.Printf("ERROR loading proxy config: %v\n", err)
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

func runHTTPServers(cfg *conf.GoInferCfg) {
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
		fmt.Println("-----------------------------")
		fmt.Println("INF: All servers started. Press CTRL+C to stop.")
	}

	// Wait for all servers to complete
	err := grp.Wait()
	if err != nil {
		fmt.Printf("ERROR: Server error: %v\n", err)
	} else {
		fmt.Println("INF: All HTTP servers stopped gracefully")
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
			fmt.Printf("ERROR Unexpected service %q - does not contain any of: model, goinfer, openai, admin\n", services)
			os.Exit(1)
		}

		e := inf.NewEcho(cfg, addr, enableAdminWebUI, enableModelsEndpoint, enableGoinferEndpoint, enableOpenAPIEndpoint)
		if e != nil {
			if cfg.Verbose {
				fmt.Println("-----------------------------")
				fmt.Println("Starting Echo server:")
				fmt.Println("- Admin web UI    : ", enableAdminWebUI)
				fmt.Println("- models  endpoint: ", enableModelsEndpoint)
				fmt.Println("- goinfer endpoint: ", enableGoinferEndpoint)
				fmt.Println("- OpenAI endpoints: ", enableOpenAPIEndpoint)
				fmt.Println("- listen:  ", addr)
				fmt.Println("- origins: ", cfg.Server.Origins)
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
			fmt.Println("-----------------------------")
			fmt.Println("Starting Gin server (llama-swap proxy)")
			fmt.Println("- listen:  ", proxyServer.Addr)
		}

		grp.Go(func() error {
			return startProxy(ctx, cfg, proxyServer, proxyHandler)
		})
	}
}

// startEcho starts a HTTP server with graceful shutdown handling.
func startEcho(ctx context.Context, cfg *conf.GoInferCfg, e *echo.Echo, addr string) error {
	err := make(chan error, 1)
	go func() {
		err <- e.Start(addr)
	}()

	select {
	case er := <-err:
		return er
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
		fmt.Printf("INF: Shutting down Echo server on %s\n", addr)
	}

	err := e.Shutdown(ctx)
	if err != nil {
		fmt.Printf("ERROR: Echo server shutdown error on %s: %v\n", addr, err)
		return err
	}

	if cfg.Verbose {
		fmt.Printf("INF: Echo server on %s stopped gracefully\n", addr)
	}
	return nil
}

// stopProxy performs graceful shutdown of a llama-swap proxy server.
func stopProxy(ctx context.Context, cfg *conf.GoInferCfg, proxyServer *http.Server, proxyHandler http.Handler) error {
	if cfg.Verbose {
		fmt.Printf("INF: Shutting down proxy server on %s\n", proxyServer.Addr)
	}

	// Check if proxyHandler has a Shutdown method
	if shutdownHandler, ok := proxyHandler.(interface{ Shutdown() }); ok {
		shutdownHandler.Shutdown()
	}

	err := proxyServer.Shutdown(ctx)
	if err != nil {
		fmt.Printf("ERROR: Proxy server shutdown error: %v\n", err)
		return err
	}

	if cfg.Verbose {
		fmt.Printf("INF: Proxy server on %s stopped gracefully\n", proxyServer.Addr)
	}
	return nil
}

// handleShutdown handles graceful shutdown upon receiving a signal.
func handleShutdown(ctx context.Context, cancel context.CancelFunc) {
	// sets up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Printf("INF: Received signal %v, initiating graceful shutdown...\n", sig)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for graceful shutdown completion or timeout
	select {
	case <-time.After(10 * time.Second):
		fmt.Println("WARNING: Graceful shutdown timed out, forcing exit")
		os.Exit(1)
	case <-ctx.Done():
		fmt.Println("INF: Graceful shutdown completed")
	}
}
