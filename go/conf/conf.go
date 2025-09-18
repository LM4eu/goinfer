// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/gie"
	"github.com/mostlygeek/llama-swap/proxy"
	"go.yaml.in/yaml/v4"
)

type (
	GoInferCfg struct {
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

var defaultGoInferCfg = GoInferCfg{
	ModelsDir: "/home/me/my/models",
	Server: ServerCfg{
		Listen: map[string]string{
			":5143": "admin,models",
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

// Read the configuration file.
func (cfg *GoInferCfg) Read(goinferCfgFile string, noAPIKey bool) error {
	// Load from file if specified
	if goinferCfgFile != "" {
		yml, err := os.ReadFile(filepath.Clean(goinferCfgFile))
		if err != nil {
			return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_FILE_READ_FAILED", "failed to read "+goinferCfgFile)
		}

		if len(yml) > 0 {
			err := yaml.Unmarshal(yml, &cfg)
			if err != nil {
				return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_UNMARSHAL_FAILED", "failed to unmarshal YAML data: "+string(yml[:100]))
			}
		}
	}

	cfg.applyEnvVars()

	// Concatenate host and ports => addr = "host:port"
	listen := make(map[string]string, len(cfg.Server.Listen))
	for addr, services := range cfg.Server.Listen {
		if addr == "" || addr[0] == ':' {
			addr = cfg.Server.Host + addr
		}
		listen[addr] = services
	}
	cfg.Server.Listen = listen

	// Validate configuration
	return cfg.validate(noAPIKey)
}

// Write populates the configuration with defaults, applies environment variables,
// writes the resulting configuration to the given file, and mutates the receiver.
func (cfg *GoInferCfg) Write(goinferCfgFile string, noAPIKey bool) error {
	cfg.Llama = defaultGoInferCfg.Llama
	cfg.ModelsDir = defaultGoInferCfg.ModelsDir
	cfg.Server = defaultGoInferCfg.Server

	cfg.applyEnvVars()

	// Set API keys
	switch {
	case noAPIKey:
		slog.InfoContext(context.Background(), "Flag -no-api-key => Do not generate API keys", "file", goinferCfgFile)

	case len(cfg.Server.APIKeys) > 0:
		slog.InfoContext(context.Background(), "Configuration file uses API keys from environment", "file", goinferCfgFile)

	default:
		cfg.Server.APIKeys["admin"] = genAPIKey(cfg.Debug)
		cfg.Server.APIKeys["user"] = genAPIKey(cfg.Debug)
		if cfg.Debug {
			slog.WarnContext(context.Background(), "Configuration file with DEBUG API key (not suitable for production)", "file", goinferCfgFile)
		} else {
			slog.InfoContext(context.Background(), "Configuration file with secure API keys", "file", goinferCfgFile)
		}
	}

	// The following temporary `Verbose`/`Debug` flag toggling block is necessary
	// to avoid polluting the configuration with: verbose=false debug=true.
	// Better to use command line flags: -q -debug.
	// Users are free to add manually verbose=false debug=true in the configuration.
	vrb := cfg.Verbose
	dbg := cfg.Debug
	cfg.Verbose = false
	cfg.Debug = false

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_MARSHAL", "failed to write config file")
	}

	cfg.Verbose = vrb
	cfg.Debug = dbg

	err = os.WriteFile(goinferCfgFile, yml, 0o600)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_WRITE_FAILED", "failed to write config file")
	}

	return cfg.validate(noAPIKey)
}

// GenerateProxyCfg generates the llama-swap-proxy configuration.
func (cfg *GoInferCfg) GenProxyCfg(proxyCfgFile string) error {
	modelFiles, err := cfg.Search()
	if err != nil {
		return err
	}

	if len(modelFiles) == 0 {
		return err
	}

	for _, model := range modelFiles {
		base := filepath.Base(model)
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)

		flags := extractFlags(model)

		// OpenAI API
		if cfg.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				slog.InfoContext(context.Background(), "Overwrite model", "model", stem, "file", proxyCfgFile)
			}
		}
		cfg.Proxy.Models[stem] = proxy.ModelConfig{
			Cmd:          "${llama-server-openai} -m " + model + " " + flags,
			Unlisted:     false,
			UseModelName: stem,
		}

		// GoInfer API: hide the model + prefix GI_
		prefixedModelName := "GI_" + stem
		if cfg.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				slog.InfoContext(context.Background(), "Overwrite model", "model", stem, "file", proxyCfgFile)
			}
		}
		cfg.Proxy.Models[prefixedModelName] = proxy.ModelConfig{
			Cmd:          "${llama-server-goinfer} -m " + model + " " + flags,
			Unlisted:     true,
			UseModelName: prefixedModelName,
		}
	}

	yml, er := yaml.Marshal(&cfg.Proxy)
	if er != nil {
		return gie.Wrap(er, gie.TypeConfiguration, "CONFIG_MARSHAL_FAILED", "failed to marshal the llama-swap-proxy config")
	}

	err = os.WriteFile(proxyCfgFile, yml, 0o600)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "PROXY_WRITE_FAILED", "failed to write "+proxyCfgFile)
	}

	if cfg.Verbose {
		slog.InfoContext(context.Background(), "Generated proxy config", "file", proxyCfgFile, "models", len(modelFiles))
	}

	return nil
}

