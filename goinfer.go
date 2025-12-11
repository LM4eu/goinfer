// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/LM4eu/garcon/vv"
	"github.com/LM4eu/goinfer/proxy"

	"github.com/LM4eu/goinfer/conf"
)

const templateJinja = "template.jinja"

func main() {
	cfg := getCfg()
	if cfg != nil {
		startServer(cfg)
	}
}

// Config is created from lower to higher priority: (1) config files, (2) env. Vars. And (3) flags.
// Depending on the flags, this function also creates config files and exits.
func getCfg() *conf.Cfg {
	quiet := flag.Bool("q", false, "quiet mode (disable verbose output)")
	debug := flag.Bool("debug", false, "debug mode (with -write: set debug ABI keys)")
	write := flag.Bool("write", false, "write config files: "+conf.GoinferINI+" "+templateJinja+" "+conf.LlamaSwapYML)
	run := flag.Bool("run", false, "run the server, can be combined with -write")
	extra := flag.String("hf", "", "configure the given extra_models and load the first one (start llama-server)")
	start := flag.String("start", "", "set the default_model and load it (start llama-server)")
	noAPIKey := flag.Bool("no-api-key", false, "disable API key check (with -write: set a warning in place of the API key)")
	vv.SetVersionFlag()
	flag.Parse()

	verbose := !*quiet

	if *extra != "" || *start != "" { // -hf and -start implies -run
		*run = true
	}

	switch {
	case *debug:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case verbose:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	default:
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}
	slog.Debug("debug mode")

	cfg := doGoinferINI(*debug, *write, *run, *noAPIKey, *extra, *start)

	if *write || verbose {
		cfg.Print()
	}

	doLlamaSwapYML(cfg, *write, verbose, *debug)

	if *write && !*run {
		slog.Info("flag -write without any -run -hf -start => stop here")
		return nil
	}
	return cfg
}

func doGoinferINI(debug, write, run, noAPIKey bool, extra, start string) *conf.Cfg {
	cfg, err := conf.ReadGoinferINI(noAPIKey, extra, start)
	if err != nil {
		switch {
		case write:
			slog.Info("Write a fresh new config file, may contain issues: "+
				"please verify the config.", "file", conf.GoinferINI, "error", err)
		case run:
			slog.Error("Error config. Flags -run -start -hf without -write prevent to write it. "+
				"Add flag -write to write the config", "file", conf.GoinferINI, "error", err)
			os.Exit(1)
		default:
			slog.Info("Error config => Write a new one using default values, env vars and flags. "+
				"May contain issues: please verify the config.", "file", conf.GoinferINI, "error", err)
		}
		write = true
	}

	if write {
		err = os.WriteFile(templateJinja, []byte("{{- messages[0].content -}}"), 0o600)
		if err != nil {
			slog.Error("Cannot write", "file", templateJinja, "error", err)
		}

		er := cfg.WriteGoinferINI(debug, noAPIKey)
		if er != nil {
			slog.Info(er.Error())
			err = er
		}

		// read "goinfer.ini" to verify it can be successfully loaded
		// Pass empty extra and start to keep the eventual fixes.
		cfg, er = conf.ReadGoinferINI(noAPIKey, "", "")
		if er != nil {
			slog.Warn("Please review "+conf.GoinferINI, "error", er)
			os.Exit(1)
		}

		// stop if any error from WriteFile(templateJinja) or WriteGoinferINI
		if err != nil {
			slog.Info("Please review the env. vars and " + conf.GoinferINI)
			os.Exit(1)
		}

		slog.Info("Successfully wrote " + conf.GoinferINI)
	}

	// command line precedes config file
	if noAPIKey {
		cfg.APIKey = ""
	}

	return cfg
}

func doLlamaSwapYML(cfg *conf.Cfg, write, verbose, debug bool) {
	yml, err := cfg.GenSwapYAMLData(verbose, debug)
	if err != nil {
		slog.Error("Failed creating a valid llama-swap config", "file", conf.LlamaSwapYML, "error", err)
		os.Exit(1)
	}

	if write {
		err = conf.WriteLlamaSwapYML(yml)
		if err != nil {
			slog.Warn("Failed writing the llama-swap config", "file", conf.LlamaSwapYML, "error", err)
		}
		slog.Info("Generated llama-swap config", "file", conf.LlamaSwapYML, "models", len(cfg.Swap.Models))
	}

	reader := bytes.NewReader(yml)
	err = cfg.ReadSwapFromReader(reader)
	if err != nil {
		slog.Error("Invalid llama-swap config. Use flag -write to check", "file", conf.LlamaSwapYML, "error", err)
		os.Exit(1)
	}
}

// startServer creates and runs the HTTP server (API).
func startServer(cfg *conf.Cfg) {
	proxyMan := proxy.New(cfg)
	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: proxyMan,
	}

	slog.Info("-------------------------------------------")
	slog.Info("Starting HTTP server", "url", url(server.Addr), "origins", cfg.Origins)
	slog.Info("CTRL+C to stop")
	err := server.ListenAndServe()
	slog.Info("Server stop", "err", err)
}

func url(addr string) string {
	if addr != "" && addr[0] == ':' {
		return "http://localhost" + addr
	}
	return "http://" + addr
}
