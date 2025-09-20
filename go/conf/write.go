// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
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

// WriteMainCfg populates the configuration with defaults, applies environment variables,
// writes the resulting configuration to the given file, and mutates the receiver.
func (cfg *Cfg) WriteMainCfg(giCfg string, noAPIKey bool) error {
	cfg.Llama = defaultGoInferCfg.Llama
	cfg.ModelsDir = defaultGoInferCfg.ModelsDir
	cfg.Server = defaultGoInferCfg.Server

	cfg.applyEnvVars()

	cfg.setAPIKeys(noAPIKey)

	// The following `Verbose`/`Debug` toggling is necessary to avoid
	// polluting the configuration with: verbose=false debug=true.
	// Better to use command line flags: -q -debug.
	// Users are free to add manually verbose=false debug=true in the configuration.
	vrb, dbg := cfg.Verbose, cfg.Debug
	cfg.Verbose, cfg.Debug = false, false

	yml, err := yaml.Marshal(&cfg)

	cfg.Verbose, cfg.Debug = vrb, dbg

	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_MARSHAL", "failed to write config file")
	}

	err = writeWithHeader(giCfg, "# Configuration of https://github.com/LM4eu/goinfer", yml)
	if err != nil {
		return err
	}

	return cfg.validate(noAPIKey)
}

// WriteProxyCfg generates the llama-swap-proxy configuration.
func (cfg *Cfg) WriteProxyCfg(pxCfg string) error {
	modelFiles, err := cfg.Search()
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
		cfg.setModelSettings(model)
	}

	yml, er := yaml.Marshal(&cfg.Proxy)
	if er != nil {
		return gie.Wrap(er, gie.TypeConfiguration, "CONFIG_MARSHAL_FAILED", "failed to marshal the llama-swap-proxy config")
	}

	err = writeWithHeader(pxCfg, "# Doc: https://github.com/mostlygeek/llama-swap/wiki/Configuration", yml)
	if err != nil {
		return err
	}

	return nil
}

// Set the settings of a model within the llama-swap-proxy configuration.
func (cfg *Cfg) setModelSettings(model string) {
	base := filepath.Base(model)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	flags := extractFlags(model)

	// OpenAI API
	if cfg.Verbose {
		_, ok := cfg.Proxy.Models[stem]
		if ok {
			slog.Info("Overwrite config", "model", stem)
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
			slog.Info("Overwrite config", "model", stem)
		}
	}
	cfg.Proxy.Models[prefixedModelName] = proxy.ModelConfig{
		Cmd:          "${llama-server-goinfer} -m " + model + " " + flags,
		Unlisted:     true,
		UseModelName: prefixedModelName,
	}
}

func (cfg *Cfg) setAPIKeys(noAPIKey bool) {
	if len(cfg.Server.APIKeys) > 0 {
		slog.Info("Configuration file uses API keys from environment")
		return
	}

	cfg.Server.APIKeys = make(map[string]string, 2)

	switch {
	case noAPIKey:
		cfg.Server.APIKeys["admin"] = unsetAPIKey
		cfg.Server.APIKeys["user"] = unsetAPIKey
		slog.Info("Flag -no-api-key => Do not generate API keys")

	case cfg.Debug:
		cfg.Server.APIKeys["admin"] = debugAPIKey
		cfg.Server.APIKeys["user"] = debugAPIKey
		slog.Warn("API keys are DEBUG => security threat")

	default:
		cfg.Server.APIKeys["admin"] = gen64HexDigits()
		cfg.Server.APIKeys["user"] = gen64HexDigits()
		slog.Info("Generated random secured API keys")
	}
}

func gen64HexDigits() string {
	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		slog.Warn("Failed to rand.Read", "error", err)
		return ""
	}

	key := make([]byte, 64)
	hex.Encode(key, buf)
	return string(key)
}

func writeWithHeader(path, header string, data []byte) error {
	path = filepath.Clean(path)
	file, err := os.Create(path)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_WRITE_FAILED", "failed to create file")
	}

	_, err = file.WriteString(header + "\n\n")
	if err == nil {
		_, err = file.Write(data)
	}

	er := file.Close()
	if err != nil {
		err = er
	}
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_WRITE_FAILED", "failed to write file")
	}

	return nil
}
