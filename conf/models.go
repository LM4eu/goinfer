// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

func (cfg *GoInferCfg) Search() ([]string, error) {
	var modelFiles []string

	for root := range strings.SplitSeq(cfg.ModelsDir, ":") {
		err := cfg.search(&modelFiles, strings.TrimSpace(root))
		if err != nil {
			if cfg.Verbose {
				fmt.Println("INF: Searching model files in:", root)
			}
			return nil, fmt.Errorf("failed to search in '%s': %w", root, err)
		}
	}

	return modelFiles, nil
}

func (cfg *GoInferCfg) search(files *[]string, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil // => step into this directory
		}

		if strings.HasSuffix(path, ".gguf") {
			if cfg.Verbose {
				fmt.Println("INF: Found model:", path)
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
