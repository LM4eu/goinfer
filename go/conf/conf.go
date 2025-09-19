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
	"strconv"
	"strings"
	"syscall"

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

// Read the configuration file, then apply the env vars and finally verify the settings.
func (cfg *GoInferCfg) Read(ctx context.Context, goinferCfgFile string, noAPIKey bool) error {
	err := cfg.load(ctx, goinferCfgFile)
	if err != nil {
		return err
	}

	cfg.applyEnvVars(ctx)

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
	return cfg.validate(ctx, noAPIKey)
}

// Write populates the configuration with defaults, applies environment variables,
// writes the resulting configuration to the given file, and mutates the receiver.
func (cfg *GoInferCfg) Write(ctx context.Context, goinferCfgFile string, noAPIKey bool) error {
	cfg.Llama = defaultGoInferCfg.Llama
	cfg.ModelsDir = defaultGoInferCfg.ModelsDir
	cfg.Server = defaultGoInferCfg.Server

	cfg.applyEnvVars(ctx)

	// Set API keys
	switch {
	case noAPIKey:
		slog.InfoContext(ctx, "Flag -no-api-key => Do not generate API keys", "file", goinferCfgFile)

	case len(cfg.Server.APIKeys) > 0:
		slog.InfoContext(ctx, "Configuration file uses API keys from environment", "file", goinferCfgFile)

	default:
		cfg.Server.APIKeys["admin"] = genAPIKey(ctx, cfg.Debug)
		cfg.Server.APIKeys["user"] = genAPIKey(ctx, cfg.Debug)
		if cfg.Debug {
			slog.WarnContext(ctx, "Configuration file with DEBUG API key (not suitable for production)", "file", goinferCfgFile)
		} else {
			slog.InfoContext(ctx, "Configuration file with secure API keys", "file", goinferCfgFile)
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

	goinferCfgFile = filepath.Clean(goinferCfgFile)
	file, err := os.Create(goinferCfgFile)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_WRITE_FAILED", "failed to open config file")
	}
	defer file.Close()

	_, err = file.WriteString("# Configuration of https://github.com/LM4eu/goinfer\n\n")
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_WRITE_FAILED", "failed to write config file")
	}

	_, err = file.Write(yml)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_WRITE_FAILED", "failed to write config file")
	}

	if cfg.Verbose {
		slog.InfoContext(ctx, "Generated main config", "file", goinferCfgFile)
	}

	return cfg.validate(ctx, noAPIKey)
}

// GenerateProxyCfg generates the llama-swap-proxy configuration.
func (cfg *GoInferCfg) GenProxyCfg(ctx context.Context, proxyCfgFile string) error {
	modelFiles, err := cfg.Search(ctx)
	if err != nil {
		return err
	}

	if len(modelFiles) == 0 {
		return err
	}

	if cfg.Proxy.Models == nil {
		cfg.Proxy.Models = make(map[string]proxy.ModelConfig, 2*len(modelFiles))
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
				slog.InfoContext(ctx, "Overwrite model", "model", stem, "file", proxyCfgFile)
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
				slog.InfoContext(ctx, "Overwrite model", "model", stem, "file", proxyCfgFile)
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

	proxyCfgFile = filepath.Clean(proxyCfgFile)
	file, err := os.Create(proxyCfgFile)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "PROXY_WRITE_FAILED", "failed to open "+proxyCfgFile)
	}
	defer file.Close()

	_, err = file.WriteString("# Doc: https://github.com/mostlygeek/llama-swap/wiki/Configuration\n\n")
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "PROXY_WRITE_FAILED", "failed to write "+proxyCfgFile)
	}

	_, err = file.Write(yml)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "PROXY_WRITE_FAILED", "failed to write "+proxyCfgFile)
	}

	if cfg.Verbose {
		slog.InfoContext(ctx, "Generated proxy config", "file", proxyCfgFile, "models", len(modelFiles))
	}

	return nil
}

func printEnvVar(ctx context.Context, key string, confidential bool) {
	v, set := syscall.Getenv(key)
	switch {
	case !set:
		slog.InfoContext(ctx, "envvar", key, "(unset)")
	case v == "":
		slog.InfoContext(ctx, "envvar", key, "(empty)")
	case confidential:
		slog.InfoContext(ctx, "envvar", key+"-length", len(v))
	default:
		slog.InfoContext(ctx, "envvar", key, v)
	}
}