// Print configuration.
func (cfg *GoInferCfg) Print() {
	slog.InfoContext(context.Background(), "-----------------------------")
	slog.InfoContext(context.Background(), "Environment Variables:")
	slog.InfoContext(context.Background(), "GI_MODELS_DIR", "value", os.Getenv("GI_MODELS_DIR"))
	slog.InfoContext(context.Background(), "GI_HOST", "value", os.Getenv("GI_HOST"))
	slog.InfoContext(context.Background(), "GI_ORIGINS", "value", os.Getenv("GI_ORIGINS"))
	slog.InfoContext(context.Background(), "GI_API_KEY_ADMIN length", "len", len(os.Getenv("GI_API_KEY_ADMIN")))
	slog.InfoContext(context.Background(), "GI_API_KEY_USER length", "len", len(os.Getenv("GI_API_KEY_USER")))
	slog.InfoContext(context.Background(), "GI_LLAMA_EXE", "value", os.Getenv("GI_LLAMA_EXE"))

	slog.InfoContext(context.Background(), "-----------------------------")

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		slog.ErrorContext(context.Background(), "yaml.Marshal error", "error", err.Error())
		return
	}

	os.Stdout.Write(yml)

	slog.InfoContext(context.Background(), "-----------------------------")
}

// applyEnvVars read optional env vars to change the configuration.
// The environment variables precede the config file.
func (cfg *GoInferCfg) applyEnvVars() {
	// Load environment variables
	if dir := os.Getenv("GI_MODELS_DIR"); dir != "" {
		cfg.ModelsDir = dir
		if cfg.Verbose {
			slog.InfoContext(context.Background(), "GI_MODELS_DIR set", "value", dir)
		}
	}

	if host := os.Getenv("GI_HOST"); host != "" {
		cfg.Server.Host = host
		if cfg.Verbose {
			slog.InfoContext(context.Background(), "GI_HOST set", "value", host)
		}
	}

	if origins := os.Getenv("GI_ORIGINS"); origins != "" {
		cfg.Server.Origins = origins
		if cfg.Verbose {
			slog.InfoContext(context.Background(), "GI_ORIGINS set", "value", origins)
		}
	}

	// Load user API key from environment
	if key := os.Getenv("GI_API_KEY_USER"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 2)
		}
		cfg.Server.APIKeys["user"] = key
		if cfg.Verbose {
			slog.InfoContext(context.Background(), "api_key[user] = GI_API_KEY_USER")
		}
	}

	// Load admin API key from environment
	if key := os.Getenv("GI_API_KEY_ADMIN"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 1)
		}
		cfg.Server.APIKeys["admin"] = key
		if cfg.Verbose {
			slog.InfoContext(context.Background(), "api_key[admin] = GI_API_KEY_ADMIN")
		}
	}

	if exe := os.Getenv("GI_LLAMA_EXE"); exe != "" {
		cfg.Llama.Exe = exe
		if cfg.Verbose {
			slog.InfoContext(context.Background(), "GI_LLAMA_EXE", "value", exe)
		}
	}
}

func genAPIKey(debugMode bool) string {
	if debugMode {
		return debugAPIKey
	}

	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		slog.WarnContext(context.Background(), "rand.Read error", "error", err)
		return ""
	}

	key := make([]byte, 64)
	hex.Encode(key, buf)
	return string(key)
}

func (cfg *GoInferCfg) validate(noAPIKey bool) error {
	modelFiles, err := cfg.Search()
	if err != nil {
		return err
	}
	if len(modelFiles) == 0 {
		slog.WarnContext(context.Background(), "No *.gguf files found", "dir", cfg.ModelsDir)
	} else if cfg.Verbose {
		slog.InfoContext(context.Background(), "Found model files", "count", len(modelFiles), "dir", cfg.ModelsDir)
	}

	// Ensure admin API key exists
	if _, exists := cfg.Server.APIKeys["admin"]; !exists {
		return gie.Wrap(gie.ErrAPIKeyMissing, gie.TypeConfiguration, "ADMIN_API_MISSING", "admin API key is missing")
	}

	if noAPIKey {
		slog.InfoContext(context.Background(), "Flag -no-api-key => Do not verify API keys.")
		return nil
	}

	// Validate API keys
	for k, v := range cfg.Server.APIKeys {
		if strings.Contains(v, "PLEASE") {
			return gie.Wrap(gie.ErrInvalidAPIKey, gie.TypeConfiguration, "API_KEY_NOT_SET", "please set your private '"+k+"' API key")
		}
		if len(v) < 64 {
			return gie.Wrap(gie.ErrInvalidAPIKey, gie.TypeConfiguration, "API_KEY_INVALID", "invalid API key '"+k+"': must be 64 hex digits")
		}
		if v == debugAPIKey {
			slog.WarnContext(context.Background(), "api_key DEBUG security threat", "key", k)
		}
	}

	return nil
}
