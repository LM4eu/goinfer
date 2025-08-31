// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/mostlygeek/llama-swap/proxy"
	"github.com/teal-finance/garcon"

	"github.com/LM4eu/goinfer/conf"
	"github.com/LM4eu/goinfer/server"
	"github.com/LM4eu/goinfer/state"
)

const (
	goinfCfgFile = "goinfer.yml"
	proxyCfgFile = "llama-swap.yml"
)

func main() {
	quiet := flag.Bool("q", false, "quiet mode (disable verbose output)")
	debug := flag.Bool("debug", false, "debug mode")
	genGiConf := flag.Bool("gen-gi-cfg", false, "generate "+goinfCfgFile)
	genPxConf := flag.Bool("gen-px-cfg", false, "generate "+proxyCfgFile+" (proxy config file)")
	noAPIKeys := flag.Bool("disable-api-key", false, "disable API key check")
	garcon.SetVersionFlag()
	flag.Parse()

	if *debug {
		fmt.Println("INFO: Debug mode is on")
		state.Debug = true
	}

	state.Verbose = !*quiet

	cfg := manageCfg(*debug, *genGiConf, *genPxConf, *noAPIKeys)

	runHTTPServers(cfg)
}

func manageCfg(debug, genGiConf, genPxConf, noAPIKeys bool) *conf.GoInferCfg {
	// Generate config
	if genGiConf {
		err := conf.Create(goinfCfgFile, debug)
		if err != nil {
			fmt.Printf("ERROR creating config: %v\n", err)
			os.Exit(1)
		}
		if state.Verbose {
			cfg, er := conf.Load(goinfCfgFile)
			if er != nil {
				fmt.Printf("ERROR loading config: %v\n", er)
				os.Exit(1)
			}
			cfg.Print()
		}
		os.Exit(0)
	}

	// Load configurations
	cfg, err := conf.Load(goinfCfgFile)
	if err != nil {
		fmt.Printf("ERROR loading config: %v\n", err)
		os.Exit(1)
	}
	cfg.Verbose = state.Verbose

	// Load the llama-swap config
	cfg.Proxy, err = proxy.LoadConfig(proxyCfgFile)
	// even if err!=nil => generate the config file,
	if genPxConf {
		err = conf.GenerateProxyCfg(cfg, proxyCfgFile)
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

	proxyServer, proxyHandler := server.NewProxyServer(cfg)

	// Setup signal handling with context
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start signal handling goroutine
	go func() {
		sig := <-sigChan
		fmt.Printf("INFO: Received signal %v, initiating graceful shutdown...\n", sig)

		// Cancel context to trigger shutdown
		cancel()

		// Wait for graceful shutdown completion or timeout
		select {
		case <-time.After(10 * time.Second):
			fmt.Println("WARNING: Graceful shutdown timed out, forcing exit")
			os.Exit(1)
		case <-ctx.Done():
			fmt.Println("INFO: Graceful shutdown completed")
		}
	}()

	var grp errgroup.Group

	// Start HTTP servers
	for addr, services := range cfg.Server.Listen {
		e := server.NewEchoServer(cfg, addr, services)
		if e != nil {
			if cfg.Verbose {
				fmt.Println("-----------------------------")
				fmt.Println("Starting Echo server:")
				fmt.Println("- services: ", services)
				fmt.Println("- listen:   ", addr)
				fmt.Println("- origins:  ", cfg.Server.Origins)
			}

			// Use the parent context for server shutdown
			grp.Go(func() error {
				// Start server in a goroutine
				serverErr := make(chan error, 1)
				go func() {
					serverErr <- e.Start(addr)
				}()

				// Wait for either server error or context cancellation
				select {
				case err := <-serverErr:
					return err
				case <-ctx.Done():
					if cfg.Verbose {
						fmt.Printf("INFO: Shutting down Echo server on %s\n", addr)
					}
					// Graceful shutdown of Echo server
					shutdownErr := e.Shutdown(ctx)
					if shutdownErr != nil {
						fmt.Printf("ERROR: Echo server shutdown error on %s: %v\n", addr, shutdownErr)
						return shutdownErr
					}
					if cfg.Verbose {
						fmt.Printf("INFO: Echo server on %s stopped gracefully\n", addr)
					}
					return nil
				}
			})
		}
	}

	if proxyServer != nil {
		if cfg.Verbose {
			fmt.Println("-----------------------------")
			fmt.Println("Starting Gin server:")
			fmt.Println("- services: llama-swap proxy")
			fmt.Println("- listen:   ", proxyServer.Addr)
		}

		// Use the parent context for proxy server shutdown
		grp.Go(func() error {
			// Start proxy server in a goroutine
			proxyErr := make(chan error, 1)
			go func() {
				proxyErr <- proxyServer.ListenAndServe()
			}()

			// Wait for either proxy server error or context cancellation
			select {
			case err := <-proxyErr:
				return err
			case <-ctx.Done():
				if cfg.Verbose {
					fmt.Printf("INFO: Shutting down proxy server on %s\n", proxyServer.Addr)
				}
				// Graceful shutdown of proxy server
				if proxyHandler != nil {
					proxyHandler.Shutdown()
				}
				shutdownErr := proxyServer.Shutdown(ctx)
				if shutdownErr != nil {
					fmt.Printf("ERROR: Proxy server shutdown error: %v\n", shutdownErr)
					return shutdownErr
				}
				if cfg.Verbose {
					fmt.Printf("INFO: Proxy server on %s stopped gracefully\n", proxyServer.Addr)
				}
				return nil
			}
		})
	}

	if cfg.Verbose {
		fmt.Println("-----------------------------")
		fmt.Println("INFO: All servers started. Press CTRL+C to stop.")
	}

	// Wait for all servers to complete
	err := grp.Wait()
	if err != nil {
		fmt.Printf("ERROR: Server error: %v\n", err)
	} else {
		fmt.Println("INFO: All HTTP servers stopped gracefully")
	}
}
