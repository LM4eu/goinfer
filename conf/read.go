// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/llama-swap/proxy/config"
	"go.yaml.in/yaml/v4"
)

// ReadGoinferYML loads the configuration file, reads the env vars and verifies the settings.
func ReadGoinferYML(noAPIKey bool, extra, start string) (*Cfg, error) {
	yml, err := os.ReadFile(GoinferYML)
	if err != nil {
		return nil, gie.Wrap(err, gie.ConfigErr, "Cannot read", "file", GoinferYML)
	}
	return ReadYAMLData(yml, noAPIKey, extra, start)
}

// ReadYAMLData unmarshals the YAML bytes, applies the env vars and verifies the settings.
func ReadYAMLData(yml []byte, noAPIKey bool, extra, start string) (*Cfg, error) {
	cfg := defaultCfg
	err := cfg.parse(yml)
	cfg.applyEnvVars()

	if extra != "" {
		cfg.DefaultModel = "" // this forces DefaultModel to be the first of the ExtraModels
		cfg.parseExtraModels(extra)
	}

	if start != "" {
		// start Goinfer using the "start" model
		cfg.DefaultModel = start
	}

	cfg.trimParamValues()

	// concatenate host and ports => addr = "host:port"
	listen := make(map[string]string, len(cfg.Listen))
	for addr, services := range cfg.Listen {
		if addr == "" || addr[0] == ':' {
			addr = cfg.Host + addr
		}
		listen[addr] = services
	}
	cfg.Listen = listen

	// error from cfg.parse()
	if err != nil {
		return nil, err
	}

	return &cfg, cfg.validate(noAPIKey)
}

// ReadSwapFromReader uses the LoadConfigFromReader() from llama-swap project.
func (cfg *Cfg) ReadSwapFromReader(r io.Reader) error {
	var err error
	cfg.Swap, err = config.LoadConfigFromReader(r)
	if err != nil {
		slog.Error("Cannot load llama-swap config", "file", LlamaSwapYML, "error", err)
		os.Exit(1)
	}
	return cfg.ValidateSwap()
}

// load the configuration file (if filename not empty).
func (cfg *Cfg) parse(yml []byte) error {
	if len(yml) == 0 {
		return gie.New(gie.ConfigErr, "empty", "file", GoinferYML)
	}

	err := yaml.Unmarshal(yml, &cfg)
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "Failed to yaml.Unmarshal", "invalid YAML", yml)
	}

	return nil
}

// applyEnvVars read optional env vars to change the configuration.
// The environment variables precede the config file.
func (cfg *Cfg) applyEnvVars() {
	if dir := os.Getenv("GI_MODELS_DIR"); dir != "" {
		cfg.ModelsDir = dir
		slog.Debug("use", "GI_MODELS_DIR", dir)
	}

	if def := os.Getenv("GI_DEFAULT_MODEL"); def != "" {
		cfg.DefaultModel = def
		slog.Debug("use", "GI_DEFAULT_MODEL", def)
	}

	if extra, ok := syscall.Getenv("GI_EXTRA_MODELS"); ok {
		extra = strings.TrimSpace(extra)
		slog.Debug("use", "GI_EXTRA_MODELS", extra)
		cfg.parseExtraModels(extra)
	}

	if host := os.Getenv("GI_HOST"); host != "" {
		cfg.Host = host
		slog.Debug("use", "GI_HOST", host)
	}

	if origins := os.Getenv("GI_ORIGINS"); origins != "" {
		cfg.Origins = origins
		slog.Debug("use", "GI_ORIGINS", origins)
	}

	if key := os.Getenv("GI_API_KEY"); key != "" {
		cfg.APIKey = key
		slog.Debug("set api_key = GI_API_KEY")
	}

	if exe := os.Getenv("GI_LLAMA_EXE"); exe != "" {
		cfg.Llama.Exe = exe
		slog.Debug("use", "GI_LLAMA_EXE", exe)
	}

	// TODO add GI_LLAMA_ARGS_xxxxxx
}

func (cfg *Cfg) parseExtraModels(extra string) {
	// empty => disable goinfer.yml/extra_models
	if extra == "" {
		cfg.ExtraModels = nil
	} else if extra[0] == '=' { // starts with "=" => replace goinfer.yml/extra_models
		cfg.ExtraModels = nil
		extra = extra[1:] // skip first "="
	}

	for pair := range strings.SplitSeq(extra, "|||") {
		model_flags := strings.SplitN(pair, "=", 2)
		model := strings.TrimSpace(model_flags[0])
		cfg.ExtraModels[model] = ""
		if len(model_flags) > 1 {
			cfg.ExtraModels[model] = strings.TrimSpace(model_flags[1])
		}
		// if DefaultModel unset => use the first ExtraModels
		if cfg.DefaultModel == "" {
			cfg.DefaultModel = model
		}
	}
}

// trimParamValues cleans settings values.
func (cfg *Cfg) trimParamValues() {
	cfg.ModelsDir = strings.TrimSpace(cfg.ModelsDir)
	cfg.ModelsDir = strings.Trim(cfg.ModelsDir, ":")

	cfg.DefaultModel = strings.TrimSpace(cfg.DefaultModel)

	cfg.Host = strings.TrimSpace(cfg.Host)

	cfg.Origins = strings.TrimSpace(cfg.Origins)
	cfg.Origins = strings.Trim(cfg.Origins, ",")

	cfg.Llama.Exe = strings.TrimSpace(cfg.Llama.Exe)
	cfg.Llama.Args.Verbose = strings.TrimSpace(cfg.Llama.Args.Verbose)
	cfg.Llama.Args.Debug = strings.TrimSpace(cfg.Llama.Args.Debug)
	cfg.Llama.Args.Common = strings.TrimSpace(cfg.Llama.Args.Common)
	cfg.Llama.Args.Goinfer = strings.TrimSpace(cfg.Llama.Args.Goinfer)
}
