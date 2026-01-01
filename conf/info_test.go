// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LynxAIeu/goinfer/proxy/config"
)

// createGGUFFile creates a temporary .gguf file of the given size (bytes).
func createGGUFFile(t *testing.T, dir, name string, size int64) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
	defer f.Close()
	if size > 0 {
		_, err = f.Write(make([]byte, size))
		if err != nil {
			t.Fatalf("failed to write to file %s: %v", path, err)
		}
	}
	return path
}

func TestListModels(t *testing.T) {
	t.Parallel()

	cfg := &Cfg{
		ModelsDir: "/home/me/models",
		Swap: &config.Config{
			Models: map[string]config.ModelConfig{
				"disk-model":  {Cmd: "llama-server -flag", Unlisted: false},
				"missing":     {Cmd: "llama-server -flag -m missing.gguf", Unlisted: false},
				A_ + "hidden": {Cmd: "llama-server -flag -m mistral.gguf", Unlisted: true},
			},
		},
	}
	models := cfg.ListModels()
	if info, ok := models["disk-model"]; !ok || info.Issue != "" {
		t.Errorf("disk-model missing or error: %v", info)
	}
	if info, ok := models["missing"]; !ok || !strings.Contains(info.Issue, "file absent") {
		t.Errorf("missing entry error not as expected: %v", info)
	}
	if _, ok := models[A_+"hidden"]; ok {
		t.Errorf(A_ + "hidden should not be listed")
	}
}

func TestCountModels(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	createGGUFFile(t, tmp, "a.gguf", 2048)
	createGGUFFile(t, tmp, "b.gguf", 2048)

	cfg := &Cfg{
		ModelsDir: tmp,
		Swap:      nil,
	}
	if n := len(cfg.getInfo()); n != 2 {
		t.Errorf("len(cfg.getInfo()) = %d, want 2", n)
	}
}
