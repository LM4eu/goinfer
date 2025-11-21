// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LM4eu/goinfer/proxy/config"
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

func TestUnderlineToSlash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"team-org_model_name", "team-org/model_name"},
		{"model_name", "model/name"},
		{"model-name", "model-name"},
		{"abcdefgh_fr_8", "abcdefgh/fr_8"},
		{"abcdefghi_fr_9", "abcdefghi/fr_9"},
		{"abcdefghij_fr_10", "abcdefghij_fr_10"},
		{"abcdefghijk_fr_11", "abcdefghijk_fr_11"},
		{"abcdefghijkl_fr_12", "abcdefghijkl_fr_12"},
		{"UI_WEB_name", "UI_WEB_name"},
		{"model1_2", "model1_2"},
		{"model-1-2", "model-1-2"},
		{"1234567890", "1234567890"},
		{"_llama-1", "_llama-1"},
		{"t_llama-1", "t_llama-1"},
		{"te_llama-1", "te_llama-1"},
		{"tea_llama-1", "tea_llama-1"},
		{"team_llama-1", "team/llama-1"},
		{"-_llama-1", "-_llama-1"},
		{"-abcd_llama-1", "-abcd_llama-1"},
		{"x-c_llama-1", "x-c_llama-1"},
		{"x-co_llama-1", "x-co_llama-1"},
		{"ab-fr_llama-1", "ab-fr_llama-1"},
		{"abc-fr_llama-1", "abc-fr_llama-1"},
		{"abcd-fr_llama-1", "abcd-fr/llama-1"},
		{"/home/me/models/abcd-fr_llama-1", "abcd-fr/llama-1"},
		{"/home/me/models/group/abcd-fr_llama-1", "abcd-fr/llama-1"},
		{"/home/me/models/group/abcd-f_llama-1", "group/abcd-f_llama-1"},
		{"/home/me/models/30b/abcd-f_llama-1", "abcd-f_llama-1"},
		{"/home/me/models/mistral-ai/abcd-f_llama-1", "mistral-ai/abcd-f_llama-1"},
		{"/home/me/models/mistral-ai/mistral-ai_llama-1", "mistral-ai/llama-1"},
		{"/home/me/models/sub/rolex/granite3.3_8b_Q4_K_M", "rolex/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/rolex/granite3.3_8b_Q4_K_M", "rolex/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/granite3.3_8b_Q4_K_M", "granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-com_granite3.3_8b_Q4_K_M", "example-com/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-eu_granite3.3_8b_Q4_K_M", "example-eu/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-x_granite3.3_8b_Q4_K_M", "folder/example-x_granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-four_granite3.3_8b_Q4_K_M", "folder/example-four_granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-four/granite3.3_8b_Q4_K_M", "granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-com/granite3.3_8b_Q4_K_M", "example-com/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-fr/granite3.3_8b_Q4_K_M", "example-fr/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example-o/granite3.3_8b_Q4_K_M", "granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/example/granite3.3_8b_Q4_K_M", "example/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/folder/granite3.3_8b_Q4_K_M", "folder/granite3.3_8b_Q4_K_M"},
		{"/home/me/models/granite3.3_8b_Q4_K_M", "granite3.3_8b_Q4_K_M"},
	}
	for _, tt := range tests {
		got := nameWithSlash("/home/me/models", tt.in)
		if got != tt.want {
			t.Errorf("underlineToSlash(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExtractFlags(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// .args file present
	modelPath := createGGUFFile(t, tmp, "model1.gguf", 2048)
	argsPath := strings.TrimSuffix(modelPath, ".gguf") + ".args"
	err := os.WriteFile(argsPath, []byte("-foo bar -baz qux"), 0o600)
	if err != nil {
		t.Fatalf("failed to write args file: %v", err)
	}
	name, flags := extractFlags(modelPath)
	if !strings.HasSuffix(name, "model1") {
		t.Errorf("extractFlags name = %q, want suffix %q", name, "model1")
	}
	if strings.TrimSpace(flags) != "-foo bar -baz qux" {
		t.Errorf("extractFlags flags = %q, want %q (ignoring trailing space)", flags, "-foo bar -baz qux")
	}

	// Flags encoded in filename
	modelPath2 := filepath.Join(tmp, "model2&foo=1&bar=2.gguf")
	createGGUFFile(t, tmp, "model2&foo=1&bar=2.gguf", 2048)
	name2, flags2 := extractFlags(modelPath2)
	if name2 == "" {
		t.Errorf("extractFlags name2 is empty")
	}
	expected := "-foo 1 -bar 2"
	if strings.TrimSpace(flags2) != expected {
		t.Errorf("extractFlags flags2 = %q, want %q (ignoring trailing space)", flags2, expected)
	}

	// No flags
	modelPath3 := createGGUFFile(t, tmp, "plain_model.gguf", 2048)
	name3, flags3 := extractFlags(modelPath3)
	if !strings.HasSuffix(name3, "plain_model") {
		t.Errorf("extractFlags name3 = %q, want suffix %q", name3, "plain_model")
	}
	if flags3 != "" {
		t.Errorf("extractFlags flags3 = %q, want empty", flags3)
	}
}

func TestGetNameAndFlags(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	modelPath := createGGUFFile(t, tmp, "my_model&opt=val.gguf", 2048)
	name, flags := getNameAndFlags(tmp, modelPath)
	if name == "" {
		t.Errorf("getNameAndFlags name is empty")
	}
	if strings.TrimSpace(flags) != "-opt val" {
		t.Errorf("getNameAndFlags flags = %q, want %q (ignoring trailing space)", flags, "-opt val")
	}
}

func TestListModels(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	// model on disk
	_ = createGGUFFile(t, tmp, "disk-model.gguf", 2048)

	cfg := &Cfg{
		ModelsDir: tmp,
		Swap: config.Config{
			Models: map[string]config.ModelConfig{
				"disk-model": {Cmd: "llama-server -flag", Unlisted: false},
				"missing":    {Cmd: "llama-server -flag -m missing.gguf", Unlisted: false},
				"GI_hidden":  {Cmd: "llama-server -flag -m mistral.gguf", Unlisted: true},
			},
		},
	}
	models := cfg.ListModels()
	if info, ok := models["disk-model"]; !ok || info.Error != "" {
		t.Errorf("disk-model missing or error: %v", info)
	}
	if info, ok := models["missing"]; !ok || !strings.Contains(info.Error, "file absent") {
		t.Errorf("missing entry error not as expected: %v", info)
	}
	if _, ok := models["GI_hidden"]; ok {
		t.Errorf("GI_hidden should not be listed")
	}
}

func TestCountModels(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	createGGUFFile(t, tmp, "a.gguf", 2048)
	createGGUFFile(t, tmp, "b.gguf", 2048)

	cfg := &Cfg{
		ModelsDir: tmp,
		Swap:      config.Config{},
	}
	if n := cfg.countModels(); n != 2 {
		t.Errorf("countModels = %d, want 2", n)
	}
}

func TestValidateFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// valid
	valid := createGGUFFile(t, tmp, "valid.gguf", 2048)
	size, err := verify(valid)
	if err != nil {
		t.Errorf("validateFile(valid) error: %v", err)
	}
	if size != 2048 {
		t.Errorf("file size: got %d want 2048", size)
	}

	// too small
	small := createGGUFFile(t, tmp, "small.gguf", 64)
	_, err = verify(small)
	if err == nil {
		t.Errorf("validateFile(small) expected error")
	}

	// series first part
	firstSeries := createGGUFFile(t, tmp, "model-00001-of-00003.gguf", 2048)
	_, err = verify(firstSeries)
	if err != nil {
		t.Errorf("validateFile(firstSeries) error: %v", err)
	}

	// series non-first part
	secondSeries := createGGUFFile(t, tmp, "model-00002-of-00003.gguf", 2048)
	_, err = verify(secondSeries)
	if err == nil {
		t.Errorf("validateFile(secondSeries) expected error")
	}
}

func TestValidateModelFiles_NoSwapModels(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cfg := &Cfg{
		ModelsDir: tmp,
		Swap:      config.Config{},
	}
	err := cfg.ValidateSwap()
	if err == nil {
		t.Fatalf("validateModelFiles should error when no models and no swap config")
	}
}

func TestValidateModelFiles_WithSwapModels(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	modelPath := createGGUFFile(t, tmp, "ref.gguf", 2048)

	cfg := &Cfg{
		ModelsDir: tmp,
		Swap: config.Config{
			Models: map[string]config.ModelConfig{
				"ref": {Cmd: "--model " + modelPath, Unlisted: false},
			},
		},
	}
	err := cfg.ValidateSwap()
	if err != nil {
		t.Fatalf("validateModelFiles error: %v", err)
	}
}
