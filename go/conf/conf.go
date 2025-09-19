// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/LM4eu/goinfer/gie"
	"github.com/mostlygeek/llama-swap/proxy"
	"go.yaml.in/yaml/v4"
)

type (
	Cfg struct {
		Server    ServerCfg    `json:"server"           yaml:"server"`
		Llama     LlamaCfg     `json:"llama"            yaml:"llama"`
		ModelsDir string       `json:"models_dir"       yaml:"models_dir"`
		Proxy     proxy.Config `json:"proxy,omitzero"   yaml:"proxy,omitempty"`
		Verbose   bool         `json:"verbose,omitzero" yaml:"verbose,omitempty"`
		Debug     bool         `json:"debug,omitzero"   yaml:"debug,omitempty"`
	}

	ServerCfg struct {
		Listen  map[string]string `json:"listen"           yaml:"listen"`
		APIKeys map[string]string `json:"api_key"          yaml:"api_key"`
		Host    string            `json:"host,omitzero"    yaml:"host,omitempty"`
		Origins string            `json:"origins,omitzero" yaml:"origins,omitempty"`
	}

	LlamaCfg struct {
		Args map[string]string `json:"args" yaml:"args"`
		Exe  string            `json:"exe"  yaml:"exe"`
	}
)

const debugAPIKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"

var defaultGoInferCfg = Cfg{
	ModelsDir: "/home/me/my/models",
	Server: ServerCfg{
		Listen: map[string]string{
			":5143": "webui,models",
			":2222": "openai,goinfer",
			":5555": "llama-swap proxy",
		},
		APIKeys: map[string]string{},
		Host:    "",
		Origins: "localhost",
	},
	Llama: LlamaCfg{
		Exe: "/home/me/llama.cpp/build/bin/llama-server",
		Args: map[string]string{
			"common":  "--no-webui --no-warmup",
			"goinfer": "--jinja --chat-template-file template.jinja",
		},
	},
}

// Print configuration.
func (cfg *Cfg) Print() {
	slog.Info("-----------------------------")

	printEnvVar("GI_MODELS_DIR", false)
	printEnvVar("GI_HOST", false)
	printEnvVar("GI_ORIGINS", false)
	printEnvVar("GI_API_KEY_ADMIN", true)
	printEnvVar("GI_API_KEY_USER", true)
	printEnvVar("GI_LLAMA_EXE", false)

	slog.Info("-----------------------------")

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		slog.Error("Failed to yaml.Marshal", "error", err.Error())
		return
	}

	os.Stdout.Write(yml)

	slog.Info("-----------------------------")
}

func printEnvVar(key string, confidential bool) {
	v, set := syscall.Getenv(key)
	switch {
	case !set:
		slog.Info("env", key, "(unset)")
	case v == "":
		slog.Info("env", key, "(empty)")
	case confidential:
		slog.Info("env", key+"-length", len(v))
	default:
		slog.Info("env", key, v)
	}
}

func (cfg *Cfg) validate(noAPIKey bool) error {
	if len(cfg.Proxy.Models) == 0 {
		n := cfg.countModels()
		if n == 0 {
			slog.Warn("No *.gguf files found", "dir", cfg.ModelsDir)
		} else {
			slog.Warn("No model configured => Use flag -gen-px-cfg to fill the config with", "files", n)
		}
	}

	for i := range cfg.Proxy.Models {
		var previous string
		for arg := range strings.SplitSeq(cfg.Proxy.Models[i].Cmd, " ") {
			if previous == "-m" {
				// Step 1: Check if the file exists
				info, err := os.Stat(arg)
				if os.IsNotExist(err) {
					slog.Error("Model file does not exist", "file", arg)
					return err
				}

				// Step 2: Check if the file is readable
				file, err := os.Open(arg)
				if err != nil {
					slog.Error("Model file is not readable", "file", arg)
					return err
				}
				defer file.Close()

				// Step 3: Check if the file is not empty
				if info.Size() < 1000 {
					slog.Error("Model file is empty (or too small)", "file", arg)
					return gie.ErrConfigValidation
				}
			}
			previous = arg
		}
	}

	if noAPIKey {
		slog.Info("Flag -no-api-key => Do not verify API keys.")
		return nil
	}

	// Ensure admin API key exists
	if _, exists := cfg.Server.APIKeys["admin"]; !exists {
		slog.Error("Admin API key is missing")
		return gie.ErrAPIKeyMissing
	}

	// Check API keys
	for k, v := range cfg.Server.APIKeys {
		if v == debugAPIKey {
			slog.Warn("API key is DEBUG => security threat", "key", k)
		} else if len(v) < 64 {
			slog.Warn("API key should be 64+ hex digits", "key", k, "len", len(v))
		}
	}

	return nil
}
