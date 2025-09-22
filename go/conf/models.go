// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/gie"
)

type ModelInfo struct {
	Flags string `json:"flags,omitempty" yaml:"flags,omitempty"`
	Path  string `json:"path,omitempty"  yaml:"path,omitempty"`
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
}

// Search returns a slice of absolute file paths for all *.gguf model files
// found under the directories listed in cfg.ModelsDir (colon-separated).
// It walks each directory recursively, aggregates matching files, and returns any error encountered.
func (cfg *Cfg) ListModels() (map[string]ModelInfo, error) {
	modelFiles, err := cfg.searchAll()
	if err != nil {
		if cfg.Debug {
			slog.Info("Search models", "err", err)
		}
	}

	all := make(map[string]ModelInfo, len(modelFiles))
	for _, path := range modelFiles {
		name, flags := extractFlags(path)
		e := "file present but not configured in llama-swap.yml"
		if _, ok := all[name]; ok {
			e = "two files have same model name (must be unique)"
		}
		all[name] = ModelInfo{flags, path, e}
	}

	for name := range cfg.Proxy.Models {
		if len(name) > 3 && name[:3] == "GI_" && cfg.Proxy.Models[name].Unlisted {
			continue // do not report models for /goinfer endpoint
		}

		info, ok := all[name]
		if ok {
			info.Error = "" // OK: model is both present in FS and configured in llama-swap.yml
		} else {
			cmd := strings.SplitN(cfg.Proxy.Models[name].Cmd, "--model", 2)
			if len(cmd) > 0 {
				info.Flags = cmd[0]
			}
			if len(cmd) > 1 {
				info.Path = cmd[1]
			}
			info.Error = "file absent but configured in llama-swap.yml"
		}
		all[name] = info
	}

	return all, err
}

// searchAll returns a slice of absolute file paths for all *.gguf model files
// found under the directories listed in cfg.ModelsDir (colon-separated).
// It walks each directory recursively, aggregates matching files, and returns any error encountered.
func (cfg *Cfg) searchAll() ([]string, error) {
	modelFiles := make([]string, 0, len(cfg.ModelsDir)/2)

	for root := range strings.SplitSeq(cfg.ModelsDir, ":") {
		err := cfg.search(&modelFiles, strings.TrimSpace(root))
		if err != nil {
			if cfg.Verbose {
				slog.Info("Searching model files", "root", root)
			}
			return nil, err
		}
	}

	return modelFiles, nil
}

func (cfg *Cfg) search(files *[]string, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return gie.Wrap(err, gie.TypeNotFound, "filepath.WalkDir", "path="+d.Name())
		}

		if d.IsDir() {
			return nil // => step into this directory
		}

		if strings.HasSuffix(path, ".gguf") {
			if cfg.Debug {
				slog.Info("Found", "model", path)
			}

			err := validateFile(path)
			if err != nil {
				slog.Info("Skip", "model", path)
			} else {
				*files = append(*files, path)
			}
		}

		return nil
	})
}

// extractFlags extracts the model name and its flags from a file path.
// It looks for a pattern starting with "&" and splits the remaining string by "&"
// to get individual flag components.
// Each component is then split by "=" to separate key and value,
// with the key prefixed by "-" to form command-line style flags.
// Returns the model name and a single string with flags separated by spaces.
func extractFlags(path string) (string, string) {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	pos := strings.Index(stem, "&")
	if pos < 0 {
		return stem, ""
	}

	var flags []string

	// Slice after the first '&' to avoid an empty first element.
	for f := range strings.SplitSeq(stem[pos+1:], "&") {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) > 0 {
			kv[0] = "-" + kv[0]
			flags = append(flags, kv...)
		}
	}

	return stem[:pos], strings.Join(flags, " ")
}

func (cfg *Cfg) countModels() int {
	modelFiles, err := cfg.searchAll()
	if err != nil {
		return 0
	}
	return len(modelFiles)
}

func (cfg *Cfg) validateModelFiles() error {
	if len(cfg.Proxy.Models) == 0 {
		n := cfg.countModels()
		if n == 0 {
			slog.Error("No *.gguf files found", "dir", cfg.ModelsDir)
			return gie.ErrConfigValidation
		}

		slog.Warn("No model configured => Use flag -gen-px-cfg to fill the config with", "files", n)
		return nil
	}

	for i := range cfg.Proxy.Models {
		var previous string
		for arg := range strings.SplitSeq(cfg.Proxy.Models[i].Cmd, " ") {
			if previous == "-m" || previous == "--model" {
				err := validateFile(arg)
				if err != nil {
					return err
				}
			}
			previous = arg
		}
	}
	return nil
}

func validateFile(path string) error {
	cleaned := filepath.Clean(path)
	if cleaned != path {
		slog.Warn("Malformed", "current", path, "better", cleaned)
		path = cleaned
	}

	// Check if the file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		slog.Error("Model file does not exist", "file", path)
		return err
	}

	// Check if the file is readable
	file, err := os.Open(path)
	if err != nil {
		slog.Error("Model file is not readable", "file", path)
		return err
	}

	err = file.Close()
	if err != nil {
		slog.Error("Model file fails closing", "file", path)
		return err
	}

	// is empty?
	if info.Size() < 1000 {
		slog.Error("Model file is empty (or too small)", "file", path)
		return gie.ErrConfigValidation
	}

	return nil
}
