// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/LM4eu/goinfer/gie"
	"go.yaml.in/yaml/v4"
)

// ReadMainCfg the configuration file, then apply the env vars and finally verify the settings.
func (cfg *Cfg) ReadMainCfg(giCfg string, noAPIKey bool) error {
	err := cfg.load(giCfg)
	if err != nil {
		return err
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

// load the configuration file (if filename not empty).
func (cfg *Cfg) load(giCfg string) error {
	if giCfg == "" {
		return nil
	}
	yml, err := os.ReadFile(filepath.Clean(giCfg))
	if err != nil {
		slog.Error("Failed to read", "file", giCfg)
		return gie.Wrap(err, gie.TypeConfiguration, "", "")
	}

	if len(yml) > 0 {
		err := yaml.Unmarshal(yml, &cfg)
		if err != nil {
			slog.Error("Failed to yaml.Unmarshal", "100FirsBytes", string(yml[:100]))
			return gie.Wrap(err, gie.TypeConfiguration, "", "")
		}
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
		cfg.Server.Host = host
		slog.Debug("use", "GI_HOST", host)
	}

	if origins := os.Getenv("GI_ORIGINS"); origins != "" {
		cfg.Server.Origins = origins
		slog.Debug("use", "GI_ORIGINS", origins)
	}

	// Load user API key from environment
	if key := os.Getenv("GI_API_KEY_USER"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 2)
		}
		cfg.Server.APIKeys["user"] = key
		slog.Debug("set api_key[user] = GI_API_KEY_USER")
	}

	// Load admin API key from environment
	if key := os.Getenv("GI_API_KEY_ADMIN"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 1)
		}
		cfg.Server.APIKeys["admin"] = key
		slog.Debug("set api_key[admin] = GI_API_KEY_ADMIN")
	}

	if exe := os.Getenv("GI_LLAMA_EXE"); exe != "" {
		cfg.Llama.Exe = exe
		slog.Debug("use", "GI_LLAMA_EXE", exe)
	}
}
