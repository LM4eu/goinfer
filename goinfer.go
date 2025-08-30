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

func main() {
	quiet := flag.Bool("q", false, "disable verbose output")
	debug := flag.Bool("debug", false, "debug mode")
	genGiConf := flag.Bool("gen-gi-conf", false, "generate goinfer.yml")
	genPxConf := flag.Bool("gen-px-conf", false, "generate llama-swap.yml (proxy config file)")
	noAPIKeys := flag.Bool("disable-api-key", false, "disable API key check")
	garcon.SetVersionFlag()
	flag.Parse()

	if *debug {
		fmt.Println("INFO: Debug mode is on")
		state.Debug = true
	}

	state.Verbose = !*quiet

	// Generate config
	if *genGiConf {
		err := conf.Create("goinfer.yml", *debug)
		if err != nil {
			fmt.Printf("ERROR creating config: %v\n", err)
			os.Exit(1)
		}
		if state.Verbose {
			cfg, er := conf.Load("goinfer.yml")
			if er != nil {
				fmt.Printf("ERROR loading config: %v\n", er)
				os.Exit(1)
			}
			cfg.Print()
		}
		return
	}

	// Load configurations
	cfg, err := conf.Load("goinfer.yml")
	if err != nil {
		fmt.Printf("ERROR loading config: %v\n", err)
		os.Exit(1)
	}
	cfg.Verbose = state.Verbose

	// Load the llama-swap config
	cfg.Proxy, err = proxy.LoadConfig("llama-swap.yml")
	// even if err!=nil => generate the config file,
	if *genPxConf {
		err = conf.GenerateProxyCfg(cfg, "llama-swap.yml")
		if err != nil {
			fmt.Printf("ERROR generating proxy config: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err != nil {
		fmt.Printf("ERROR loading proxy config: %v\n", err)
		os.Exit(1)
	}

	if *noAPIKeys {
		cfg.Server.APIKeys = nil
	}

	if state.Debug {
		cfg.Print()
	}

	proxyServer, proxyHandler := server.NewProxyServer(cfg)

	// Setup signal handling
	exitChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// shutdown on signal
	go func() {
		sig := <-sigChan
		fmt.Printf("INFO: Received signal %v, shutting down...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if proxyServer != nil {
			proxyHandler.Shutdown()
			err = proxyServer.Shutdown(ctx)
			if err != nil {
				fmt.Printf("ERROR Server shutdown: %v\n", err)
			}
		}

		close(exitChan)
	}()

	var g errgroup.Group

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
			g.Go(func() error { return e.Start(addr) })
		}
	}

	if proxyServer != nil {
		if cfg.Verbose {
			fmt.Println("-----------------------------")
			fmt.Println("Starting Gin server:")
			fmt.Println("- services: llama-swap proxy")
			fmt.Println("- listen:   ", proxyServer.Addr)
		}
		g.Go(func() error { return proxyServer.ListenAndServe() })
	}

	if cfg.Verbose {
		fmt.Println("-----------------------------")
	}

	// Wait for exit signal
	<-exitChan
	err = g.Wait()
	if err != nil {
		fmt.Printf("ERROR HTTT server: %v\n", err)
	} else {
		fmt.Println("INFO: All HTTP servers stopped")
	}
}
