// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/LM4eu/goinfer/gie"
	"github.com/pelletier/go-toml/v2"
)

// GoinferINI is the config filename.
const GoinferINI = "goinfer.ini"

// ReadGoinferINI loads the configuration file, reads the env vars and verifies the settings.
// Always return a valid configuration, because the receiver may want to write a valid config.
func ReadGoinferINI(noAPIKey bool, extra, start string) (*Cfg, error) {
	data, err := os.ReadFile(GoinferINI)
	if err != nil {
		err = gie.Wrap(err, gie.ConfigErr, "Cannot read", "file", GoinferINI)
		slog.Warn("Skip " + GoinferINI + " => Use default settings and env. vars")
	}

	cfg, er := parseGoinferINI(data, noAPIKey, extra, start)
	if er != nil {
		if err == nil {
			err = er
		}
	}
	return cfg, err
}

// parseGoinferINI unmarshals the TOML bytes, applies the env vars and verifies the settings.
// Always return a valid configuration, because the receiver may want to write a valid config.
func parseGoinferINI(data []byte, noAPIKey bool, extra, start string) (*Cfg, error) {
	cfg := DefaultCfg()
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
		switch {
		case strings.ContainsRune(cfg.Host[:len(cfg.Host)-1], ':'):
			cfg.Addr = cfg.Host // Host contains the port
		case cfg.Addr == "":
			cfg.Addr = cfg.Host + ":8080"
		case cfg.Addr[0] == ':':
			cfg.Addr = cfg.Host + cfg.Addr
		default:
			p := strings.IndexRune(cfg.Addr[1:], ':')
			cfg.Addr = cfg.Host + cfg.Addr[p:]
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

// WriteGoinferINI populates the configuration with defaults, applies environment variables,
// writes the resulting configuration to the given file.
func (cfg *Cfg) WriteGoinferINI(debug, noAPIKey bool) error {
	data, err := cfg.GenGoinferINI(debug, noAPIKey)
	er := writeWithHeader(GoinferINI, "# Configuration of https://github.com/LM4eu/goinfer\n\n", data)
	if er != nil {
		if err != nil {
			return errors.Join(err, er)
		}
		return er
	}
	return err
}

// GenGoinferINI sets the API keys, reads the environment variables,
// fix some settings and writes the result config to a buffer.
func (cfg *Cfg) GenGoinferINI(debug, noAPIKey bool) ([]byte, error) {
	cfg.setAPIKey(debug, noAPIKey)
	cfg.applyEnvVars()
	cfg.trimParamValues()
	cfg.fixDefaultModel()

	err := cfg.validate(noAPIKey)

	data, er := toml.Marshal(&cfg)
	if er != nil {
		er = gie.Wrap(err, gie.ConfigErr, "failed to yaml.Marshal", "cfg", cfg)
		if err != nil {
			return data, errors.Join(err, er)
		}
		return data, er
	}
	return data, err
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
		model, flags, ok := strings.Cut(pair, "=")
		model = strings.TrimSpace(model)
		model = strings.Replace(model, "-GGUF", "", 1)
		cfg.ExtraModels[model] = ""
		if ok {
			cfg.ExtraModels[model] = strings.TrimSpace(flags)
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

func writeWithHeader(path, header string, data []byte) error {
	path = filepath.Clean(path)
	file, err := os.Create(path)
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "failed to create file="+path)
	}

	_, err = file.WriteString(header)
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
