// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"log/slog"
	"net"
	"os"
	"slices"
	"strings"
	"syscall"

	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/llama-swap/proxy/config"
	"go.yaml.in/yaml/v4"
)

type (
	Cfg struct {
		Server    ServerCfg     `json:"server"        yaml:"server"`
		Llama     LlamaCfg      `json:"llama"         yaml:"llama"`
		ModelsDir string        `json:"models_dir"    yaml:"models_dir"`
		Swap      config.Config `json:"swap,omitzero" yaml:"swap,omitempty"`
	}

	ServerCfg struct {
		Listen  map[string]string `json:"listen"           yaml:"listen"`
		APIKey  string            `json:"api_key"          yaml:"api_key"`
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
	// Hex code: C0ffee 15 C001, 50 C0ffee 15 900d. Bad C0de 15 Dead, 101. Cafe 91f7, 90 Cafe, 7e57 C0de.
	debugAPIKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de"
	unsetAPIKey = "Please ⚠️ Set your private 64-hex-digit API keys (32 bytes)"

	// Arguments for llama-server command line.
	argsCommon  = "--props --no-warmup"
	argsGoinfer = "--jinja --chat-template-file template.jinja"
)

var (
	defaultGoinferCfg = Cfg{
		ModelsDir: "/home/me/models",
		Server: ServerCfg{
			Listen: map[string]string{
				":4444": "infer",
				":5555": "llama-swap",
			},
			APIKey:  "",
			Host:    "",
			Origins: "localhost",
		},
		Llama: LlamaCfg{
			Exe: "/home/me/llama.cpp/build/bin/llama-server",
			Args: map[string]string{
				"common": argsCommon,
				"infer":  argsInfer,
			},
		},
	}

	// The Fetch standard defines the bad ports the browsers should block.
	// https://fetch.spec.whatwg.org/#port-blocking
	badPorts = []string{
		"0", "1", "7", "9", "11", "13", "15", "17", "19", "20", "21", "22", "23", "25",
		"37", "42", "43", "53", "69", "77", "79", "87", "95", "101", "102", "103", "104", "109", "110",
		"111", "113", "115", "117", "119", "123", "135", "137", "139", "143", "161", "179", "389", "427",
		"465", "512", "513", "514", "515", "526", "530", "531", "532", "540", "548", "554", "556", "563",
		"587", "601", "636", "989", "990", "993", "995", "1719", "1720", "1723", "2049", "3659", "4045",
		"4190", "5060", "5061", "6000", "6566", "6665", "6666", "6667", "6668", "6669", "6679", "6697", "10080",
	}
)

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
	default:
		fallthrough
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
	printEnvVar("GI_API_KEY", true)
	printEnvVar("GI_LLAMA_EXE", false)

	slog.Info("-----------------------------")

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		slog.Error("Failed to yaml.Marshal", "error", err.Error())
		return
	}

	_, _ = os.Stdout.Write(yml)

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
	err := cfg.validatePorts()
	if err != nil {
		return err
	}

	if noAPIKey {
		slog.Info("Flag -no-api-key => Do not verify API keys.")
		return nil
	}

	// Check API key
	if cfg.Server.APIKey == "" || strings.Contains(cfg.Server.APIKey, "Please") {
		return gie.New(gie.ConfigErr, "API key not set, please set your private API key")
		}
	if cfg.Server.APIKey == debugAPIKey {
		slog.Warn("API key is DEBUG => security threat")
	} else if len(cfg.Server.APIKey) < 64 {
		slog.Warn("API key should be 64+ hex digits", "len", len(cfg.Server.APIKey))
	}

	return nil
}

// The Fetch standard defines the bad ports the browsers should block.
// https://fetch.spec.whatwg.org/#port-blocking
//
//nolint:revive // for better readability => do not rewrite with `if !c { continue }`
func (cfg *Cfg) validatePorts() error {
	for hostPort := range cfg.Server.Listen {
		_, port, err := net.SplitHostPort(hostPort)
		if err != nil {
			slog.Error("Cannot split", "hostPort", hostPort, "err", err)
			return err
		}
		if slices.Contains(badPorts, port) {
			const msg = "Chrome/Firefox block the bad ports"
			slog.Error(msg, "port", port, "reference", "https://fetch.spec.whatwg.org/#port-blocking")
			return gie.New(gie.ConfigErr, msg, "port", port, "reference", "https://fetch.spec.whatwg.org/#port-blocking")
		}
	}
	return nil
}
