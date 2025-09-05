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
	"github.com/LM4eu/goinfer/server"
	"github.com/LM4eu/goinfer/state"
)

const (
	goInferCfgFile = "goinfer.yml"
	proxyCfgFile = "llama-swap.yml"
)

func main() {
	quiet := flag.Bool("q", false, "quiet mode (disable verbose output)")
	debug := flag.Bool("debug", false, "debug mode")
	genGiConf := flag.Bool("gen-gi-cfg", false, "generate "+goInferCfgFile)
	genPxConf := flag.Bool("gen-px-cfg", false, "generate "+proxyCfgFile+" (proxy config file)")
	noAPIKeys := flag.Bool("no-api-key", false, "disable API key check")
	garcon.SetVersionFlag()
	flag.Parse()

	if *debug {
		fmt.Println("DBG: Debug mode is on")
		state.Debug = true
	}

	state.Verbose = !*quiet

	cfg := manageCfg(*debug, *genGiConf, *genPxConf, *noAPIKeys)

	runHTTPServers(cfg)
}

func manageCfg(debug, genGiConf, genPxConf, noAPIKeys bool) *conf.GoInferCfg {
	// Generate config
	if genGiConf {
		err := conf.Create(goInferCfgFile, debug)
		if err != nil {
			fmt.Printf("ERROR creating config: %v\n", err)
			os.Exit(1)
		}
		if state.Verbose {
			cfg, er := conf.Load(goInferCfgFile)
			if er != nil {
				fmt.Printf("ERROR loading config: %v\n", er)
				os.Exit(1)
			}
			cfg.Print()
		}
		fmt.Printf("INF: Configuration file %s created successfully.\n", goInferCfgFile)
		os.Exit(0)
	}

	// Load configurations
	cfg, err := conf.Load(goInferCfgFile)
	if err != nil {
		fmt.Printf("ERROR loading config: %v\n", err)
		os.Exit(1)
	}
	cfg.Verbose = state.Verbose

	// Load the llama-swap config
	cfg.Proxy, err = proxy.LoadConfig(proxyCfgFile)
	// even if err!=nil => generate the config file,
	if genPxConf {
		err = conf.GenProxyCfg(cfg, proxyCfgFile)
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

	if noAPIKeys {
		cfg.Server.APIKeys = nil
	}

	if state.Debug {
		cfg.Print()
	}

	return cfg
}

func runHTTPServers(cfg *conf.GoInferCfg) {
	// Create context with cancel for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling and start shutdown handler
	sigChan := setupSignalHandling()
	go handleShutdown(sigChan, ctx, cancel)

	// Start all servers using errgroup for coordination
	var grp errgroup.Group

	// Start HTTP servers and proxy server if configured
	startHTTPServers(ctx, cfg, &grp)

	// Wait for all servers to complete
	err := grp.Wait()
	if err != nil {
		fmt.Printf("ERROR: Server error: %v\n", err)
	} else {
		fmt.Println("INF: All HTTP servers stopped gracefully")
	}
}

// setupSignalHandling sets up signal handling for graceful shutdown.
func setupSignalHandling() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return sigChan
}

// startHTTPServers starts all HTTP servers configured in the config.
func startHTTPServers(ctx context.Context, cfg *conf.GoInferCfg, grp *errgroup.Group) {
	for addr, services := range cfg.Server.Listen {
		if strings.Contains(services, "swap") {
			continue // reserved for llama-swap proxy
		}
		if len(addr) == 0 || addr[0] == ':' {
			addr = cfg.Server.Host + addr
		}
		e := server.NewEcho(cfg, addr, services)
		if e != nil {
			if cfg.Verbose {
				fmt.Println("-----------------------------")
				fmt.Println("Starting Echo server:")
				fmt.Println("- services: ", services)
				fmt.Println("- listen:   ", addr)
				fmt.Println("- origins:  ", cfg.Server.Origins)
			}
			grp.Go(func() error {
				return runServerWithGracefulShutdown(ctx, cfg, e, addr)
			})
		}
	}

	// Initialize proxy server
	proxyServer, proxyHandler := server.NewProxy(cfg)

	if proxyServer != nil {
		if cfg.Verbose {
			fmt.Println("-----------------------------")
			fmt.Println("Starting Gin server:")
			fmt.Println("- services: llama-swap proxy")
			fmt.Println("- listen:   ", proxyServer.Addr)
		}
		grp.Go(func() error {
			return runProxyServerWithGracefulShutdown(ctx, cfg, proxyServer, proxyHandler)
		})
	}

	// prints a startup message when all servers are running.
	if cfg.Verbose {
		fmt.Println("-----------------------------")
		fmt.Println("INF: All servers started. Press CTRL+C to stop.")
	}
}

// runServerWithGracefulShutdown runs a server with graceful shutdown handling.
func runServerWithGracefulShutdown(ctx context.Context, cfg *conf.GoInferCfg, e *echo.Echo, addr string) error {
	err := make(chan error, 1)
	go func() {
		err <- e.Start(addr)
	}()

	select {
	case er := <-err:
		return er
	case <-ctx.Done():
		return gracefulShutdownEchoServer(ctx, cfg, e, addr)
	}
}

// runProxyServerWithGracefulShutdown runs a proxy server with graceful shutdown handling.
func runProxyServerWithGracefulShutdown(ctx context.Context, cfg *conf.GoInferCfg, proxyServer *http.Server, proxyHandler http.Handler) error {
	err := make(chan error, 1)
	go func() {
		err <- proxyServer.ListenAndServe()
	}()

	select {
	case er := <-err:
		return er
	case <-ctx.Done():
		return gracefulShutdownProxyServer(ctx, cfg, proxyServer, proxyHandler)
	}
}

// gracefulShutdownEchoServer performs graceful shutdown of an Echo server.
func gracefulShutdownEchoServer(ctx context.Context, cfg *conf.GoInferCfg, e *echo.Echo, addr string) error {
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

// gracefulShutdownProxyServer performs graceful shutdown of a proxy server.
func gracefulShutdownProxyServer(ctx context.Context, cfg *conf.GoInferCfg, proxyServer *http.Server, proxyHandler http.Handler) error {
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
func handleShutdown(sigChan <-chan os.Signal, ctx context.Context, cancel context.CancelFunc) {
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
