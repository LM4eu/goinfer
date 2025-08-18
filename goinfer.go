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

	"github.com/synw/goinfer/conf"
	"github.com/synw/goinfer/server"
	"github.com/synw/goinfer/state"
)

func main() {
	quiet := flag.Bool("q", false, "disable the verbose output")
	debug := flag.Bool("debug", false, "debug mode")
	genGiConf := flag.Bool("gen-gi-conf", false, "generate the goinfer config file (use: MODELS_DIR=/home/me/my/models)")
	genPxConf := flag.Bool("gen-px-conf", false, "generate the llama-swap proxy config file")
	disableApiKeys := flag.Bool("disable-api-key", false, "http server will not check the api key")
	garcon.SetVersionFlag()
	flag.Parse()

	if *debug {
		fmt.Println("Debug mode is on")
		state.Debug = true
	}

	state.Verbose = !*quiet

	if *genGiConf {
		conf.Create("goinfer.yml", *debug)
		if state.Verbose {
			cfg := conf.Load("goinfer.yml")
			cfg.Print()
		}
		return
	}

	cfg := conf.Load("goinfer.yml")
	cfg.Verbose = state.Verbose

	// Load the llama-swap config
	var err error
	cfg.Proxy, err = proxy.LoadConfig("llama-swap.yml")
	if *genPxConf {
		conf.GenProxyConfFromModelFiles(&cfg, "llama-swap.yml")
		return
	}
	if err != nil {
		panic(fmt.Errorf("error LoadConfig(llama-swap.yml) %w", err))
	}

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

	err = g.Wait()
	if err != nil {
		fmt.Printf("ERROR http server %v\n", err)
	} else {
		fmt.Println("All http servers have stoped")
	}
}
