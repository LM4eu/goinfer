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

	"github.com/lynxai-team/garcon/gerr"
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
		Issue  string       `json:"error,omitempty"           yaml:"error,omitempty"`
		Size   int64        `json:"size,omitempty"            yaml:"size,omitempty"`
	}

	// ModelParams provides some model customizations.
	ModelParams struct {
		Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
		Flags string `json:"flags,omitempty" yaml:"flags,omitempty"`
		Issue string `json:"error,omitempty" yaml:"error,omitempty"`
		Ctx   int    `json:"ctx,omitempty"   yaml:"ctx,omitempty"`
	}
)

const (
	paramsYML     = "params.yml"
	notConfigured = "file present but not configured in llama-swap.yml"
	plusA         = "+A"
	agentSmith    = false // true => add suffixed models for Agent-Smith compliance
)

// ListModels returns the model names from the config and from the models_dir.
// TODO: this function should not change cfg.Info.
func (cfg *Cfg) ListModels() map[string]*ModelInfo {
	info := cfg.getInfo()
	for name, mi := range info {
		if info[name].Issue == "" {
			mi.Issue = notConfigured
			info[name] = mi
		}
	}

	if cfg.Swap != nil {
		for model := range cfg.Swap.Models {
			if agentSmith && strings.HasSuffix(model, plusA) && cfg.Swap.Models[model].Unlisted {
				continue // do not report models for /completion endpoint
			}
			cfg.refineModelInfo(model)
		}
	}

	return cfg.getInfo()
}

