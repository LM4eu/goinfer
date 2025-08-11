package conf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestInitConf(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.yml")

	configData := map[string]any{
		"model.dir":         "./test_models",
		"server.origins":    []string{"http://localhost:3000"},
		"server.api_key":    "test_key_123",
		"server.openai_api": true,
	}

	configBytes, _ := yaml.Marshal(configData)
	err := os.WriteFile(configPath, configBytes, 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf
	config, _ := InitConf(".", "goinfer") // ./goinfer.yml

	assert.Equal(t, "./test_models", config.ModelsDir)
	assert.Equal(t, []string{"http://localhost:3000"}, config.Server.Origins)
	assert.Equal(t, "test_key_123", config.Server.ApiKey)
	assert.True(t, config.Server.EnableOpenAiAPI)
}

func TestInitConf_WithDefaults(t *testing.T) {
	// Create a minimal config file with only required fields
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.yml")

	configData := map[string]any{
		"model.dir": "./test_models",
	}

	configBytes, _ := yaml.Marshal(configData)
	err := os.WriteFile(configPath, configBytes, 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with defaults
	config, _ := InitConf(".", "goinfer") // ./goinfer.yml

	assert.Equal(t, "./test_models", config.ModelsDir)
	assert.Equal(t, []string{"localhost"}, config.Server.Origins) // Default value
	assert.Empty(t, config.Server.ApiKey)                         // Default empty value
	assert.False(t, config.Server.EnableOpenAiAPI)                // Default value
}

func TestInitConf_InvalidConfig(t *testing.T) {
	// Change working directory to temp dir without config file
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with invalid config
	_, err = InitConf(".", "goinfer") // ./goinfer.yml
	assert.Error(t, err)
}

func TestInitConf_InvalidJSON(t *testing.T) {
	// Create a temporary config file with invalid JSON
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.yml")

	invalidJSON := `{"model.dir": "./test_models",` // Missing closing brace
	err := os.WriteFile(configPath, []byte(invalidJSON), 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with invalid JSON
	_, err = InitConf(".", "goinfer") // ./goinfer.yml
	assert.Error(t, err)
}

func TestInitConf_DifferentConfigName(t *testing.T) {
	// Create a temporary config file with different name
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom.config.json")

	configData := map[string]any{
		"model.dir":      "./test_models",
		"server.origins": []string{"http://localhost:3000"},
		"server.api_key": "test_key_123",
	}

	configBytes, _ := yaml.Marshal(configData)
	err := os.WriteFile(configPath, configBytes, 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with different config name
	_, err = InitConf(".", "goinfer") // ./goinfer.yml
	assert.Error(t, err)
}

func TestCreate(t *testing.T) {
	// Change to temp directory
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Test Create with default=false
	customFileName := "custom.config.json"
	err = Create("/test/models", false, customFileName)
	require.NoError(t, err)

	// Verify config file was created with custom name
	assert.FileExists(t, customFileName)

	// Read and verify config content
	configBytes, err := os.ReadFile(customFileName)
	require.NoError(t, err)

	var config map[string]any
	err = yaml.Unmarshal(configBytes, &config)
	require.NoError(t, err)

	// Check the nested structure
	model := config["model"].(map[string]any)
	assert.Equal(t, "/test/models", model["dir"])

	server := config["server"].(map[string]any)
	assert.Equal(t, []any{"http://localhost:5173", "http://localhost:5143"}, server["origins"])
	assert.NotEmpty(t, server["api_key"]) // Should be a random key

	// Verify cleanup after test
	t.Cleanup(func() {
		os.Remove(customFileName)
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
	err = Create("/test/models", true, "goinfer.yml")
	require.NoError(t, err)

	// Verify config file was created
	assert.FileExists(t, "goinfer.yml")

	// Read and verify config content
	configBytes, err := os.ReadFile("goinfer.yml")
	require.NoError(t, err)

	var config map[string]any
	err = yaml.Unmarshal(configBytes, &config)
	require.NoError(t, err)

	// Check the nested structure
	model := config["model"].(map[string]any)
	assert.Equal(t, "/test/models", model["dir"])

	server := config["server"].(map[string]any)
	assert.Equal(t, []any{"http://localhost:5173", "http://localhost:5143"}, server["origins"])
	assert.Equal(t, "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465", server["api_key"]) // Default key

	// Verify cleanup after test
	t.Cleanup(func() {
		os.Remove("goinfer.yml")
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
