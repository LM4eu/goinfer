package files

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadModels(t *testing.T) {
	// Test reading models from a directory
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test files
	testFiles := []struct {
		name    string
		content string
		isDir   bool
	}{
		{"model1.bin", "test content", false},
		{"model2.gguf", "test content", false},
		{"config.yml", "test content", false},
		{"model3.safetensors", "test content", false},
		{"subdir", "", true}, // Test directory exclusion
	}

	for _, tf := range testFiles {
		if tf.isDir {
			err := os.Mkdir(filepath.Join(tempDir, tf.name), 0755)
			require.NoError(t, err)
		} else {
			filePath := filepath.Join(tempDir, tf.name)
			err := os.WriteFile(filePath, []byte(tf.content), 0644)
			require.NoError(t, err)
		}
	}

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify no error
	require.NoError(t, err)

	// Verify expected models are returned (only model files)
	expectedModels := []string{"model1.bin", "model2.gguf", "model3.safetensors"}
	assert.ElementsMatch(t, expectedModels, models)

	// Verify non-model files are not included
	assert.NotContains(t, models, "config.yml")

	// Verify directories are not included
	assert.NotContains(t, models, "subdir")
}

func TestReadModelsEmptyDirectory(t *testing.T) {
	// Test reading models from an empty directory
	tempDir := t.TempDir()

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify no error
	require.NoError(t, err)

	// Verify empty slice is returned
	assert.Empty(t, models)
}

func TestReadModelsNonExistentDirectory(t *testing.T) {
	// Test reading models from a non-existent directory
	nonExistentDir := "/path/that/does/not/exist"

	// Test ReadModels function
	models, err := ReadModels(nonExistentDir)

	// Verify error is returned
	assert.Error(t, err)

	// Verify empty slice is returned
	assert.Empty(t, models)
}

func TestReadModelsOnlyYMLFiles(t *testing.T) {
	// Test reading models from a directory with only .yml files
	tempDir := t.TempDir()

	// Create only .yml files (matching the implementation which only excludes .yml)
	testFiles := []string{
		"config1.yml",
		"settings.yml",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify no error
	require.NoError(t, err)

	// Verify empty slice is returned (no model files)
	assert.Empty(t, models)
}

func TestReadModelsMixedFiles(t *testing.T) {
	// Test reading models from a directory with mixed file types
	tempDir := t.TempDir()

	// Create test files with various extensions
	testFiles := []string{
		"model1.bin",
		"model2.gguf",
		"config.yml",
		"model3.safetensors",
		"weights.bin",
		"settings.yaml",
		"model4.pt",
		"ignored.txt",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify no error
	require.NoError(t, err)

	// Verify expected models are returned (only model files)
	expectedModels := []string{
		"model1.bin",
		"model2.gguf",
		"model3.safetensors",
		"weights.bin",
		"model4.pt",
		"settings.yaml",
		"ignored.txt",
	}
	assert.ElementsMatch(t, expectedModels, models)

	// Verify non-model files are not included
	assert.NotContains(t, models, "config.yml")
}

func TestReadModelsDirectoryWithSubdirectories(t *testing.T) {
	// Test reading models from a directory with subdirectories
	tempDir := t.TempDir()

	// Create test files in main directory
	testFiles := []string{
		"model1.bin",
		"model2.gguf",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Create subdirectory with files
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	subFiles := []string{
		"submodel1.bin",
		"submodel2.gguf",
		"subconfig.yml",
	}

	for _, file := range subFiles {
		filePath := filepath.Join(subDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify no error
	require.NoError(t, err)

	// Verify only files from main directory are returned (not recursive)
	expectedModels := []string{"model1.bin", "model2.gguf"}
	assert.ElementsMatch(t, expectedModels, models)

	// Verify subdirectory files are not included
	assert.NotContains(t, models, "submodel1.bin")
	assert.NotContains(t, models, "submodel2.gguf")
}

func TestReadModelsFileSorting(t *testing.T) {
	// Test that models are sorted by file size
	tempDir := t.TempDir()

	// Create test files with different sizes
	testFiles := []struct {
		name string
		size int64
	}{
		{"small.bin", 100},
		{"medium.bin", 1000},
		{"large.bin", 10000},
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		content := make([]byte, tf.size)
		err := os.WriteFile(filePath, content, 0644)
		require.NoError(t, err)
	}

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify no error
	require.NoError(t, err)

	// Verify models are sorted by size (ascending)
	// Check that small.bin comes before medium.bin and large.bin
	assert.Equal(t, "small.bin", models[0], "Files should be sorted by size in ascending order")
	assert.Equal(t, "medium.bin", models[1], "Files should be sorted by size in ascending order")
	assert.Equal(t, "large.bin", models[2], "Files should be sorted by size in ascending order")

	// Verify all expected files are present
	assert.Contains(t, models, "small.bin")
	assert.Contains(t, models, "medium.bin")
	assert.Contains(t, models, "large.bin")
}

func TestReadModelsPermissionError(t *testing.T) {
	// Test reading models from a directory with permission issues
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"model1.bin",
		"model2.gguf",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Remove read permission from directory
	err := os.Chmod(tempDir, 0000)
	require.NoError(t, err)

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify error is returned
	assert.Error(t, err)

	// Verify empty slice is returned
	assert.Empty(t, models)

	// Restore permissions for cleanup
	_ = os.Chmod(tempDir, 0755)
}

func TestReadModelsLargeNumberOfFiles(t *testing.T) {
	// Test reading models from a directory with many files
	tempDir := t.TempDir()

	// Create many test files
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("model%d.bin", i)
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Test ReadModels function
	models, err := ReadModels(tempDir)

	// Verify no error
	require.NoError(t, err)

	// Verify all model files are returned
	assert.Len(t, models, numFiles)

	// Verify files are sorted by size (all same size, so order may vary)
	for i := 0; i < numFiles; i++ {
		assert.Contains(t, models, fmt.Sprintf("model%d.bin", i))
	}
}
