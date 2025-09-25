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
		Server    ServerCfg    `json:"server"        yaml:"server"`
		Llama     LlamaCfg     `json:"llama"         yaml:"llama"`
		ModelsDir string       `json:"models_dir"    yaml:"models_dir"`
		Swap      proxy.Config `json:"swap,omitzero" yaml:"swap,omitempty"`
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

const (
	// Sentence: Coffee is cool, so coffee is good. Bad code is dead, lol. Cafe gift, go cafe, test code.
	// Hex code: C0ffee 15 C001, 50 C0ffee 15 900d. Bad C0de 15 Dead, 101. Cafe 91f7, 90 Cafe, 7e57C0de.
	debugAPIKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de"
	unsetAPIKey = "PLEASE ⚠️ Set your private 64-hex-digit API keys (32 bytes) 🚨"

	// Arguments for llama-server command line.
	argsCommon  = "--props --no-warmup"
	argsGoinfer = "--jinja --chat-template-file template.jinja"
)

var defaultGoInferCfg = Cfg{
	ModelsDir: "/home/me/models",
	Server: ServerCfg{
		Listen: map[string]string{
			":2222": "webui models openai goinfer",
			":5555": "llama-swap",
		},
		APIKeys: map[string]string{},
		Host:    "",
		Origins: "localhost",
	},
	Llama: LlamaCfg{
		Exe: "/home/me/llama.cpp/build/bin/llama-server",
		Args: map[string]string{
			"common":  argsCommon,
			"goinfer": argsGoinfer,
		},
	},
}

func (cfg *Cfg) SetLogLevel(verbose, debug bool) {
	switch {
	case debug:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case verbose:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case cfg.Swap.LogLevel == "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}
}

func (cfg *Cfg) SetLogSwap(verbose, debug bool) {
	switch {
	case debug:
		cfg.Swap.LogLevel = "debug"
	case verbose:
		cfg.Swap.LogLevel = "info"
	case cfg.Swap.LogLevel != "error":
		cfg.Swap.LogLevel = "warn"
	}
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

func (cfg *Cfg) validateMain(noAPIKey bool) error {
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
		if strings.Contains(v, "PLEASE") {
			slog.Error("Please set your private", "key", k)
			return gie.Wrap(gie.ErrInvalidAPIKey, gie.TypeConfiguration, "API_KEY_NOT_SET", "please set your private '"+k+"' API key")
		}

		if v == debugAPIKey {
			slog.Warn("API key is DEBUG => security threat", "key", k)
		} else if len(v) < 64 {
			slog.Warn("API key should be 64+ hex digits", "key", k, "len", len(v))
		}
	}

	return nil
}
