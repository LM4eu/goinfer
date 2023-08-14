package main

import (
	"flag"
	"fmt"

	"github.com/synw/goinfer/conf"
	"github.com/synw/goinfer/server"
	"github.com/synw/goinfer/state"
)

func main() {
	quiet := flag.Bool("q", false, "disable the verbose output")
	local := flag.Bool("local", false, "run in local mode with a gui (default is api mode: no gui and no websockets, api key required)")
	var genConfModelsDir = flag.String("conf", "", "generate a config file. Provide a models directory absolute path as argument")
	flag.Parse()

	if !*quiet {
		state.IsVerbose = *quiet
	}
	if len(*genConfModelsDir) > 0 {
		conf.Create(*genConfModelsDir)
		fmt.Println("File goinfer.config.json created")
		return
	}

	conf := conf.InitConf()
	state.ModelsDir = conf.ModelsDir
	state.TasksDir = conf.TasksDir
	state.OpenAiConf = conf.OpenAiConf
	state.IsVerbose = !*quiet
	if state.IsVerbose {
		fmt.Println("Starting the http server with allowed origins", conf.Origins)
	}
	server.RunServer(conf.Origins, conf.ApiKey, *local, conf.OpenAiConf.Enable)
}
