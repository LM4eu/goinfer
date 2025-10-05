// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LM4eu/llama-swap/proxy/config"
	"go.yaml.in/yaml/v4"
)

// Helper to create a temporary configuration file.
func writeTempCfg(t *testing.T, cfg *Cfg) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// TestReadMainCfg loads a config file, applies env vars, and validates.
func TestReadMainCfg(t *testing.T) {
	// t.Parallel omitted because of t.Setenv usage.

	// Minimal config.
	cfg := Cfg{Main: DefaultMain}
	cfg.Main.ModelsDir = "/tmp/models"

	// Provide a dummy admin API key to satisfy validation.
	cfg.Main.APIKey = "dummy"

	path := writeTempCfg(t, &cfg)

	// Override via env.
	dir := filepath.Dir(path)
	t.Setenv("GI_MODELS_DIR", dir)
	t.Setenv("GI_HOST", "127.0.0.1")

	// Create a dummy model file.
	modelPath := filepath.Join(dir, "model.gguf")
	err := os.WriteFile(modelPath, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create model file: %v", err)
	}

	err = cfg.ReadMainCfg(path, true)
	if err != nil {
		t.Fatalf("ReadMainCfg failed: %v", err)
	}
	if cfg.Main.ModelsDir != dir {
		t.Errorf("ReadMainCfg did not apply GI_MODELS_DIR")
	}
	if cfg.Main.Host != "127.0.0.1" {
		t.Errorf("ReadMainCfg did not apply GI_HOST")
	}
}

// TestWriteMainCfg creates a config file and validates it.
func TestWriteMainCfg(t *testing.T) {
	// t.Parallel omitted because of t.Setenv usage.

	cfg := &Cfg{}
	modelsDir := t.TempDir()
	modelPath := filepath.Join(modelsDir, "model.gguf")
	err := os.WriteFile(modelPath, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create model file: %v", err)
	}
	t.Setenv("GI_MODELS_DIR", modelsDir)

	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.yaml")
	err = cfg.WriteMainCfg(path, false, true)
	if err != nil {
		t.Fatalf("WriteMainCfg failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("cannot read written config: %v", err)
	}
	var loaded Cfg
	err = yaml.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("written config is not valid YAML: %v", err)
	}
}

// TestWriteSwapCfg generates a swap config file.
func TestWriteSwapCfg(t *testing.T) {
	t.Parallel()
	cfg := &Cfg{}
	modelsDir := t.TempDir()
	modelPath := filepath.Join(modelsDir, "model.gguf")
	err := os.WriteFile(modelPath, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create model file: %v", err)
	}
	cfg.Main.ModelsDir = modelsDir

	tmp := t.TempDir()
	swapPath := filepath.Join(tmp, "swap.yaml")
	err = cfg.WriteSwapCfg(swapPath, false, false)
	if err != nil {
		t.Fatalf("WriteSwapCfg failed: %v", err)
	}
	_, err = os.Stat(swapPath)
	if err != nil {
		t.Fatalf("swap config not created: %v", err)
	}
}

// TestListModelsIntegration checks model discovery and swap config merging.
func TestListModelsIntegration(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	modelPath := filepath.Join(tmp, "model1.gguf")
	err := os.WriteFile(modelPath, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create model file: %v", err)
	}
	cfg := &Cfg{
		Main: GoinferYML{ModelsDir: tmp},
		Swap: config.Config{Models: map[string]config.ModelConfig{"model1": {Cmd: "", Unlisted: false}}},
	}
	models, err := cfg.ListModels()
	if err != nil {
		t.Fatalf("ListModels error: %v", err)
	}
	if info, ok := models["model1"]; !ok || info.Error != "" {
		t.Fatalf("expected model1 to be listed without error, got %v", info)
	}
}

// TestCfg_UnmarshalAndValidate verifies JSON/YAML unmarshaling and validation.
func TestCfg_UnmarshalAndValidate(t *testing.T) {
	t.Parallel()
	modelsDir := t.TempDir()
	modelPath := filepath.Join(modelsDir, "model.gguf")
	err := os.WriteFile(modelPath, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create model file: %v", err)
	}
	cfg := Cfg{Main: DefaultMain}
	cfg.Main.ModelsDir = modelsDir
	cfg.Main.APIKey = "dummy"
	err = cfg.validateMain(false)
	if err != nil {
		t.Fatalf("validation1 error: %v", err)
	}

	// JSON round‑trip.
	data, _ := json.Marshal(cfg) //nolint:errchkjson // this is a test
	var cfg2 Cfg
	err = json.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	err = cfg2.validateMain(false)
	if err != nil {
		t.Fatalf("validation2 error: %v", err)
	}
	// Missing admin key should fail.
	cfgMissing := Cfg{Main: DefaultMain}
	cfgMissing.Main.ModelsDir = modelsDir
	cfgMissing.Main.APIKey = ""
	err = cfgMissing.validateMain(false)
	if err == nil {
		t.Fatalf("expected validation error for missing admin API key")
	}
}

// TestCfg_ConcurrentReadMainCfg runs ReadMainCfg concurrently.
func TestCfg_ConcurrentReadMainCfg(t *testing.T) {
	// t.Parallel omitted because of t.Setenv usage.
	cfg := Cfg{Main: DefaultMain}
	cfg.Main.ModelsDir = t.TempDir()
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml marshal error: %v", err)
	}
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err = os.WriteFile(tmpFile, yamlData, 0o600)
	if err != nil {
		t.Fatalf("write config error: %v", err)
	}
	dir := filepath.Dir(tmpFile)
	t.Setenv("GI_MODELS_DIR", dir)
	t.Setenv("GI_HOST", "127.0.0.1")
	// Set admin API key to satisfy validation.
	t.Setenv("GI_API_KEY", "dummy")
	// Ensure a model file exists for validation.
	modelPath := filepath.Join(dir, "model.gguf")
	err = os.WriteFile(modelPath, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create model file: %v", err)
	}

	var grp sync.WaitGroup
	for i := range 10 {
		grp.Go(func() {
			var cfg Cfg
			err = cfg.ReadMainCfg(tmpFile, i&1 == 0)
			if err != nil {
				t.Errorf("#%d ReadMainCfg error: %v", i, err)
			}
			if cfg.Main.ModelsDir != dir {
				t.Errorf("#%d ModelsDir not overridden, got %q want %q", i, cfg.Main.ModelsDir, dir)
			}
			if cfg.Main.Host != "127.0.0.1" {
				t.Errorf("#%d Server.Host not overridden, got %q want 127.0.0.1", i, cfg.Main.Host)
			}
		})
	}
	grp.Wait()
}
