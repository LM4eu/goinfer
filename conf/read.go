// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/llama-swap/proxy/config"
	"github.com/pelletier/go-toml/v2"
)

// ReadGoinferINI loads the configuration file, reads the env vars and verifies the settings.
// Always return a valid configuration, because the receiver may want to write a valid config.
func ReadGoinferINI(noAPIKey bool, extra, start string) (*Cfg, error) {
	data, err := os.ReadFile(GoinferINI)
	if err != nil {
		err = gie.Wrap(err, gie.ConfigErr, "Cannot read", "file", GoinferINI)
		slog.Warn("Skip " + GoinferINI + " => Use default settings and env. vars")
	}

	cfg, er := ReadFileData(data, noAPIKey, extra, start)
	if er != nil {
		if err == nil {
			err = er
		}
	}
	return cfg, err
}

// ReadFileData unmarshals the TOML bytes, applies the env vars and verifies the settings.
// Always return a valid configuration, because the receiver may want to write a valid config.
func ReadFileData(data []byte, noAPIKey bool, extra, start string) (*Cfg, error) {
	cfg := defaultCfg()
	err := cfg.parse(data)
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
	cfg.fixDefaultModel()

	// concatenate host and ports => addr = "host:port"
	if cfg.Host != "" {
		for addr, service := range cfg.Listen {
			if addr != "" && addr[0] != ':' {
				continue
			}
			delete(cfg.Listen, addr)
			p := strings.IndexRune(cfg.Host[:len(cfg.Host)-1], ':')
			if p > 0 { // Host contains the port
				if service == "goinfer" {
					addr = cfg.Host
				} else {
					addr = cfg.Host[:p] + addr
				}
			} else {
				addr = cfg.Host + addr
			}
			cfg.Listen[addr] = service
		}
	}

	er := cfg.validate(noAPIKey)
	if er != nil {
		if err != nil {
			return cfg, errors.Join(err, er)
		}
		return cfg, er
	}
	return cfg, err
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
func (cfg *Cfg) parse(fileData []byte) error {
	if len(fileData) == 0 {
		return gie.New(gie.ConfigErr, "empty", "file", GoinferINI)
	}

	err := toml.Unmarshal(fileData, &cfg)
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "Failed to yaml.Unmarshal", "invalid TOML", string(fileData))
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
	// empty => disable extra_models (goinfer.ini)
	if extra == "" {
		cfg.ExtraModels = nil
	} else if extra[0] == '=' { // starts with "=" => replace extra_models (goinfer.ini)
		cfg.ExtraModels = nil
		extra = extra[1:] // skip first "="
	}

	for pair := range strings.SplitSeq(extra, "|||") {
		// split model=flags
		mf := strings.SplitN(pair, "=", 2)
		model := strings.TrimSpace(mf[0])
		cfg.ExtraModels[model] = ""
		if len(mf) > 1 {
			flags := strings.TrimSpace(mf[1])
			cfg.ExtraModels[model] = flags
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
	cfg.Llama.Verbose = strings.TrimSpace(cfg.Llama.Verbose)
	cfg.Llama.Debug = strings.TrimSpace(cfg.Llama.Debug)
	cfg.Llama.Common = strings.TrimSpace(cfg.Llama.Common)
	cfg.Llama.Goinfer = strings.TrimSpace(cfg.Llama.Goinfer)
}
