package main

import (
	"flag"
	"fmt"

	"github.com/teal-finance/garcon"

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

	server.RunServers(cfg)
}
