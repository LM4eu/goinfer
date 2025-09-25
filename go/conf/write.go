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

	"github.com/LM4eu/goinfer/gie"
	"github.com/mostlygeek/llama-swap/proxy"
	"go.yaml.in/yaml/v4"
)

// WriteMainCfg populates the configuration with defaults, applies environment variables,
// writes the resulting configuration to the given file, and mutates the receiver.
func (cfg *Cfg) WriteMainCfg(mainCfg string, debug, noAPIKey bool) error {
	cfg.Llama = defaultGoInferCfg.Llama
	cfg.ModelsDir = defaultGoInferCfg.ModelsDir
	cfg.Server = defaultGoInferCfg.Server

	cfg.applyEnvVars()
	cfg.trimParamValues()

	cfg.setAPIKeys(debug, noAPIKey)

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_MARSHAL", "failed to write config file")
	}

	err = writeWithHeader(mainCfg, "# Configuration of https://github.com/LM4eu/goinfer", yml)
	if err != nil {
		return err
	}

	return cfg.validate(noAPIKey)
}

// WriteSwapCfg generates the llama-swap configuration.
func (cfg *Cfg) WriteSwapCfg(swapCfg string, verbose, debug bool) error {
	switch {
	case debug:
		cfg.Swap.LogLevel = "debug"
	case verbose:
		cfg.Swap.LogLevel = "info"
	default:
		cfg.Swap.LogLevel = "warn"
	}

	cfg.Swap.StartPort = 5800         // first ${PORT} incremented for each model
	cfg.Swap.HealthCheckTimeout = 120 // seconds to wait for a model to become ready
	cfg.Swap.MetricsMaxInMemory = 500 // maximum number of metrics to keep in memory

	common, ok := cfg.Llama.Args["common"]
	if !ok {
		common = argsCommon
	}

	goinfer, ok := cfg.Llama.Args["goinfer"]
	if !ok {
		goinfer = argsGoinfer
	}

	cmd := cfg.Llama.Exe + " --port ${PORT} " + common

	cfg.Swap.Macros = map[string]string{
		"cmd-openai":  cmd,
		"cmd-goinfer": cmd + " " + goinfer,
	}

	info, err := cfg.search()
	if err != nil {
		return err
	}

	if cfg.Swap.Models == nil {
		cfg.Swap.Models = make(map[string]proxy.ModelConfig, 2*len(info))
	}

	for name, mi := range info {
		cfg.setModelSettings(name, mi.Path, mi.Flags, false) // OpenAI
		cfg.setModelSettings(name, mi.Path, mi.Flags, true)  // Goinfer
	}

	yml, er := yaml.Marshal(&cfg.Swap)
	if er != nil {
		return gie.Wrap(er, gie.TypeConfiguration, "CONFIG_MARSHAL_FAILED", "failed to marshal the llama-swap config")
	}

	err = writeWithHeader(swapCfg, "# Doc: https://github.com/mostlygeek/llama-swap/wiki/Configuration", yml)
	if err != nil {
		return err
	}

	return nil
}

// Set the settings of a model within the llama-swap configuration.
// For /goinfer API, hide the model + prefix the with GI_.
func (cfg *Cfg) setModelSettings(name, path, flags string, goinfer bool) {
	macro := "${cmd-openai}"
	if goinfer {
		macro = "${cmd-goinfer}"
	}

	modelCfg := proxy.ModelConfig{
		Cmd:           macro + " -m " + path + " " + flags,
		Proxy:         "http://localhost:${PORT}",
		CheckEndpoint: "/health",
	}

	if goinfer {
		// hide model name in /v1/models and /upstream API response
		modelCfg.Unlisted = true
		// overrides the model name that is sent to upstream server
		modelCfg.UseModelName = name
	}

	modelName := name
	if goinfer {
		modelName = "GI_" + name
	}

	_, ok := cfg.Swap.Models[modelName]
	if ok {
		slog.Debug("Overwrite config", "model", modelName)
	}

	cfg.Swap.Models[modelName] = modelCfg
}

func (cfg *Cfg) setAPIKeys(debug, noAPIKey bool) {
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

	case debug:
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