func (cfg *Cfg) refineModelInfo(name string) {
	mi, ok := cfg.Info[name]
	if ok {
		if mi.Issue == notConfigured {
			mi.Issue = "" // OK: model is both present in FS and configured in llama-swap.yml
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
			mi.Issue = "file absent but configured in llama-swap.yml"
		}
	} else {
		slog.Debug("WARN missing space characters", "cmd", cfg.Swap.Models[name].Cmd)
		mi.Issue = "missing space characters in cmd=" + cfg.Swap.Models[name].Cmd
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
	if cfg.Info == nil {
		cfg.Info = make(map[string]*ModelInfo, 16)
	} else {
		clear(cfg.Info)
	}

	var params map[string]ModelParams
	var shells []*ModelInfo

	// collect params.yml and GUFF files
	for root := range strings.SplitSeq(cfg.ModelsDir, ":") {
		rootFS := NewRoot(strings.TrimSpace(root))
		var err error
		err = cfg.search(params, &shells, rootFS)
		if err != nil {
			slog.Warn("cannot search files in", "root", root, "err", err)
			// should we continue?
		}

		var count uint
		var errStr string

		for _, mi := range cfg.Info {
			if mi.Issue != "" {
				count++
				errStr = mi.Issue
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

	// Reuse the shell scripts
	slog.Debug("parse ", "shells", len(shells), "model-presets", len(cfg.Info))
	for _, sh := range shells {
		var mi *ModelInfo
		modelBase := filepath.Base(sh.Path)
		for _, mi = range cfg.Info {
			if modelBase != filepath.Base(mi.Path) {
				break
			}
		}

		if mi == nil {
			slog.Debug("WARN model from shell not found", "shell", sh.Origin, "model", sh.Path)
			continue
		}

		name := filepath.Base(sh.Origin)
		name = strings.TrimSuffix(name, ".sh")
		if old, ok := cfg.Info[name]; ok {
			slog.Debug("WARN Duplicated models (new is from shell)", "name", name, "old", old, "new", sh)
			mi.Issue = "two ModelInfo have same model name (skip " + old.Path
			if old.Origin != "" {
				mi.Issue += " origin=" + old.Origin
			}
			mi.Issue += ")"
			if old.Issue != "" {
				mi.Issue += " " + old.Issue
			}
		} else {
			slog.Debug("add from shell", "name", name, "model", sh.Path, "origin", sh.Origin)
		}
		sh.Size = mi.Size
		cfg.Info[name] = sh
	}

	slog.Debug("registered ", "model-presets", len(cfg.Info))
}

// search walks the given root directory and appends any valid *.gguf model file to
// cfg.Info. It validates each file using validateFile and warns about errors (logs).
func (cfg *Cfg) search(params map[string]ModelParams, shells *[]*ModelInfo, root Root) error {
	err := fs.WalkDir(root.FS, ".", func(path string, dir fs.DirEntry, err error) error {
		switch {
		case err != nil:
			if dir == nil {
				return gerr.Wrap(err, gerr.NotFound, "filepath.WalkDir")
			}
			return gerr.Wrap(err, gerr.NotFound, "filepath.WalkDir", "dir", dir.Name())
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
			keepFlags(shells, root, path)
		default:
		}
		return nil
	})
	return err
}

func keepParams(params map[string]ModelParams, root Root, path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return gerr.Wrap(err, gerr.ConfigErr, "os.ReadFile", "file", path)
	}

	if len(data) == 0 {
		slog.Info("Empty params", "file", path)
		return nil
	}

	slog.Debug("Found params", "file", path)

	var mp map[string]ModelParams
	err = json.Unmarshal(data, &mp)
	if err != nil {
		return gerr.Wrap(err, gerr.ConfigErr, "json.Unmarshal", "file", path, "100FirsBytes", string(data[:100]))
	}

	for name, p := range mp {
		p.Flags = replaceDIR(root.FullPath(path), p.Flags)
		if params == nil {
			params = map[string]ModelParams{name: p}
			continue
		}
		old, ok := params[name]
		if ok {
			slog.Warn("Duplicated params", "root", root.Path, "name", name, "old", old, "new", p)
			p.Issue = "two ModelParams have same model name (skip " + old.Flags + ")"
			if old.Issue != "" {
				p.Issue += " " + old.Issue
			}
		}
		params[name] = p
	}

	return nil
}

func (cfg *Cfg) keepGUFF(root Root, path string) {
	size, err := verify(root, path)
	if err != nil {
		slog.Debug("skip GGUF", "root", root.Path, "file", path, "err", err)
		return
	}

	slog.Debug("Found model", "root", root.Path, "file", path)

	name, flags, origin := getNameAndFlags(root.Path, path)

	fullPath := root.FullPath(path)

	mi := ModelInfo{
		Flags:  replaceDIR(fullPath, flags),
		Path:   fullPath,
		Size:   size,
		Origin: root.FullPath(origin),
	}
	if old, ok := cfg.Info[name]; ok {
		slog.Debug("WARN Duplicated models", "root", root.Path, "name", name, "old", old, "new", mi)
		mi.Issue = "two ModelInfo have same model name (skip " + old.Path
		if old.Origin != "" {
			mi.Issue += " origin=" + old.Origin
		}
		mi.Issue += ")"
		if old.Issue != "" {
			mi.Issue += " " + old.Issue
		}
	}
	cfg.Info[name] = &mi
}

func keepFlags(shells *[]*ModelInfo, root Root, path string) {
	modelPath, flags := extractModelNameAndFlags(root, path)
	if modelPath == nil {
		return
	}

	origin := root.FullPath(path)
	mi := &ModelInfo{
		Flags:  replaceDIR(root.FullPath(path), string(flags)),
		Path:   string(modelPath),
		Origin: origin,
	}

	if *shells == nil {
		*shells = []*ModelInfo{mi}
		slog.Debug("Add", "shell", origin, "total", len(*shells))
		return
	}

	for _, mi := range *shells {
		if mi.Origin == origin {
			slog.Warn("Already present", "shell", path, "total", len(*shells))
			return
		}
	}

	*shells = append(*shells, mi)
	slog.Debug("Add", "shell", origin, "total", len(*shells))
}

// replaceDIR in flags by the current dir of he file.
// When using models like GPT OSS, we need to provide a grammar file.
// See: github.com/ggml-org/llama.cpp/discussions/15396#discussioncomment-14145537
// We want the possibility to keep both model and grammar files in the same folder.
// But we also want to be free to move that folder
// without having to update the path within tho command line arguments.
// Thus, we use $DIR as a placeholder for the folder.
func replaceDIR(path, flags string) string {
	return strings.ReplaceAll(flags, "$DIR", filepath.Dir(path))
}
