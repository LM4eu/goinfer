// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/gie"
)

// Search returns a slice of absolute file paths for all *.gguf model files
// found under the directories listed in cfg.ModelsDir (colon-separated).
// It walks each directory recursively, aggregates matching files, and returns any error encountered.
func (cfg *GoInferCfg) Search() ([]string, error) {
	modelFiles := make([]string, len(cfg.ModelsDir)/2)

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

func (cfg *GoInferCfg) search(files *[]string, root string) error {
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
			*files = append(*files, path)
		}

		return nil
	})
}

// extractFlags extracts flags from a model filename.
// It looks for a pattern starting with "&" and splits the remaining string by "&"
// to get individual flag components.
// Each component is then split by "=" to separate key and value,
// with the key prefixed by "-" to form command-line style flags.
// Returns a single string with flags separated by spaces.
func extractFlags(stem string) string {
	pos := strings.Index(stem, "&")
	if pos < 0 {
		return ""
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

	return strings.Join(flags, " ")
}
