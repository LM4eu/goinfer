// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/gie"
)

type (
	// ModelInfo is used for the response of the /models endpoint, including:
	// - command-line flags found of file system
	// - eventual error (if the model is missing or misconfigured).
	ModelInfo struct {
		Params *ModelParams `json:"params,omitempty,omitzero" yaml:"params,omitempty"`
		Path   string       `json:"path,omitempty"            yaml:"path,omitempty"`
		Flags  string       `json:"cmd,omitempty"             yaml:"cmd,omitempty"`
		Origin string       `json:"origin,omitempty"          yaml:"origin,omitempty"`
		Error  string       `json:"error,omitempty"           yaml:"error,omitempty"`
		Size   int64        `json:"size,omitempty"            yaml:"size,omitempty"`
	}

	// ModelParams provides some model customizations.
	ModelParams struct {
		Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
		Flags string `json:"flags,omitempty" yaml:"flags,omitempty"`
		Error string `json:"error,omitempty" yaml:"error,omitempty"`
		Ctx   int    `json:"ctx,omitempty"   yaml:"ctx,omitempty"`
	}
)

const (
	paramsYML     = "params.yml"
	notConfigured = "file present but not configured in llama-swap.yml"
	D_            = "D_"
	A_            = "A_"
	PLUS_A        = "+A"
)

// ListModels returns the model names from the config and from the models_dir.
// TODO: this function should not change cfg.Info.
func (cfg *Cfg) ListModels() map[string]*ModelInfo {
	info := cfg.getInfo()
	for name, mi := range info {
		if info[name].Error == "" {
			mi.Error = notConfigured
			info[name] = mi
		}
	}

	if cfg.Swap != nil {
		for name := range cfg.Swap.Models {
			if len(name) > len(A_) && name[:len(A_)] == A_ && cfg.Swap.Models[name].Unlisted {
				continue // do not report models for /completion endpoint
			}
			cfg.refineModelInfo(name)
		}
	}

	return cfg.getInfo()
}

func (cfg *Cfg) refineModelInfo(name string) {
	mi, ok := cfg.Info[name]
	if ok {
		if mi.Error == notConfigured {
			mi.Error = "" // OK: model is both present in FS and configured in llama-swap.yml
		}
		cfg.Info[name] = mi
		return
	}

	mi = &ModelInfo{}

	// after the first space, the arguments
	pos := strings.Index(cfg.Swap.Models[name].Cmd, " ")
	if pos > 1 {
		// split the arguments at -m: -arg1 -arg2 -m path/to/file.gguf
		flags, path, ok := strings.Cut(cfg.Swap.Models[name].Cmd[pos:], " -m ")
		mi.Flags = flags
		if ok {
			mi.Path = path
			mi.Error = "file absent but configured in llama-swap.yml"
		}
	} else {
		slog.Debug("WARN missing space characters", "cmd", cfg.Swap.Models[name].Cmd)
		mi.Error = "missing space characters in cmd=" + cfg.Swap.Models[name].Cmd
	}
	cfg.Info[name] = mi
}

// getInfo return the cached cfg.Info else compute if it is empty.
func (cfg *Cfg) getInfo() map[string]*ModelInfo {
	if len(cfg.Info) == 0 {
		cfg.updateInfo()
	}
	return cfg.Info
}

