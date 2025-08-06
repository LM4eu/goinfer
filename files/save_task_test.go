package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
	"github.com/stretchr/testify/assert"
)

func TestSaveTask(t *testing.T) {
	// Create a temporary directory for tasks
	tempDir := t.TempDir()
	
	// Save the original TasksDir
	originalTasksDir := state.TasksDir
	// Set the tasks directory for testing
	state.TasksDir = tempDir
	// Restore the original TasksDir after the test
	defer func() {
		state.TasksDir = originalTasksDir
	}()
	
	// Create test task
	task := types.Task{
		Name:     "test_task",
		Template: "{prompt}",
		ModelConf: types.ModelConf{
			Name: "test_model",
			Ctx:  2048,
		},
		InferParams: types.InferenceParams{
			Stream:      false,
			Threads:     4,
			NPredict:    100,
			TopK:        40,
			TopP:        0.9,
			Temperature: 0.7,
		},
	}
	
	// Test SaveTask
	err := SaveTask(task)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Verify file was created
	expectedPath := filepath.Join(tempDir, "test_task.yml")
	assert.FileExists(t, expectedPath)
	
	// Read and verify file content
	fileContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	
	// The content should be a valid YAML representation of the task
	assert.Contains(t, string(fileContent), "name: test_task.yml")
	assert.Contains(t, string(fileContent), "template: '{prompt}'")
	assert.Contains(t, string(fileContent), "name: test_model")
	assert.Contains(t, string(fileContent), "ctx: 2048")
}

func TestSaveTask_WithSubdirectory(t *testing.T) {
	// Create a temporary directory for tasks
	tempDir := t.TempDir()
	
	// Save the original TasksDir
	originalTasksDir := state.TasksDir
	// Set the tasks directory for testing
	state.TasksDir = tempDir
	// Restore the original TasksDir after the test
	defer func() {
		state.TasksDir = originalTasksDir
	}()
	
	// Create test task with subdirectory path
	task := types.Task{
		Name:     "subdir/test_task",
		Template: "{system}\n\n{prompt}",
		ModelConf: types.ModelConf{
			Name: "test_model",
			Ctx:  4096,
		},
		InferParams: types.InferenceParams{
			Stream:      true,
			Threads:     8,
			NPredict:    200,
			TopK:        50,
			TopP:        0.95,
			Temperature: 0.8,
		},
	}
	
	// Test SaveTask
	err := SaveTask(task)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Verify file was created in subdirectory
	expectedPath := filepath.Join(tempDir, "subdir", "test_task.yml")
	assert.FileExists(t, expectedPath)
	
	// Verify subdirectory was created
	subdirPath := filepath.Join(tempDir, "subdir")
	assert.DirExists(t, subdirPath)
	
	// Read and verify file content
	fileContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	
	// The content should be a valid YAML representation of the task
	assert.Contains(t, string(fileContent), "name: test_task.yml")
	assert.Contains(t, string(fileContent), "template: |-\n    {system}\n\n    {prompt}")
	assert.Contains(t, string(fileContent), "name: test_model")
	assert.Contains(t, string(fileContent), "ctx: 4096")
	assert.Contains(t, string(fileContent), "stream: true")
}

func TestSaveTask_NestedSubdirectory(t *testing.T) {
	// Create a temporary directory for tasks
	tempDir := t.TempDir()
	
	// Save the original TasksDir
	originalTasksDir := state.TasksDir
	// Set the tasks directory for testing
	state.TasksDir = tempDir
	// Restore the original TasksDir after the test
	defer func() {
		state.TasksDir = originalTasksDir
	}()
	
	// Create test task with deeply nested subdirectory path
	task := types.Task{
		Name:     "level1/level2/level3/deep_task",
		Template: "Custom template: {prompt}",
		ModelConf: types.ModelConf{
			Name:      "deep_model",
			Ctx:       8192,
			GPULayers: 1,
		},
		InferParams: types.InferenceParams{
			Stream:            false,
			Threads:           16,
			NPredict:          500,
			TopK:              100,
			TopP:              0.99,
			Temperature:       1.0,
			FrequencyPenalty:  0.1,
			PresencePenalty:   0.1,
			RepeatPenalty:     1.1,
			TailFreeSamplingZ: 0.5,
			StopPrompts:       []string{"\n", "User:"},
		},
	}
	
	// Test SaveTask
	err := SaveTask(task)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Verify file was created in nested subdirectory
	expectedPath := filepath.Join(tempDir, "level1", "level2", "level3", "deep_task.yml")
	assert.FileExists(t, expectedPath)
	
	// Verify all nested subdirectories were created
	for _, level := range []string{"level1", "level1/level2", "level1/level2/level3"} {
		subdirPath := filepath.Join(tempDir, level)
		assert.DirExists(t, subdirPath)
	}
	
	// Read and verify file content
	fileContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	
	// The content should be a valid YAML representation of the task
	assert.Contains(t, string(fileContent), "name: deep_task.yml")
	assert.Contains(t, string(fileContent), "template: 'Custom template: {prompt}'")
	assert.Contains(t, string(fileContent), "name: deep_model")
	assert.Contains(t, string(fileContent), "ctx: 8192")
	assert.Contains(t, string(fileContent), "gpu_layers: 1")
	assert.Contains(t, string(fileContent), "n_predict: 500")
	assert.Contains(t, string(fileContent), "stop:")
	assert.Contains(t, string(fileContent), "- |4+")
	assert.Contains(t, string(fileContent), "- 'User:'")
}

