package conf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitConf(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.json")

	configData := map[string]any{
		"models_dir": "./test_models",
		"tasks_dir":  "./test_tasks",
		"origins":    []string{"http://localhost:3000"},
		"api_key":    "test_key_123",
		"oai": map[string]any{
			"enable":   true,
			"threads":  4,
			"template": "{system}\n\n{prompt}",
		},
	}

	configBytes, _ := json.MarshalIndent(configData, "", "    ")
	err := os.WriteFile(configPath, configBytes, 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf
	config := InitConf(".", "goinfer") // ./goinfer.json

	assert.Equal(t, "./test_models", config.ModelsDir)
	assert.Equal(t, "./test_tasks", config.TasksDir)
	assert.Equal(t, []string{"http://localhost:3000"}, config.Origins)
	assert.Equal(t, "test_key_123", config.ApiKey)
	assert.True(t, config.OpenAiConf.Enable)
	assert.Equal(t, 4, config.OpenAiConf.Threads)
	assert.Equal(t, "{system}\n\n{prompt}", config.OpenAiConf.Template)
}

func TestInitConf_WithDefaults(t *testing.T) {
	// Create a minimal config file with only required fields
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.json")

	configData := map[string]any{
		"models_dir": "./test_models",
	}

	configBytes, _ := json.MarshalIndent(configData, "", "    ")
	err := os.WriteFile(configPath, configBytes, 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with defaults
	config := InitConf(".", "goinfer") // ./goinfer.json

	assert.Equal(t, "./test_models", config.ModelsDir)
	assert.Equal(t, "./tasks", config.TasksDir)                         // Default value should be set
	assert.Equal(t, []string{"localhost"}, config.Origins)              // Default value
	assert.Empty(t, config.ApiKey)                                      // Default empty value
	assert.False(t, config.OpenAiConf.Enable)                           // Default value
	assert.Equal(t, 4, config.OpenAiConf.Threads)                       // Default value
	assert.Equal(t, "{system}\n\n{prompt}", config.OpenAiConf.Template) // Default value
}

func TestInitConf_InvalidConfig(t *testing.T) {
	// Change working directory to temp dir without config file
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with invalid config (should panic)
	assert.Panics(t, func() {
		InitConf(".", "goinfer") // ./goinfer.json
	})
}

func TestInitConf_InvalidJSON(t *testing.T) {
	// Create a temporary config file with invalid JSON
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.json")

	invalidJSON := `{"models_dir": "./test_models", "tasks_dir": "./test_tasks",` // Missing closing brace
	err := os.WriteFile(configPath, []byte(invalidJSON), 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with invalid JSON (should panic)
	assert.Panics(t, func() {
		InitConf(".", "goinfer") // ./goinfer.json
	})
}

func TestInitConf_DifferentConfigName(t *testing.T) {
	// Create a temporary config file with different name
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom.config.json")

	configData := map[string]any{
		"models_dir": "./test_models",
		"tasks_dir":  "./test_tasks",
		"origins":    []string{"http://localhost:3000"},
		"api_key":    "test_key_123",
	}

	configBytes, _ := json.MarshalIndent(configData, "", "    ")
	err := os.WriteFile(configPath, configBytes, 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with different config name (should panic as it looks for "goinfer.json")
	assert.Panics(t, func() {
		InitConf(".", "goinfer") // ./goinfer.json
	})
}

func TestCreate(t *testing.T) {
	// Change to temp directory
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Test Create with default=false
	Create("/test/models", false)

	// Verify config file was created
	assert.FileExists(t, "goinfer.json")

	// Read and verify config content
	configBytes, err := os.ReadFile("goinfer.json")
	require.NoError(t, err)

	var config map[string]any
	err = json.Unmarshal(configBytes, &config)
	require.NoError(t, err)

	assert.Equal(t, "/test/models", config["models_dir"])
	assert.Equal(t, []any{"http://localhost:5173", "http://localhost:5143"}, config["origins"])
	assert.Equal(t, "./tasks", config["tasks_dir"])
	assert.NotEmpty(t, config["api_key"]) // Should be a random key

	// Verify cleanup after test
	t.Cleanup(func() {
		os.Remove("goinfer.json")
	})
}

func TestCreate_WithDefaults(t *testing.T) {
	// Change to temp directory
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Test Create with default=true
	Create("/test/models", true)

	// Verify config file was created
	assert.FileExists(t, "goinfer.json")

	// Read and verify config content
	configBytes, err := os.ReadFile("goinfer.json")
	require.NoError(t, err)

	var config map[string]any
	err = json.Unmarshal(configBytes, &config)
	require.NoError(t, err)

	assert.Equal(t, "/test/models", config["models_dir"])
	assert.Equal(t, []any{"http://localhost:5173", "http://localhost:5143"}, config["origins"])
	assert.Equal(t, "./tasks", config["tasks_dir"])
	assert.Equal(t, "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465", config["api_key"]) // Default key

	// Verify cleanup after test
	t.Cleanup(func() {
		os.Remove("goinfer.json")
	})
}

func TestGenerateRandomKey(t *testing.T) {
	key := generateRandomKey()

	// Verify key format (should be 64 characters hex string)
	assert.Len(t, key, 64)
	assert.Regexp(t, "^[a-f0-9]+$", key) // Should be hex characters only

	// Verify different calls produce different keys
	key2 := generateRandomKey()
	assert.NotEqual(t, key, key2)
}

func TestGenerateRandomKey_WithFixedSeed(t *testing.T) {
	// This test verifies the function works with a fixed seed
	// In a real scenario, you might want to mock crypto/rand
	assert.NotPanics(t, func() { generateRandomKey() })

	key := generateRandomKey()
	assert.Len(t, key, 64)
	assert.Regexp(t, "^[a-f0-9]+$", key)

	// Test that multiple calls produce different keys
	keys := make(map[string]bool)

	for range 10 {
		k := generateRandomKey()
		assert.False(t, keys[k], "Duplicate key generated")
		keys[k] = true
	}
}

func TestCreateWithFileName(t *testing.T) {
	// Change to temp directory
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Test CreateWithFileName with default=false
	customFileName := "custom.config.json"
	CreateWithFileName("/test/models", false, customFileName)

	// Verify config file was created with custom name
	assert.FileExists(t, customFileName)

	// Read and verify config content
	configBytes, err := os.ReadFile(customFileName)
	require.NoError(t, err)

	var config map[string]any
	err = json.Unmarshal(configBytes, &config)
	require.NoError(t, err)

	assert.Equal(t, "/test/models", config["models_dir"])
	assert.Equal(t, []any{"http://localhost:5173", "http://localhost:5143"}, config["origins"])
	assert.Equal(t, "./tasks", config["tasks_dir"])
	assert.NotEmpty(t, config["api_key"]) // Should be a random key

	// Verify cleanup after test
	t.Cleanup(func() {
		os.Remove(customFileName)
	})
}
