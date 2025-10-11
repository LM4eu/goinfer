// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/llama-swap/proxy/config"
	"go.yaml.in/yaml/v4"
)

// WriteGoinferYML populates the configuration with defaults, applies environment variables,
// writes the resulting configuration to the given file.
func (cfg *Cfg) WriteGoinferYML(debug, noAPIKey bool) error {
	yml, err := cfg.WriteBytes(debug, noAPIKey)
	if err != nil {
		return err
	}
	return writeWithHeader(GoinferYML, "# Configuration of https://github.com/LM4eu/goinfer", yml)
}

// WriteBytes populates the configuration with defaults, applies environment variables,
// and writes the resulting configuration to a buffer.
func (cfg *Cfg) WriteBytes(debug, noAPIKey bool) ([]byte, error) {
	cfg.setAPIKey(debug, noAPIKey)
	cfg.applyEnvVars()
	cfg.fixDefaultModel()
	cfg.trimParamValues()

	err := cfg.validate(noAPIKey)
	if err != nil {
		return nil, err
	}

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, gie.Wrap(err, gie.ConfigErr, "failed to yaml.Marshal", "cfg", cfg)
	}

	return yml, nil
}

// WriteLlamaSwapYML generates the llama-swap configuration.
func WriteLlamaSwapYML(yml []byte) error {
	// start the llama-swap.yml with these comment lines:
	header := `# DO NOT EDIT - This file is generated at Goinfer start time.
# Doc:
# - https://github.com/LM4eu/goinfer/?tab=readme-ov-file#llamaswapyml
# - https://github.com/mostlygeek/llama-swap/wiki/Configuration
`
	return writeWithHeader(LlamaSwapYML, header, yml)
}

// GenSwapYAMLData generates the llama-swap configuration.
func (cfg *Cfg) GenSwapYAMLData(verbose, debug bool) ([]byte, error) {
	switch {
	case debug:
		cfg.Swap.LogLevel = "debug"
	case verbose:
		cfg.Swap.LogLevel = "info"
	default:
		cfg.Swap.LogLevel = "warn"
	}

	// HealthCheckTimeout has some limitations:
	// - "llama-server -hf model-name" is nice to ease deployment, but may take one or two hours for very large models (200GB+)
	// - very large models (480B) need minutes to initialize their tensors
	// - startup time is different from runtime health check (TODO: different check during startup)
	cfg.Swap.HealthCheckTimeout = 300 // 5 minutes to initialize 480B model
	cfg.Swap.MetricsMaxInMemory = 500
	cfg.Swap.StartPort = 5800

	commonArgs := " " + cfg.Llama.Args.Common
	goinferArgs := " " + cfg.Llama.Args.Goinfer
	if verbose {
		commonArgs += " " + cfg.Llama.Args.Verbose
	}
	if debug {
		commonArgs += " " + cfg.Llama.Args.Debug
	}

	cfg.Swap.Macros = config.MacroList{
		{Name: "cmd-fim", Value: cfg.Llama.Exe + commonArgs},
		{Name: "cmd-common", Value: cfg.Llama.Exe + commonArgs + " --port ${PORT}"},
		{Name: "cmd-goinfer", Value: cfg.Llama.Exe + commonArgs + " --port ${PORT}" + goinferArgs},
	}

	err := cfg.setSwapModels()
	if err != nil {
		return nil, err
	}

	// when Goinfer starts, llama-server is started with the DefaultModel
	if cfg.DefaultModel != "" {
		cfg.Swap.Hooks.OnStartup.Preload = []string{cfg.DefaultModel}
	}

	err = cfg.ValidateSwap()
	if err != nil {
		return nil, err
	}

	yml, er := yaml.Marshal(&cfg.Swap)
	if er != nil {
		return nil, gie.Wrap(er, gie.ConfigErr, "failed to marshal the llama-swap config")
	}

	return yml, nil
}

func (cfg *Cfg) fixDefaultModel() {
	err := cfg.setSwapModels()
	if err != nil {
		return
	}

	cfg.DefaultModel = cfg.FixModelName(cfg.DefaultModel, true)
}