func TestSaveTask_InvalidDirectoryPermissions(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	
	// Create a read-only directory to simulate permission issues
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0555) // Read-only permissions
	assert.NoError(t, err)
	
	// Save the original TasksDir
	originalTasksDir := state.TasksDir
	// Set the tasks directory to the read-only directory for testing
	state.TasksDir = readOnlyDir
	// Restore the original TasksDir after the test
	defer func() {
		state.TasksDir = originalTasksDir
	}()
	
	// Create test task that should try to write to the read-only directory
	task := types.Task{
		Name:     "test_task",
		Template: "{prompt}",
		ModelConf: types.ModelConf{
			Name: "test_model",
		},
		InferParams: types.InferenceParams{
			Stream: false,
		},
	}
	
	// Test SaveTask - should fail due to permissions
	err = SaveTask(task)
	
	// Assert error occurred
	assert.Error(t, err)
	
	// Verify file was not created
	expectedPath := filepath.Join(readOnlyDir, "test_task.yml")
	assert.NoFileExists(t, expectedPath)
}

func TestSaveTask_EmptyTask(t *testing.T) {
	// Create a temporary directory for tasks
	tempDir := t.TempDir()
	
	// Save the original TasksDir
	originalTasksDir := state.TasksDir
	// Set the tasks directory for testing
	state.TasksDir = tempDir
	// Restore the original TasksDir after the test
	defer func() {
		state.TasksDir = originalTasksDir
	}()
	
	// Create test task with minimal required fields
	task := types.Task{
		Name:     "minimal_task",
		Template: "{prompt}",
	}
	
	// Test SaveTask
	err := SaveTask(task)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Verify file was created
	expectedPath := filepath.Join(tempDir, "minimal_task.yml")
	assert.FileExists(t, expectedPath)
	
	// Read and verify file content
	fileContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	
	// The content should be a valid YAML representation of the task
	assert.Contains(t, string(fileContent), "name: minimal_task.yml")
	assert.Contains(t, string(fileContent), "template: '{prompt}'")
}

func TestSaveTask_WithSpecialCharacters(t *testing.T) {
	// Create a temporary directory for tasks
	tempDir := t.TempDir()
	
	// Save the original TasksDir
	originalTasksDir := state.TasksDir
	// Set the tasks directory for testing
	state.TasksDir = tempDir
	// Restore the original TasksDir after the test
	defer func() {
		state.TasksDir = originalTasksDir
	}()
	
	// Create test task with special characters in name and template
	task := types.Task{
		Name:     "special-chars_task_123",
		Template: "Special: {prompt}\nWith: \"quotes\" and 'apostrophes'",
		ModelConf: types.ModelConf{
			Name: "model-with-dashes",
		},
		InferParams: types.InferenceParams{
			Stream:      true,
			StopPrompts: []string{"STOP", "END", "\n\n"},
		},
	}
	
	// Test SaveTask
	err := SaveTask(task)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Verify file was created
	expectedPath := filepath.Join(tempDir, "special-chars_task_123.yml")
	assert.FileExists(t, expectedPath)
	
	// Read and verify file content
	fileContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	
	// The content should be a valid YAML representation of the task
	assert.Contains(t, string(fileContent), "name: special-chars_task_123.yml")
	assert.Contains(t, string(fileContent), "template: |-\n    Special: {prompt}\n    With: \"quotes\" and 'apostrophes'")
	assert.Contains(t, string(fileContent), "name: model-with-dashes")
	assert.Contains(t, string(fileContent), "stream: true")
	assert.Contains(t, string(fileContent), "stop:")
	assert.Contains(t, string(fileContent), "- STOP")
	assert.Contains(t, string(fileContent), "- END")
	assert.Contains(t, string(fileContent), "- |4+")
}
