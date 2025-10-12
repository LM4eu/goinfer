// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	er := writeWithHeader(GoinferYML, "# Configuration of https://github.com/LM4eu/goinfer", yml)
	if er != nil {
		if err != nil {
			return errors.Join(err, er)
		}
		return er
	}
	return err
}

// WriteBytes populates the configuration with defaults, applies environment variables,
// and writes the resulting configuration to a buffer.
func (cfg *Cfg) WriteBytes(debug, noAPIKey bool) ([]byte, error) {
	cfg.setAPIKey(debug, noAPIKey)
	cfg.applyEnvVars()
	cfg.trimParamValues()
	cfg.fixDefaultModel()

	err := cfg.validate(noAPIKey)

	yml, er := yaml.Marshal(&cfg)
	if er != nil {
		er = gie.Wrap(err, gie.ConfigErr, "failed to yaml.Marshal", "cfg", cfg)
		if err != nil {
			return yml, errors.Join(err, er)
		}
		return yml, er
	}
	return yml, err
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

	// set the macros
	commonArgs := " " + cfg.Llama.Args.Common
	if verbose {
		commonArgs += " " + cfg.Llama.Args.Verbose
	}
	if debug {
		commonArgs += " " + cfg.Llama.Args.Debug
	}
	cfg.Swap.Macros = config.MacroList{
		{Name: "cmd-fim", Value: cfg.Llama.Exe + commonArgs},
		{Name: "cmd-common", Value: cfg.Llama.Exe + commonArgs + " --port ${PORT}"},
		{Name: "cmd-goinfer", Value: cfg.Llama.Exe + commonArgs + " --port ${PORT} " + cfg.Llama.Args.Goinfer},
	}

	cfg.setSwapModels()

	// when Goinfer starts, llama-server is started with the DefaultModel
	if cfg.DefaultModel != "" {
		cfg.Swap.Hooks.OnStartup.Preload = []string{cfg.DefaultModel}
	}

	err := cfg.ValidateSwap()
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
	cfg.setSwapModels()

	_, ok := cfg.Swap.Models[cfg.DefaultModel]
	if ok {
		return // valid
	}

	betterName, reason := cfg.selectModelName(cfg.DefaultModel, true)

	if cfg.DefaultModel != "" && cfg.DefaultModel != betterName {
		slog.Warn("change invalid default_model", "old", cfg.DefaultModel, "new", betterName, "reason", reason+" default_model")
	}

	cfg.DefaultModel = betterName
}

func (cfg *Cfg) FixModelName(modelName string) string {
	_, ok := cfg.Swap.Models[modelName]
	if ok {
		return modelName // valid
	}

	betterName, reason := cfg.selectModelName(modelName, false)

	if modelName != "" && modelName != betterName {
		slog.Info("change requested model (invalid)", "old", modelName, "new", betterName, "reason", reason+" the requested model")
	}

	return betterName
}

func (cfg *Cfg) selectModelName(model string, useSmallest bool) (betterName, reason string) {
	lowModel := strings.ToLower(model)
	lowPath := ""
	supName := ""    // the DefaultModel contains a model name
	subName := ""    // a model name contains the DefaultModel
	supname := ""    // same as supName but with a lowercase comparison
	subname := ""    // same as subName but with a lowercase comparison
	minName := model // the name of the smallest model
	minSize := int64(math.MaxInt64)
	for name, mi := range cfg.Info {
		lowName := strings.ToLower(name)
		switch {
		case model == "": // skip the following strings.Contains checks
		case strings.Contains(mi.Path, model):
			slog.Info("replace path or filename by valid model name", "old", model, "new", name)
			return name, "use a model having a path/file containing the"
		case strings.Contains(strings.ToLower(mi.Path), lowModel):
			lowPath = name
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
		if useSmallest && mi.Size < minSize {
			minSize = mi.Size
			minName = name
		}
	}
	// if the given model name is not related to a pathname or filename,
	// then select one by order of preference:
	switch {
	case lowPath != "":
		return lowPath, "use a model having a path/file insensitively containing the"
	case subName != "":
		return subName, "use a full model name containing the"
	case supName != "":
		return supName, "use a model name being a substring of"
	case subname != "":
		return subname, "use a full model name insensitively containing the"
	case supname != "":
		return supname, "use a model name being a insensitive substring of"
	default:
		return minName, "use the smallest model because no model related to"
	}
}

func (cfg *Cfg) setSwapModels() {
	cfg.updateInfo()

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
		case strings.Contains(flags, "-m "), strings.Contains(flags, "-hf "):
			cfg.addModelCfg(model, "${cmd-common} "+flags, commonMC)
			cfg.addModelCfg("GI_"+model, "${cmd-goinfer} "+flags, goinferMC)
		default:
			cfg.addModelCfg(model, "${cmd-common} -hf "+model+" "+flags, commonMC)
			cfg.addModelCfg("GI_"+model, "${cmd-goinfer} -hf "+model+" "+flags, goinferMC)
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
}

// Add the model settings within the llama-swap configuration.
func (cfg *Cfg) addModelCfg(modelName, cmd string, mc *config.ModelConfig) {
	mCfg := *mc // copy
	mCfg.Cmd = cmd

	mCfg.CheckEndpoint = "/health"
	if strings.Contains(cmd, " -hf ") {
		// -hf may download a model for a while
		// but /health check will stop it,
		// so better to disable /health check
		mCfg.CheckEndpoint = "none"
	}

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
		slog.Info("Generated random API key")
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
