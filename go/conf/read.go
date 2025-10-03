// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/gie"
	"go.yaml.in/yaml/v4"
)

// ReadMainCfg the configuration file, then apply the env vars and finally verify the settings.
func (cfg *Cfg) ReadMainCfg(mainCfg string, noAPIKey bool) error {
	err := cfg.load(mainCfg)
	if err != nil {
		return err
	}

	cfg.applyEnvVars()
	cfg.trimParamValues()

	// Concatenate host and ports => addr = "host:port"
	listen := make(map[string]string, len(cfg.Listen))
	for addr, services := range cfg.Listen {
		if addr == "" || addr[0] == ':' {
			addr = cfg.Host + addr
		}
		listen[addr] = services
	}
	cfg.Listen = listen

	// Validate configuration
	return cfg.validateMain(noAPIKey)
}

// load the configuration file (if filename not empty).
func (cfg *Cfg) load(mainCfg string) error {
	if mainCfg == "" {
		return nil
	}

	yml, err := os.ReadFile(filepath.Clean(mainCfg))
	if err != nil {
		slog.Error("Failed to read", "file", mainCfg)
		return gie.Wrap(err, gie.ConfigErr, "os.ReadFile", "file", mainCfg)
	}

	if len(yml) == 0 {
		return gie.Wrap(err, gie.ConfigErr, "empty", "file", mainCfg)
	}

	err = yaml.Unmarshal(yml, &cfg)
	if err != nil {
		slog.Error("Failed to yaml.Unmarshal", "100FirsBytes", string(yml[:100]))
		return gie.Wrap(err, gie.ConfigErr, "yaml.Unmarshal")
	}

	return nil
}

// applyEnvVars read optional env vars to change the configuration.
// The environment variables precede the config file.
func (cfg *Cfg) applyEnvVars() {
	// Load environment variables
	if dir := os.Getenv("GI_MODELS_DIR"); dir != "" {
		cfg.ModelsDir = dir
		slog.Debug("use", "GI_MODELS_DIR", dir)
	}

	if host := os.Getenv("GI_HOST"); host != "" {
		cfg.Host = host
		slog.Debug("use", "GI_HOST", host)
	}

	if origins := os.Getenv("GI_ORIGINS"); origins != "" {
		cfg.Origins = origins
		slog.Debug("use", "GI_ORIGINS", origins)
	}

	// Load API key from environment
	if key := os.Getenv("GI_API_KEY"); key != "" {
		cfg.APIKey = key
		slog.Debug("set api_key = GI_API_KEY")
	}

	if exe := os.Getenv("GI_LLAMA_EXE"); exe != "" {
		cfg.Llama.Exe = exe
		slog.Debug("use", "GI_LLAMA_EXE", exe)
	}
}

// trimParamValues cleans each parameter.
func (cfg *Cfg) trimParamValues() {
	cfg.ModelsDir = strings.TrimSpace(cfg.ModelsDir)
	cfg.ModelsDir = strings.Trim(cfg.ModelsDir, ":")

	cfg.Host = strings.TrimSpace(cfg.Host)

	cfg.Origins = strings.TrimSpace(cfg.Origins)
	cfg.Origins = strings.Trim(cfg.Origins, ",")

	cfg.Llama.Exe = strings.TrimSpace(cfg.Llama.Exe)
}