func (cfg *Cfg) FixModelName(model string, useSmallest bool) string {
	_, ok := cfg.Swap.Models[model]
	if ok {
		return model // valid model name
	}

	supName := "" // the DefaultModel contains a model name
	subName := "" // a model name contains the DefaultModel
	minName := "" // the name of the smallest model
	supname := "" // same as supName but with a lowercase comparison
	subname := "" // same as subName but with a lowercase comparison
	lowModel := strings.ToLower(model)
	minSize := int64(math.MaxInt64)
	for name, mi := range cfg.Info {
		lowName := strings.ToLower(name)

		switch {
		case model == "":
			// skip the following strings.Contains checks
		case strings.Contains(mi.Path, model):
			slog.Info("replace path or filename by valid model name", "old", model, "new", name)
			return name
		case strings.Contains(model, name): // this model name is a portion of the default_model
			subName = name
		case strings.Contains(name, model): // this model name contains the default_model
			supName = name
		case strings.Contains(lowModel, lowName): // same as above but in lower case
			subname = name
		case strings.Contains(lowName, lowModel):
			supname = name
		default:
		}

		if minSize > mi.Size {
			minSize = mi.Size
			minName = name
		}
	}

	// if the default_model is not related to a pathname or filename,
	// then select one by order of preference:
	//   - subName is a portion of the default_model
	//   - supName contains the default_model
	//   - minName = name of the model having the smallest size

	var reason string
	newModel := model

	switch {
	case subName != "":
		newModel = subName
		reason = "a model name being a substring of the default_model"
	case supName != "":
		newModel = supName
		reason = "a model name containing the default_model"
	case subname != "":
		newModel = subname
		reason = "a model name being a substring of the default_model"
	case supname != "":
		newModel = supname
		reason = "a model name containing the default_model"
	default:
		if useSmallest {
			newModel = minName
			reason = "the smallest model"
		}
	}

	if model != "" {
		slog.Info("default_model is invalid, select "+reason, "old", newModel, "new", model)
	}

	return newModel
}

func (cfg *Cfg) setSwapModels() error {
	err := cfg.updateInfo()
	if err != nil {
		return err
	}

	if cfg.Swap.Models == nil {
		cfg.Swap.Models = make(map[string]config.ModelConfig, 2*len(cfg.Info)+9)
	}

	commonMC := &config.ModelConfig{Proxy: "http://localhost:${PORT}"}
	fimMC := &config.ModelConfig{Proxy: "http://localhost:8012"} // the flag --fim-qwen-xxxx sets port=8012
	goinferMC := &config.ModelConfig{
		Proxy:    "http://localhost:${PORT}",
		Unlisted: true, // hide model in /v1/models and /upstream responses
	}

	for model, flags := range cfg.ExtraModels {
		switch {
		case flags == "":
			cfg.addModelCfg(model, "${cmd-common} -hf "+model, commonMC)
			cfg.addModelCfg("GI_"+model, "${cmd-goinfer} -hf "+model, goinferMC)
		case strings.HasPrefix(flags, "--embd-"):
			cfg.addModelCfg(model, "${cmd-common} "+flags, commonMC)
		case strings.HasPrefix(flags, "--fim-"):
			cfg.addModelCfg(model, "${cmd-common} "+flags, fimMC)
		default:
			cfg.addModelCfg(model, "${cmd-common} "+flags, commonMC)
			cfg.addModelCfg("GI_"+model, "${cmd-goinfer} "+flags, goinferMC)
		}
	}

	// For each model, set two model settings:
	// 1. for the OpenAI endpoints
	// 2. for the /completion endpoint (prefix with GI_ and hide the model)
	for name, mi := range cfg.Info {
		goinferMC.UseModelName = name // overrides the model name that is sent to /upstream server
		args := " " + mi.Flags + " -m " + mi.Path
		cfg.addModelCfg(name, "${cmd-common}"+args, commonMC)         // API for Cline, RooCode, RolePlay...
		cfg.addModelCfg("GI_"+name, "${cmd-goinfer}"+args, goinferMC) // API for Agent-Smith...
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

func (cfg *Cfg) setAPIKey(debug, noAPIKey bool) {
	switch {
	case noAPIKey:
		cfg.APIKey = unsetAPIKey
		slog.Info("Flag -no-api-key => Do not generate API key")

	case debug:
		cfg.APIKey = debugAPIKey
		slog.Warn("API key is DEBUG => security threat")

	default:
		cfg.APIKey = gen64HexDigits()
		slog.Info("Generated random secured API key")
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
