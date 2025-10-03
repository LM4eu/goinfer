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
	"github.com/LM4eu/llama-swap/proxy/config"
	"go.yaml.in/yaml/v4"
)

// WriteMainCfg populates the configuration with defaults, applies environment variables,
// writes the resulting configuration to the given file, and mutates the receiver.
func (cfg *Cfg) WriteMainCfg(mainCfg string, debug, noAPIKey bool) error {
	cfg.Llama = defaultCfg.Llama
	cfg.ModelsDir = defaultCfg.ModelsDir
	cfg.Server = defaultCfg.Server

	cfg.applyEnvVars()
	cfg.trimParamValues()

	cfg.setAPIKeys(debug, noAPIKey)

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "failed to yaml.Marshal")
	}

	err = cfg.validateMain(noAPIKey)
	if err != nil {
		return err
	}

	return writeWithHeader(mainCfg, "# Configuration of https://github.com/LM4eu/goinfer", yml)
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

	cfg.Swap.StartPort = 5800
	cfg.Swap.HealthCheckTimeout = 120
	cfg.Swap.MetricsMaxInMemory = 500

	common, ok := cfg.Llama.Args["common"]
	if !ok {
		common = argsCommon
	}

	infer, ok := cfg.Llama.Args["infer"]
	if !ok {
		infer = argsInfer
	}

	cfg.Swap.Macros = map[string]string{
		"cmd-fim":    cfg.Llama.Exe + " " + common,
		"cmd-openai": cfg.Llama.Exe + " " + common + " --port ${PORT}",
		"cmd-infer":  cfg.Llama.Exe + " " + common + " --port ${PORT} " + infer,
	}

	info, err := cfg.search()
	if err != nil {
		return err
	}

	if cfg.Swap.Models == nil {
		cfg.Swap.Models = make(map[string]config.ModelConfig, 2*len(info)+9)
	}

	openaiCfg := &config.ModelConfig{Proxy: "http://localhost:${PORT}"}
	fimCfg := &config.ModelConfig{Proxy: "http://localhost:8012"} // the flag --fim-qwen-xxxx sets port=8012
	goinferCfg := &config.ModelConfig{
		Proxy:    "http://localhost:${PORT}",
		Unlisted: true, // hide model in /v1/models and /upstream responses
	}

	// Output of `llama-server -h` contains:
	//  --embd-bge-small-en-default  bge-small-en-v1.5
	//  --embd-e5-small-en-default   e5-small-v2
	//  --embd-gte-small-default     gte-small
	//  --fim-qwen-1.5b-default      Qwen 2.5 Coder 1.5B
	//  --fim-qwen-3b-default        Qwen 2.5 Coder 3B
	//  --fim-qwen-7b-default        Qwen 2.5 Coder 7B
	//  --fim-qwen-7b-spec           Qwen 2.5 Coder 7B + 0.5B draft for speculative decoding
	//  --fim-qwen-14b-spec          Qwen 2.5 Coder 14B + 0.5B draft for speculative decoding
	//  --fim-qwen-30b-default       Qwen 3 Coder 30B A3B Instruct
	cfg.addModelCfg("ggml-org/bge-small-en-v1.5-Q8_0-GGUF", "${cmd-openai} --embd-bge-small-en-default", openaiCfg)
	cfg.addModelCfg("ggml-org/e5-small-v2-Q8_0-GGUF", "${cmd-openai} --embd-e5-small-en-default", openaiCfg)
	cfg.addModelCfg("ggml-org/gte-small-Q8_0-GGUF", "${cmd-openai} --embd-gte-small-default", openaiCfg)
	cfg.addModelCfg("ggml-org/Qwen2.5-Coder-1.5B-Q8_0-GGUF", "${cmd-fim} --fim-qwen-1.5b-default", fimCfg)
	cfg.addModelCfg("ggml-org/Qwen2.5-Coder-3B-Q8_0-GGUF", "${cmd-fim} --fim-qwen-3b-default", fimCfg)
	cfg.addModelCfg("ggml-org/Qwen2.5-Coder-7B-Q8_0-GGUF", "${cmd-fim} --fim-qwen-7b-default", fimCfg)
	cfg.addModelCfg("ggml-org/Qwen2.5-Coder-7B-Q8_0-GGUF", "${cmd-fim} --fim-qwen-7b-spec", fimCfg)
	cfg.addModelCfg("ggml-org/Qwen2.5-Coder-14B-Q8_0-GGUF", "${cmd-fim} --fim-qwen-14b-spec", fimCfg)
	cfg.addModelCfg("ggml-org/Qwen3-Coder-30B-A3B-Instruct-Q8_0-GGUF", "${cmd-fim} --fim-qwen-30b-default", fimCfg)

	// For each model, set two model settings:
	// 1. for the OpenAI endpoints
	// 2. for the /completion endpoint (prefix with GI_ and hide the model)
	for name, mi := range info {
		goinferCfg.UseModelName = name // overrides the model name that is sent to /upstream server
		args := " " + mi.Flags + " -m " + mi.Path
		cfg.addModelCfg(name, "${cmd-openai}"+args, openaiCfg)       // API=OpenAI
		cfg.addModelCfg("GI_"+name, "${cmd-infer}"+args, goinferCfg) //
	}

	err = cfg.ValidateSwap()
	if err != nil {
		return err
	}

	yml, er := yaml.Marshal(&cfg.Swap)
	if er != nil {
		return gie.Wrap(er, gie.ConfigErr, "failed to marshal the llama-swap config")
	}

	err = writeWithHeader(swapCfg, "# Doc: https://github.com/mostlygeek/llama-swap/wiki/Configuration", yml)
	if err != nil {
		return err
	}

	return nil
}

// Add the model settings within the llama-swap configuration.
func (cfg *Cfg) addModelCfg(modelName, cmd string, mc *config.ModelConfig) {
	mCfg := *mc // copy
	mCfg.Cmd = cmd
	mCfg.CheckEndpoint = "/health"

	old, ok := cfg.Swap.Models[modelName]
	if ok {
		slog.Debug("Overwrite config", "old", old)
		slog.Debug("Overwrite config", "new", modelName)
	}

	cfg.Swap.Models[modelName] = mCfg
}

func (cfg *Cfg) setAPIKeys(debug, noAPIKey bool) {
	if cfg.Server.APIKey == "" {
		slog.Info("Configuration file uses API keys from environment")
		return
	}

	switch {
	case noAPIKey:
		cfg.Server.APIKey = unsetAPIKey
		slog.Info("Flag -no-api-key => Do not generate API keys")

	case debug:
		cfg.Server.APIKey = debugAPIKey
		slog.Warn("API keys are DEBUG => security threat")

	default:
		cfg.Server.APIKey = gen64HexDigits()
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
		return gie.Wrap(err, gie.ConfigErr, "failed to create file="+path)
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
		return gie.Wrap(err, gie.ConfigErr, "failed to write file="+path)
	}

	return nil
}
