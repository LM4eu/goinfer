package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/teal-finance/garcon"
	"golang.org/x/sync/errgroup"

	"github.com/synw/goinfer/conf"
	"github.com/synw/goinfer/server"
	"github.com/synw/goinfer/state"
)

func main() {
	quiet := flag.Bool("q", false, "disable the verbose output")
	debug := flag.Bool("debug", false, "debug mode")
	genConf := flag.Bool("conf", false, "generate a config file (export MODELS_DIR=/home/me/my/models)")
	disableApiKeys := flag.Bool("disable-api-key", false, "http server will not check the api key")
	garcon.SetVersionFlag()
	flag.Parse()

	if *debug {
		fmt.Println("Debug mode is on")
		state.Debug = true
	}

	// Fix: Correct the logic for verbose mode
	state.Verbose = !*quiet

	if *genConf {
		conf.Create("goinfer.yml", *debug)
		return
	}

	cfg := conf.Load("goinfer.yml", "llama-swap.yml")
	cfg.Verbose = state.Verbose

	if *disableApiKeys {
		cfg.Server.ApiKeys = nil
	}

	if state.Debug {
		cfg.Print()
	}

	proxyServer, proxyHandler := server.NewProxyServer(cfg)

	// Setup channels for server management
	exitChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// shutdown on signal
	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal %v, shutting down...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		if proxyServer != nil {
			proxyHandler.Shutdown()
			if err := proxyServer.Shutdown(ctx); err != nil {
				fmt.Printf("Server shutdown error: %v\n", err)
			}
		}

		close(exitChan)
	}()

	var g errgroup.Group

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

	// Wait for exit signal
	<-exitChan

	err := g.Wait()
	if err != nil {
		fmt.Printf("ERROR http server %v\n", err)
	} else {
		fmt.Println("All http servers have stoped")
	}
}
