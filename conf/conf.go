// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package conf reads/writes configuration
package conf

import (
	"errors"
	"io/fs"
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
	// Cfg holds all settings.
	Cfg struct {
		APIKey       string               `yaml:"api_key"             toml:"api_key"             comment:"⚠️ Set your API key, can be 64-hex-digit (32-byte) 🚨\nGoinfer sets a random API key with: ./goinfer -write"`
		Host         string               `yaml:"host,omitempty"      toml:"host,omitempty"      comment:"\nHost to listen (env. var: GI_HOST)"`
		Origins      string               `yaml:"origins"             toml:"origins"             comment:"\nCORS whitelist (env. var: GI_ORIGINS)"`
		ModelsDir    string               `yaml:"models_dir"          toml:"models_dir"          comment:"\nGoinfer recursively searches GGUF files in one or multiple folders separated by ':'\nList your GGUF dirs with: locate .gguf | sed -e 's,/[^/]*$,,' | uniq\nenv. var: GI_MODELS_DIR"`
		DefaultModel string               `yaml:"default_model"       toml:"default_model"       comment:"\nThe default model name to load at startup\nCan also be set with: ./goinfer -start <model-name>"`
		ExtraModels  map[string]string    `yaml:"extra_models"        toml:"extra_models"        comment:"List model names and their llama-server flags"`
		Llama        Llama                `yaml:"llama"               toml:"llama"`
		Listen       map[string]string    `yaml:"listen"              toml:"listen"              comment:"Addresses (ports) to listen\nAddress can be <ip|host>:<port> or simply :<port> when <host> is localhost"`
		Templates    map[string]string    `yaml:"templates,omitempty" toml:"templates,omitempty" comment:"Provide a template file for each model (not yet fully implemented)"`
		Info         map[string]ModelInfo `yaml:"-"                   toml:"-"                                                                                                                                                                 `
		Swap         config.Config        `yaml:"-"                   toml:"-"                                                                                                                                                                 `
		// Swap is stored in llama-swap.yml
	}

	// Llama holds the inference engine settings.
	Llama struct {
		Exe     string `yaml:"exe"     toml:"exe"     comment:"path of llama-server"`
		Common  string `yaml:"common"  toml:"common"  comment:"common args used for every model"`
		Goinfer string `yaml:"goinfer" toml:"goinfer" comment:"extra args to let tools like Agent-Smith doing the templating (/completions endpoint)"`
		Verbose string `yaml:"verbose" toml:"verbose" comment:"extra llama-server flag when ./goinfer is used without the -q flag"`
		Debug   string `yaml:"debug"   toml:"debug"   comment:"extra llama-server flag for ./goinfer -debug"`
	}
)

const (
	// Sentence: Coffee is cool, so coffee is good. Bad code is dead, lol. Cafe gift, go cafe, test code.
	// Hex code: C0ffee 15 C001, 50 C0ffee 15 900d. Bad C0de 15 Dead, 101. Cafe 91f7, 90 Cafe, 7e57 C0de.
	debugAPIKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de"
	unsetAPIKey = "Please ⚠️ Set your private 64-hex-digit API key (32 bytes)"

	// GoinferINI is the config filename.
	GoinferINI = "goinfer.ini"
	// LlamaSwapYML is the llama-swap config filename.
	LlamaSwapYML = "llama-swap.yml"
)

// Do not use the bad ports: they are blocked by web browsers,
// as specified by the Fetch standard: fetch.spec.whatwg.org/#port-blocking.
var badPorts = []string{
	"0", "1", "7", "9", "11", "13", "15", "17", "19", "20", "21", "22", "23", "25", "37", "42", "43",
	"53", "69", "77", "79", "87", "95", "101", "102", "103", "104", "109", "110", "111", "113",
	"115", "117", "119", "123", "135", "137", "139", "143", "161", "179", "389", "427", "465",
	"512", "513", "514", "515", "526", "530", "531", "532", "540", "548", "554", "556", "563", "587",
	"601", "636", "989", "990", "993", "995", "1719", "1720", "1723", "2049", "3659", "4045", "4190",
	"5060", "5061", "6000", "6566", "6665", "6666", "6667", "6668", "6669", "6679", "6697", "10080",
}

