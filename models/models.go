// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package models

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/state"
	"github.com/labstack/echo/v4"
)

type Dir string

// Handler returns the state of models.
func (dir Dir) Handler(c echo.Context) error {
	models, err := dir.Search()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("failed to search models: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"models": models,
		"count":  len(models),
	})
}

// ExtractFlags extracts flags from a model filename.
// It looks for a pattern starting with "&" and splits the remaining string
// by "&" to get individual flag components. Each component is then split
// by "=" to separate key and value, with the key prefixed by "-" to form
// command-line style flags. Returns a slice of strings representing the extracted flags.
func ExtractFlags(modelStem string) string {
	var flags []string

	p := strings.Index(modelStem, "&")
	if p < 0 {
		return ""
	}

	for f := range strings.SplitSeq(modelStem[p:], "&") {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) > 0 {
			kv[0] = "-" + kv[0]
			flags = append(flags, kv...)
		}
	}

	return strings.Join(flags, " ")
}

func (dir Dir) Search() ([]string, error) {
	var modelFiles []string

	for root := range strings.SplitSeq(string(dir), ":") {
		err := search(&modelFiles, strings.TrimSpace(root))
		if err != nil {
			if state.Verbose {
				fmt.Println("INF: Searching model files in:", root)
			}
			return nil, fmt.Errorf("failed to search in '%s': %w", root, err)
		}
	}

	return modelFiles, nil
}

func search(files *[]string, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil // => step into this directory
		}

		if strings.HasSuffix(path, ".gguf") {
			if state.Verbose {
				fmt.Println("INF: Found model:", path)
			}
			*files = append(*files, path)
		}

		return nil
	})
}
