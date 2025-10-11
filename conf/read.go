// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/LM4eu/goinfer/gie"
	"go.yaml.in/yaml/v4"
)

// ReadMain the configuration file, then apply the env vars and finally verify the settings.
func (cfg *Cfg) ReadMain(mainCfg string, noAPIKey bool, extra, start string) error {
	err := cfg.load(mainCfg)
	cfg.applyEnvVars()

	if extra != "" {
		// force DefaultModel to be the first of the ExtraModels
		cfg.DefaultModel = ""
		cfg.parseExtraModels(extra)
	}

	if start != "" {
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

	// error from cfg.load()
	if err != nil {
		return err
	}

	// Validate configuration
	return cfg.validate(noAPIKey)
}

// load the configuration file (if filename not empty).
func (cfg *Cfg) load(mainCfg string) error {
	if mainCfg == "" {
		return nil
	}

	yml, err := os.ReadFile(filepath.Clean(mainCfg))
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "Cannot read", "file", mainCfg)
	}

	if len(yml) == 0 {
		return gie.Wrap(err, gie.ConfigErr, "empty", "file", mainCfg)
	}

	err = yaml.Unmarshal(yml, &cfg)
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
