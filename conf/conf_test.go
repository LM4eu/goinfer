// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LM4eu/llama-swap/proxy/config"
	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v4"
)

// Helper to create a temporary configuration file.
func createCfgData(t *testing.T, cfg *Cfg) []byte {
	t.Helper()
	data, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	return data
}

// TestReadMainCfg loads a config file, applies env vars, and validates.
func TestReadMainCfg(t *testing.T) {
	// t.Parallel omitted because of t.Setenv usage.

	// Minimal config.
	cfg := defaultCfg()
	cfg.ModelsDir = "/tmp/models"
	cfg.Llama.Exe = filepath.Join(t.TempDir(), "llama-server")
	err := os.WriteFile(cfg.Llama.Exe, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create dummy llama-sever file: %v", err)
	}
	cfg.APIKey = "dummy" // dummy admin API key to satisfy validation.
	data := createCfgData(t, cfg)

	// Override via env.
	dir := t.TempDir()
	t.Setenv("GI_MODELS_DIR", dir)
	t.Setenv("GI_HOST", "127.0.0.1")

	// Create a dummy model file.
	modelDir := filepath.Join(dir, "model.gguf")
	err = os.WriteFile(modelDir, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create model file: %v", err)
	}

	cfg2, err := ReadFileData(data, true, "", "")
	if err != nil {
		t.Fatalf("ReadMainCfg failed: %v", err)
	}
	if cfg2.ModelsDir != dir {
		t.Errorf("ReadMainCfg did not apply GI_MODELS_DIR")
	}
	if cfg2.Host != "127.0.0.1" {
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
	llamaExe := filepath.Join(t.TempDir(), "llama-server")
	err = os.WriteFile(llamaExe, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create dummy llama-sever file: %v", err)
	}

	t.Setenv("GI_MODELS_DIR", modelsDir)
	t.Setenv("GI_LLAMA_EXE", llamaExe)

	data, err := cfg.GenFileData(false, true)
	if err != nil {
		t.Fatalf("WriteMainCfg failed: %v", err)
	}
	var loaded Cfg
	err = toml.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("written config is not valid TOML: %v", err)
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
	cfg.ModelsDir = modelsDir

	ymlData, err := cfg.GenSwapYAMLData(false, false)
	if err != nil {
		t.Fatalf("WriteSwapCfg failed: %v ymlData=%s", err, string(ymlData))
	}

	err = yaml.Unmarshal(ymlData, &cfg.Swap)
	if err != nil {
		t.Logf("ymlData:\n---\n%s\n---\n", string(ymlData))
		t.Fatal(err.Error())
	}

	err = cfg.ValidateSwap()
	if err != nil {
		t.Fatalf("cfg.ValidateSwap error: %v", err)
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
		ModelsDir: tmp,
		Swap:      config.Config{Models: map[string]config.ModelConfig{"model1": {Cmd: "", Unlisted: false}}},
	}
	models := cfg.ListModels()
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
		t.Fatalf("cannot create dummy model file: %v", err)
	}
	llamaExe := filepath.Join(modelsDir, "llama-server")
	err = os.WriteFile(llamaExe, make([]byte, 2048), 0o600)
	if err != nil {
		t.Fatalf("cannot create dummy llama-sever file: %v", err)
	}

	cfg := defaultCfg()
	cfg.ModelsDir = modelsDir
	cfg.APIKey = "dummy"
	cfg.Llama.Exe = llamaExe

	err = cfg.validate(false)
	if err != nil {
		t.Fatalf("validation1 error: %v", err)
	}

	// JSON round-trip.
	data, _ := toml.Marshal(cfg) //nolint:errchkjson // this is a test
	var cfg2 Cfg
	err = toml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	err = cfg2.validate(false)
	if err != nil {
		t.Fatalf("validation2 error: %v", err)
	}
	// Missing admin key should fail.
	cfgMissing := defaultCfg()
	cfgMissing.ModelsDir = modelsDir
	cfgMissing.APIKey = ""
	err = cfgMissing.validate(false)
	if err == nil {
		t.Fatalf("expected validation error for missing admin API key")
	}
}

// TestCfg_ConcurrentReadMainCfg runs ReadMainCfg concurrently.
func TestCfg_ConcurrentReadMainCfg(t *testing.T) {
	// t.Parallel omitted because of t.Setenv usage.
	cfg := defaultCfg()
	cfg.ModelsDir = t.TempDir()
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml marshal error: %v", err)
	}
	dir := t.TempDir()
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

	t.Setenv("GI_LLAMA_EXE", modelPath) // dummy llama-server

	var grp sync.WaitGroup
	for i := range 30 {
		grp.Go(func() {
			cfg, err := ReadFileData(yamlData, i&1 == 0, "", "")
			if err != nil {
				t.Errorf("#%d ReadMainCfg error: %v", i, err)
			}
			if cfg.ModelsDir != dir {
				t.Errorf("#%d ModelsDir not overridden, got %q want %q", i, cfg.ModelsDir, dir)
			}
			if cfg.Host != "127.0.0.1" {
				t.Errorf("#%d Server.Host not overridden, got %q want 127.0.0.1", i, cfg.Host)
			}
		})
	}
	grp.Wait()
}
