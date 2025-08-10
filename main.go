package main

import (
	"flag"
	"fmt"

	"github.com/synw/goinfer/conf"
	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/server"
	"github.com/synw/goinfer/state"
)

func main() {
	quiet := flag.Bool("q", false, "disable the verbose output")
	debug := flag.Bool("debug", false, "debug mode")
	local := flag.Bool("local", false, "run in local mode with a gui (default is api mode: no gui and no websockets, api key required)")
	genConfModelsDir := flag.String("conf", "", "generate a config file. Provide a models directory absolute path as argument")
	genLocalConfModelsDir := flag.String("localconf", "", "generate a config file for local mode usage. Provide a models directory absolute path as argument")
	disableApiKey := flag.Bool("disable-api-key", false, "disable the api key")
	flag.Parse()

	if *debug {
		fmt.Println("Debug mode is on")
		state.IsDebug = true
	}

	if !*quiet {
		state.IsVerbose = *quiet
	}

	if len(*genConfModelsDir) > 0 {
		if err := conf.Create(*genConfModelsDir, false, "goinfer.yml"); err != nil {
			panic(err)
		}
		fmt.Println("File goinfer.yml created with random API key")
		return
	}

	if len(*genLocalConfModelsDir) > 0 {
		if err := conf.Create(*genLocalConfModelsDir, true, "goinfer.yml"); err != nil {
			panic(err)
		}
		fmt.Println("File goinfer.yml created with default API key")
		return
	}

	conf, err := conf.InitConf(".", "goinfer") // ./goinfer.json or goinfer.yml ...
	if err != nil {
		panic(err)
	}

	// Initializes the Llama server manager.
	state.Llama = llama.NewLlamaServerManager(&conf.Llama)
	state.Monitor = llama.NewMonitor(&conf.Llama)
	state.Monitor.Start()
	defer state.StopLlamaServer()

	state.ModelsDir = conf.ModelsDir
	state.IsVerbose = !*quiet

	if state.IsVerbose {
		fmt.Println("Starting the http server with allowed origins", conf.WebServer.Origins)
	}

	server.RunServer(conf.WebServer, *local, *disableApiKey)
}
