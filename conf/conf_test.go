package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func init() {
	// Reset viper before each test
	viper.Reset()
}

// resetViper resets viper state to ensure clean test environment
func resetViper() {
	viper.Reset()
}

func TestInitConf(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.yml")

	configData := map[string]any{
		"models_dir":        "./test_models",
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
	config, _ := Load(".", "goinfer") // ./goinfer.yml

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
		"models_dir": "./test_models",
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
	config, _ := Load(".", "goinfer") // ./goinfer.yml

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
	_, err = Load(".", "goinfer") // ./goinfer.yml
	assert.Error(t, err)
}

func TestInitConf_InvalidJSON(t *testing.T) {
	// Create a temporary config file with invalid JSON
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.yml")

	invalidJSON := `{"models_dir": "./test_models",` // Missing closing brace
	err := os.WriteFile(configPath, []byte(invalidJSON), 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test InitConf with invalid JSON
	_, err = Load(".", "goinfer") // ./goinfer.yml
	assert.Error(t, err)
}

func TestInitConf_DifferentConfigName(t *testing.T) {
	// Create a temporary config file with different name
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom.config.json")

	configData := map[string]any{
		"models_dir":     "./test_models",
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
	_, err = Load(".", "goinfer") // ./goinfer.yml
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
	t.Setenv("MODELS_DIR", "/test/models")
	err = Create(customFileName, false)
	require.NoError(t, err)

	// Verify config file was created with custom name
	assert.FileExists(t, customFileName)

	// Read and verify config content
	configBytes, err := os.ReadFile(customFileName)
	require.NoError(t, err)

	var config map[string]any
	err = yaml.Unmarshal(configBytes, &config)
	require.NoError(t, err)

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

	// Test Create with random=true
	t.Setenv("MODELS_DIR", "/test/models")
	err = Create("goinfer.yml", false)
	require.NoError(t, err)

	// Verify config file was created
	assert.FileExists(t, "goinfer.yml")

	// Read and verify config content
	configBytes, err := os.ReadFile("goinfer.yml")
	require.NoError(t, err)

	var config map[string]any
	err = yaml.Unmarshal(configBytes, &config)
	require.NoError(t, err)

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

func TestInitConf_EnvironmentVariablePrecedence(t *testing.T) {
	// Reset viper to ensure clean test environment
	resetViper()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.yml")

	configData := map[string]any{
		"models_dir":        "./test_models",
		"llama.exe_path":    "./config-value", // This should be overridden by env var
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

	// Set environment variable before InitConf
	t.Setenv("LLAMA_EXE_PATH", "/custom/path/llama")
	t.Setenv("MODELS_DIR", "/custom/models") // Test top-level env var

	// Debug: Check environment variables
	fmt.Printf("Debug: LLAMA_EXE_PATH=%s\n", os.Getenv("LLAMA_EXE_PATH"))
	fmt.Printf("Debug: MODELS_DIR=%s\n", os.Getenv("MODELS_DIR"))

	// Test InitConf
	config, err := Load(".", "goinfer") // ./goinfer.yml
	require.NoError(t, err)

	// Debug: Check the actual values
	fmt.Printf("Debug: config.ModelsDir=%s\n", config.ModelsDir)
	fmt.Printf("Debug: config.Llama.ExePath=%s\n", config.Llama.ExePath)

	// Environment variable should override config file value
	assert.Equal(t, "/custom/path/llama", config.Llama.ExePath)

	// Environment variable should override config file value
	assert.Equal(t, "/custom/models", config.ModelsDir)

	// Config file value should be used when no env var is set
	assert.Equal(t, []string{"http://localhost:3000"}, config.Server.Origins)
	assert.Equal(t, "test_key_123", config.Server.ApiKey)
	assert.True(t, config.Server.EnableOpenAiAPI)
}

func TestInitConf_EnvironmentVariableNaming(t *testing.T) {
	// Reset viper to ensure clean test environment
	resetViper()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "goinfer.yml")

	configData := map[string]any{
		"models_dir": "./test_models",
	}

	configBytes, _ := yaml.Marshal(configData)
	err := os.WriteFile(configPath, configBytes, 0o644)
	require.NoError(t, err)

	// Change working directory to temp dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test incorrect naming with double underscores (should not work)
	t.Setenv("LLAMA__EXE_PATH", "/wrong/path")

	config1, err := Load(".", "goinfer")
	require.NoError(t, err)
	assert.Equal(t, "./llama-server", config1.Llama.ExePath) // Should use default

	// Test correct naming (should work)
	t.Setenv("LLAMA_EXE_PATH", "/correct/path")

	config2, err := Load(".", "goinfer")
	require.NoError(t, err)
	assert.Equal(t, "/correct/path", config2.Llama.ExePath) // Should use env var
}