// updateInfo search params.yml and *.gguf model files recursively
// in the directories listed in cfg.ModelsDir (colon-separated).
// It aggregates matching files, and updates info.
func (cfg *Cfg) updateInfo() {
	params := map[string]ModelParams{}
	if cfg.Info == nil {
		cfg.Info = make(map[string]*ModelInfo, 16)
	} else {
		clear(cfg.Info)
	}

	// collect params.yml and GUFF files
	for root := range strings.SplitSeq(cfg.ModelsDir, ":") {
		err := cfg.search(params, strings.TrimSpace(root))
		if err != nil {
			slog.Warn("cannot search files in", "root", root, "err", err)
			// should we continue?
		}

		var count uint
		var errStr string

		for _, mi := range cfg.Info {
			if mi.Error != "" {
				count++
				errStr = mi.Error
			}
		}

		switch count {
		case 0:
		case 1:
			slog.Warn(errStr, "root", root)
		default:
			slog.Warn("search models in", "root", root, "warnings", count)
		}
	}

	// Put the ModelParams in the corresponding ModelInfo
	for name, ti := range params {
		mi := cfg.Info[name]
		mi.Params = &ti
		if mi.Flags != "" {
			mi.Flags = ti.Flags
			ti.Flags = ""
		}
		cfg.Info[name] = mi
	}
}

// search walks the given root directory and appends any valid *.gguf model file to
// cfg.Info. It validates each file using validateFile and warns about errors (logs).
func (cfg *Cfg) search(params map[string]ModelParams, root string) error {
	rootFS := NewRoot(root)
	return filepath.WalkDir(root, func(path string, dir fs.DirEntry, err error) error {
		switch {
		case err != nil:
			if dir == nil {
				return gie.Wrap(err, gie.NotFound, "filepath.WalkDir")
			}
			return gie.Wrap(err, gie.NotFound, "filepath.WalkDir", "dir", dir.Name())
		case dir.IsDir():
			// => step into this directory
		case filepath.Base(path) == paramsYML:
			err = keepParams(params, root, path)
			if err != nil {
				slog.Warn("skip params file", "path", path, "err", err)
			}
		case filepath.Ext(path) == ".gguf":
			cfg.keepGUFF(root, path)
		case filepath.Ext(path) == ".sh":
			cfg.keepFlags(rootFS, path[len(root):])
		default:
		}
		return nil
	})
}

func keepParams(params map[string]ModelParams, root, path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "os.ReadFile", "file", path)
	}

	if len(data) == 0 {
		slog.Info("Empty params", "file", path)
		return nil
	}

	slog.Debug("Found params", "file", path)

	var tpl map[string]ModelParams
	err = json.Unmarshal(data, &tpl)
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "json.Unmarshal", "file", path, "100FirsBytes", string(data[:100]))
	}

	for name, ti := range tpl {
		if old, ok := params[name]; ok {
			slog.Warn("Duplicated params", "dir", root, "name", name, "old", old, "new", ti)
			ti.Error = "two files have same model name (must be unique)"
		}
		ti.Flags = replaceDIR(path, ti.Flags)
		params[name] = ti
	}

	return nil
}

func (cfg *Cfg) keepGUFF(root, path string) {
	size, err := verify(path)
	if err != nil {
		slog.Debug("skip", "GGUF", path, "err", err)
		return
	}

	slog.Debug("Found", "model", path)

	name, flags, origin := getNameAndFlags(root, path)

	flags = replaceDIR(path, flags)

	mi := ModelInfo{Params: nil, Flags: flags, Path: path, Error: "", Size: size, Origin: origin}
	if old, ok := cfg.Info[name]; ok {
		slog.Debug("WARN Duplicated models", "dir", root, "name", name, "old", old, "new", mi)
		mi.Error = "two files have same model name (must be unique)"
	}
	cfg.Info[name] = &mi
}

func (cfg *Cfg) keepFlags(root Root, path string) {
	modelPath, args := extractModelNameAndFlags(root, path)
	if modelPath == nil {
		return
	}

	flags := replaceDIR(path, string(args))

	mi := ModelInfo{Params: nil, Flags: flags, Path: string(modelPath), Origin: path}
	if cfg.Shells == nil {
		slog.Debug("Found first", "shell", path)
		cfg.Shells = []*ModelInfo{&mi}
		return
	}

	for _, mm := range cfg.Shells {
		if mm.Origin == mi.Origin {
			slog.Debug("Already present", "shell", path)
			return
		}
	}

	cfg.Shells = append(cfg.Shells, &mi)
}