// Print configuration.
func (cfg *GoInferCfg) Print(ctx context.Context) {
	slog.InfoContext(ctx, "-----------------------------")

	printEnvVar(ctx, "GI_MODELS_DIR", false)
	printEnvVar(ctx, "GI_HOST", false)
	printEnvVar(ctx, "GI_ORIGINS", false)
	printEnvVar(ctx, "GI_API_KEY_ADMIN", true)
	printEnvVar(ctx, "GI_API_KEY_USER", true)
	printEnvVar(ctx, "GI_LLAMA_EXE", false)

	slog.InfoContext(ctx, "-----------------------------")

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		slog.ErrorContext(ctx, "yaml.Marshal error", "error", err.Error())
		return
	}

	os.Stdout.Write(yml)

	slog.InfoContext(ctx, "-----------------------------")
}

// load the configuration file (if filename not empty).
func (cfg *GoInferCfg) load(ctx context.Context, goinferCfgFile string) error {
	if goinferCfgFile == "" {
		return nil
	}
	yml, err := os.ReadFile(filepath.Clean(goinferCfgFile))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to read", "file", goinferCfgFile)
		return gie.Wrap(err, gie.TypeConfiguration, "", "")
	}

	// command line parameters have precedence on config settings
	dbg := cfg.Debug
	vrb := cfg.Verbose

	if len(yml) > 0 {
		err := yaml.Unmarshal(yml, &cfg)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to unmarshal YAML data", "100FirsBytes", string(yml[:100]))
			return gie.Wrap(err, gie.TypeConfiguration, "", "")
		}
	}

	if dbg {
		cfg.Debug = dbg
	}
	if !vrb {
		cfg.Verbose = vrb
	}

	return nil
}

// applyEnvVars read optional env vars to change the configuration.
// The environment variables precede the config file.
func (cfg *GoInferCfg) applyEnvVars(ctx context.Context) {
	// Load environment variables
	if dir := os.Getenv("GI_MODELS_DIR"); dir != "" {
		cfg.ModelsDir = dir
		if cfg.Verbose {
			slog.InfoContext(ctx, "GI_MODELS_DIR set", "value", dir)
		}
	}

	if host := os.Getenv("GI_HOST"); host != "" {
		cfg.Server.Host = host
		if cfg.Verbose {
			slog.InfoContext(ctx, "GI_HOST set", "value", host)
		}
	}

	if origins := os.Getenv("GI_ORIGINS"); origins != "" {
		cfg.Server.Origins = origins
		if cfg.Verbose {
			slog.InfoContext(ctx, "GI_ORIGINS set", "value", origins)
		}
	}

	// Load user API key from environment
	if key := os.Getenv("GI_API_KEY_USER"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 2)
		}
		cfg.Server.APIKeys["user"] = key
		if cfg.Verbose {
			slog.InfoContext(ctx, "api_key[user] = GI_API_KEY_USER")
		}
	}

	// Load admin API key from environment
	if key := os.Getenv("GI_API_KEY_ADMIN"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 1)
		}
		cfg.Server.APIKeys["admin"] = key
		if cfg.Verbose {
			slog.InfoContext(ctx, "api_key[admin] = GI_API_KEY_ADMIN")
		}
	}

	if exe := os.Getenv("GI_LLAMA_EXE"); exe != "" {
		cfg.Llama.Exe = exe
		if cfg.Verbose {
			slog.InfoContext(ctx, "GI_LLAMA_EXE", "value", exe)
		}
	}
}

func genAPIKey(ctx context.Context, debugMode bool) string {
	if debugMode {
		return debugAPIKey
	}

	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		slog.WarnContext(ctx, "rand.Read error", "error", err)
		return ""
	}

	key := make([]byte, 64)
	hex.Encode(key, buf)
	return string(key)
}

func (cfg *GoInferCfg) validate(ctx context.Context, noAPIKey bool) error {
	modelFiles, err := cfg.Search(ctx)
	if err != nil {
		return err
	}
	if len(modelFiles) == 0 {
		slog.WarnContext(ctx, "No *.gguf files found", "dir", cfg.ModelsDir)
	} else if cfg.Verbose {
		slog.InfoContext(ctx, "Found model files", "count", len(modelFiles), "dir", cfg.ModelsDir)
	}

	if noAPIKey {
		slog.InfoContext(ctx, "Flag -no-api-key => Do not verify API keys.")
		return nil
	}

	// Ensure admin API key exists
	if _, exists := cfg.Server.APIKeys["admin"]; !exists {
		return gie.Wrap(gie.ErrAPIKeyMissing, gie.TypeConfiguration, "ADMIN_API_MISSING", "admin API key is missing")
	}

	// Validate API keys
	for k, v := range cfg.Server.APIKeys {
		if len(v) < 64 {
			return gie.Wrap(gie.ErrInvalidAPIKey, gie.TypeConfiguration, "API_KEY_INVALID", "API key must be 64 hex digits key="+k+"len="+strconv.Itoa(len(v)))
		}
		if v == debugAPIKey {
			slog.WarnContext(ctx, "API key is DEBUG => security threat", "key", k)
		}
	}

	return nil
}