// defaultCfg returns a unique copy of its local variable
// to each receiver preventing data race (concurrency testing).
func defaultCfg() *Cfg {
	return &Cfg{
		ModelsDir:    "/home/me/path/to/models",
		DefaultModel: "",
		APIKey:       "",
		Host:         "",
		Origins:      "localhost",
		Listen: map[string]string{
			":4444": "goinfer",
			":5555": "llama-swap",
		},
		Llama: Llama{
			Exe:     "/home/me/llama.cpp/build/bin/llama-server",
			Verbose: "--verbose-prompt",
			Debug:   "--verbosity 3",
			Common:  "--props --no-warmup --no-mmap",
			Goinfer: "--jinja --chat-template-file template.jinja",
		},
		ExtraModels: map[string]string{ // Output of `llama-server -h` contains:
			// github.com/ggml-org/llama.cpp/blob/master/common/arg.cpp#L3857
			"ggml-org/bge-small-en-v1.5-Q8_0-GGUF":            "--embd-bge-small-en-default",
			"ggml-org/e5-small-v2-Q8_0-GGUF":                  "--embd-e5-small-en-default",
			"ggml-org/gte-small-Q8_0-GGUF":                    "--embd-gte-small-default",
			"ggml-org/Qwen2.5-Coder-1.5B-Q8_0-GGUF":           "--fim-qwen-1.5b-default",
			"ggml-org/Qwen2.5-Coder-3B-Q8_0-GGUF":             "--fim-qwen-3b-default",
			"ggml-org/Qwen2.5-Coder-7B-Q8_0-GGUF":             "--fim-qwen-7b-default",
			"ggml-org/Qwen2.5-Coder-7B-Q8_0-GGUF+0.5B-draft":  "--fim-qwen-7b-spec",
			"ggml-org/Qwen2.5-Coder-14B-Q8_0-GGUF+0.5B-draft": "--fim-qwen-14b-spec",
			"ggml-org/Qwen3-Coder-30B-A3B-Instruct-Q8_0-GGUF": "--fim-qwen-30b-default",
		},
	}
}

// Print configuration.
func (cfg *Cfg) Print() {
	slog.Info("-----------------------------")

	printEnvVar("GI_MODELS_DIR", false)
	printEnvVar("GI_EXTRA_MODELS", false)
	printEnvVar("GI_DEFAULT_MODEL", false)
	printEnvVar("GI_HOST", false)
	printEnvVar("GI_ORIGINS", false)
	printEnvVar("GI_API_KEY", true)
	printEnvVar("GI_LLAMA_EXE", false)

	slog.Info("-----------------------------")

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		slog.Error("Failed yaml.Marshal", "error", err.Error(), "input struct", &cfg)
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

func (cfg *Cfg) validate(noAPIKey bool) error {
	err := cfg.validatePorts()
	if err != nil {
		return err
	}

	// GI_MODELS_DIR
	for dir := range strings.SplitSeq(cfg.ModelsDir, ":") {
		info, er := os.Stat(dir)
		if errors.Is(er, fs.ErrNotExist) {
			return gie.New(gie.ConfigErr, "Verify GI_MODELS_DIR or 'models_dir' in "+GoinferINI, "does not exist", dir)
		}
		if er != nil {
			return gie.Wrap(er, gie.ConfigErr, "Verify GI_MODELS_DIR or 'models_dir' in "+GoinferINI, "problem with", dir)
		}
		if !info.IsDir() {
			return gie.New(gie.ConfigErr, "Verify GI_MODELS_DIR or 'models_dir' in "+GoinferINI, "must be a directory", dir)
		}
	}

	// GI_LLAMA_EXE
	info, err := os.Stat(cfg.Llama.Exe)
	if errors.Is(err, fs.ErrNotExist) {
		return gie.New(gie.ConfigErr, "GI_LLAMA_EXE or 'exe' in goinfer.ini: file does not exist", "exe", cfg.Llama.Exe)
	}
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "GI_MODELS_DIR or 'models_dir' in goinfer.ini", "exe", cfg.Llama.Exe)
	}
	if info.IsDir() {
		return gie.New(gie.ConfigErr, "GI_LLAMA_EXE or 'exe' in goinfer.ini: must be a file, not a directory", "exe", cfg.Llama.Exe)
	}

	// API key
	if noAPIKey {
		slog.Info("Flag -no-api-key => Do not verify API key.")
		return nil
	}
	if cfg.APIKey == "" || strings.Contains(cfg.APIKey, "Please") {
		return gie.New(gie.ConfigErr, "API key not set, please set your private API key")
	}
	if cfg.APIKey == debugAPIKey {
		slog.Warn("API key is DEBUG => security threat")
	} else if len(cfg.APIKey) < 64 {
		slog.Warn("API key should be 64+ hex digits", "len", len(cfg.APIKey))
	}
	return nil
}

// validatePorts() prevents bad ports: they are blocked by web browsers,
// as specified by the Fetch standard: http://fetch.spec.whatwg.org/#bad-port
func (cfg *Cfg) validatePorts() error {
	for hostPort := range cfg.Listen {
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
