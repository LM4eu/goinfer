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
	local := flag.Bool("local", false, "run in local mode with a gui (default is api mode: no gui and no websockets, api key required)")
	genConf := flag.Bool("conf", false, "generate a config file (export MODELS_DIR=/home/me/my/models)")
	genLocalConf := flag.Bool("localconf", false, "generate a config file for local mode usage (export MODELS_DIR=/home/me/my/models)")
	disableApiKey := flag.Bool("disable-api-key", false, "disable the api key")
	garcon.SetVersionFlag()
	flag.Parse()

	if *debug {
		fmt.Println("Debug mode is on")
		state.IsDebug = true
	}

	// Fix: Correct the logic for verbose mode
	state.IsVerbose = !*quiet

	if *genLocalConf || *genConf {
		if err := conf.Create("goinfer.yml", *genLocalConf); err != nil {
			panic(err)
		}
		if *genLocalConf {
			fmt.Println("File goinfer.yml created with default API key")
		} else {
			fmt.Println("File goinfer.yml created with random API key")
		}
		return
	}

	cfg, err := conf.Load("goinfer.yml")
	if err != nil {
		panic(err)
	}

	if state.IsDebug {
		if err := cfg.Debug(); err != nil {
			panic(err)
		}
	}

	if state.IsVerbose {
		fmt.Println("Starting the http server with allowed origins", cfg.Server.Origins)
	}

	server.RunServer(cfg.Server, *local, *disableApiKey)

}
