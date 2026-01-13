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

	"github.com/lynxai-team/garcon/vv"
	"github.com/lynxai-team/goinfer/proxy"

	"github.com/lynxai-team/goinfer/conf"
)

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
	debug := flag.Bool("debug", false, "debug mode (set debug ABI keys in "+conf.GoinferINI+" with -overwrite-all)")
	writeAll := flag.Bool("overwrite-all", false, "write config files: "+conf.GoinferINI+" "+conf.TemplateJinja+" "+conf.ModelsINI)
	writeSwap := flag.Bool("write-swap", false, "write the llama-swap config file "+conf.LlamaSwapYML+" (this file is not used by goinfer)")
	updateModelsINI := flag.Bool("update-models-ini", false, "write only the llama.cpp config file "+conf.ModelsINI+" (do not modify "+conf.GoinferINI+" )")
	run := flag.Bool("run", false, "run the server, to be used with -update-models-ini (or -overwrite-all)")
	extra := flag.String("hf", "", "configure the given extra_models and load the first one (start llama-server)")
	start := flag.String("start", "", "set the default_model and load it (start llama-server)")
	noAPIKey := flag.Bool("no-api-key", false, "disable API key check (set a warning-fake API key in "+conf.GoinferINI+" with -overwrite-all)")
	vv.SetVersionFlag()
	flag.Parse()

	verbose := !*quiet

	if *extra != "" || *start != "" { // -hf and -start implies -run
		*run = true
	}
	if *writeAll {
		*updateModelsINI = true
	}

	switch {
	case *debug:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case verbose:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	default:
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}

	cfg := doGoinferINI(*debug, *writeAll, *run, *noAPIKey, *extra, *start)

	if *writeAll || verbose {
		cfg.Print()
	}

	doLlamaSwapYML(cfg, *writeSwap, verbose, *debug)

	if *updateModelsINI {
		doModelsINI(cfg)
	}

	if *writeAll && !*run {
		slog.Info("flag -overwrite-all without any -run -hf -start => stop here")
		return nil
	}
	if *updateModelsINI && !*run {
		slog.Info("flag -update-models-ini without any -run -hf -start => stop here")
		return nil
	}
	if *writeSwap && !*run {
		slog.Info("flag -write-swap without any -run -hf -start => stop here")
		return nil
	}

	return cfg
}

func doGoinferINI(debug, writeAll, run, noAPIKey bool, extra, start string) *conf.Cfg {
	slog.Info("Read", "file", conf.GoinferINI)
	cfg, err := conf.ReadGoinferINI(noAPIKey, extra, start)
	if err != nil {
		switch {
		case writeAll:
			slog.Info("Write a fresh new config file, may contain issues: "+
				"please verify the config.", "file", conf.GoinferINI, "err", err)
		case run:
			slog.Error("Error config. To auto-fix it use the flag -overwrite-all "+
				"(note: flags -run -start -hf preserve the config)", "file", conf.GoinferINI, "err", err)
			os.Exit(1)
		default:
			slog.Info("Error config => Write a new one using default values, env vars and flags. "+
				"May contain issues: please verify the config.", "file", conf.GoinferINI, "err", err)
		}
		writeAll = true
	}

	if writeAll {
		slog.Info("Write", "file", conf.TemplateJinja)
		err = os.WriteFile(conf.TemplateJinja, []byte("{{- messages[0].content -}}"), 0o600)
		if err != nil {
			slog.Error("Cannot write", "file", conf.TemplateJinja, "err", err)
		}

		wrote, er := cfg.WriteGoinferINI(debug, noAPIKey)
		if er != nil {
			err = er
			slog.Info(err.Error())
		} else if wrote {
			slog.Info("Wrote", "file", conf.GoinferINI)
		} else {
			slog.Info("Do not write because content unchanged", "file", conf.GoinferINI)
			return cfg
		}

		// read "goinfer.ini" to verify it can be successfully loaded
		// Pass empty extra and start to keep the eventual fixes.
		slog.Info("Verify the written config by reading/parsing it", "file", conf.GoinferINI)
		cfg, er = conf.ReadGoinferINI(noAPIKey, "", "")
		if er != nil {
			slog.Warn("Please review", "file", conf.GoinferINI, "err", er)
			os.Exit(1)
		}

		// stop if any error from WriteFile(conf.TemplateJinja) or WriteGoinferINI
		if err != nil {
			slog.Info("Please review env. vars and", "file", conf.GoinferINI)
			os.Exit(1)
		}

		slog.Info("Successfully wrote", "file", conf.GoinferINI)
	}

	// command line precedes config file
	if noAPIKey {
		cfg.APIKey = ""
	}

	return cfg
}

func doLlamaSwapYML(cfg *conf.Cfg, writeSwap, verbose, debug bool) {
	yml, err := cfg.GenLlamaSwapYAML(verbose, debug)
	if err != nil {
		slog.Error("Failed generating a valid llama-swap config", "err", err)
		os.Exit(1)
	}

	if writeSwap {
		wrote, err := conf.WriteLlamaSwapYML(yml)
		if err != nil {
			slog.Warn("Failed writing the llama-swap config", "file", conf.LlamaSwapYML, "err", err)
		} else if wrote {
			slog.Info("Wrote llama-swap config", "file", conf.LlamaSwapYML, "models", len(cfg.Swap.Models))
		} else {
			slog.Info("Do not write llama-swap config because unchanged content", "file", conf.LlamaSwapYML, "models", len(cfg.Swap.Models))
		}
	}

	reader := bytes.NewReader(yml)
	err = cfg.ReadSwapFromReader(reader)
	if err != nil {
		slog.Error("Invalid llama-swap config. Use flag -write-swap to investigate", "file", conf.LlamaSwapYML, "err", err)
		os.Exit(1)
	}
}

func doModelsINI(cfg *conf.Cfg) {
	ini := cfg.GenModelsINI()
	wrote, err := conf.WriteModelsINI(ini)
	if err != nil {
		slog.Warn("Failed writing the llama.cpp config", "file", conf.ModelsINI, "err", err)
	} else if wrote {
		slog.Info("Wrote llama.cpp config", "file", conf.ModelsINI, "presets", len(cfg.Info))
	} else {
		slog.Info("Unchanged content => do not write llama.cpp config", "file", conf.ModelsINI, "presets", len(cfg.Info))
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
