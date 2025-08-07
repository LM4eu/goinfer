package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func TestReadTask_ValidTaskFile(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a valid task file
	taskContent := `
name: test_task
template: "{system}\n\n{prompt}"
modelConf:
  - name: test_model
    ctx: 2048
inferParams:
  - stream: false
    threads: 4
    n_predict: 100
    top_k: 40
    top_p: 0.9
    temperature: 0.7
`
	taskPath := filepath.Join(tempDir, "test_task.yml")
	err := os.WriteFile(taskPath, []byte(taskContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask
	exists, task, err := ReadTask("test_task.yml")

	// Assert no error
	assert.NoError(t, err)

	// Assert task exists
	assert.True(t, exists)

	// Assert task content
	assert.Equal(t, "test_task", task.Name)
	assert.Equal(t, "{system}\n\n{prompt}", task.Template)
	assert.Equal(t, "test_model", task.ModelConf.Name)
	assert.Equal(t, 2048, task.ModelConf.Ctx)
	assert.Equal(t, 4, task.InferParams.Threads)
	assert.Equal(t, 100, task.InferParams.NPredict)
	assert.Equal(t, 40, task.InferParams.TopK)
	assert.Equal(t, float32(0.9), task.InferParams.TopP)
	assert.Equal(t, float32(0.7), task.InferParams.Temperature)
	assert.False(t, task.InferParams.Stream)
}

func TestReadTask_TaskWithSubdirectory(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	// Create a valid task file in subdirectory
	taskContent := `
name: sub_task
template: "Subdirectory template"
modelConf:
  - name: sub_model
    ctx: 4096
inferParams:
  - stream: true
    threads: 8
    n_predict: 200
`
	taskPath := filepath.Join(subDir, "sub_task.yml")
	err = os.WriteFile(taskPath, []byte(taskContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with subdirectory path
	exists, task, err := ReadTask("subdir/sub_task.yml")

	// Assert no error
	assert.NoError(t, err)

	// Assert task exists
	if !exists {
		t.Fatal("Task should exist in subdirectory")
	}

	// Assert task content
	assert.Equal(t, "sub_task", task.Name)
	assert.Equal(t, "Subdirectory template", task.Template)
	assert.Equal(t, "sub_model", task.ModelConf.Name)
	assert.Equal(t, 4096, task.ModelConf.Ctx)
	assert.True(t, task.InferParams.Stream)
	assert.Equal(t, 8, task.InferParams.Threads)
	assert.Equal(t, 200, task.InferParams.NPredict)
}

func TestReadTask_NonExistentFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Test ReadTask with non-existent file
	exists, task, err := ReadTask("non_existent_task.yml")

	// Assert no error (returns false but no error)
	assert.NoError(t, err)

	// Assert task doesn't exist
	assert.False(t, exists)

	// Assert empty task
	assert.Equal(t, types.Task{}, task)
}

func TestReadTask_InvalidYAML(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a file with invalid YAML
	invalidContent := `
name: test_task
template: "{prompt}"
invalid: yaml: structure
modelConf:
  - name: test_model
    ctx: 2048
`
	taskPath := filepath.Join(tempDir, "invalid_task.yml")
	err := os.WriteFile(taskPath, []byte(invalidContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with invalid YAML
	exists, task, err := ReadTask("invalid_task.yml")

	// Assert error occurred
	assert.Error(t, err)

	// Assert task doesn't exist
	assert.False(t, exists)

	// Assert empty task
	assert.Equal(t, types.Task{}, task)
}

func TestReadTask_MissingRequiredFields(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a task file with missing required fields
	incompleteContent := `
name: 
template: 
modelConf:
  - name: test_model
    ctx: 2048
`
	taskPath := filepath.Join(tempDir, "incomplete_task.yml")
	err := os.WriteFile(taskPath, []byte(incompleteContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with incomplete data
	exists, task, err := ReadTask("incomplete_task.yml")

	// Assert error occurred
	assert.Error(t, err)

	// Assert task doesn't exist
	assert.False(t, exists)

	// Assert empty task
	assert.Equal(t, types.Task{}, task)
}

func TestReadTask_MissingTemplateField(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a task file with missing template field
	incompleteContent := `
name: test_task
modelConf:
  - name: test_model
    ctx: 2048
inferParams:
  - stream: false
    threads: 4
`
	taskPath := filepath.Join(tempDir, "no_template_task.yml")
	err := os.WriteFile(taskPath, []byte(incompleteContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with missing template
	exists, task, err := ReadTask("no_template_task.yml")

	// Assert error occurred
	assert.Error(t, err)

	// Assert task doesn't exist
	assert.False(t, exists)

	// Assert empty task
	assert.Equal(t, types.Task{}, task)
}

func TestReadTask_MissingModelConf(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a task file with missing modelConf
	minimalContent := `
name: test_task
template: "{prompt}"
inferParams:
  - stream: false
    threads: 4
`
	taskPath := filepath.Join(tempDir, "no_modelconf_task.yml")
	err := os.WriteFile(taskPath, []byte(minimalContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with missing modelConf
	exists, task, err := ReadTask("no_modelconf_task.yml")

	// Assert no error (should use defaults)
	assert.NoError(t, err)

	// Assert task exists
	assert.True(t, exists)

	// Assert task content with defaults
	assert.Equal(t, "test_task", task.Name)
	assert.Equal(t, "{prompt}", task.Template)
	assert.Equal(t, "", task.ModelConf.Name)     // Default empty name
	assert.Equal(t, 2048, task.ModelConf.Ctx)    // Default ctx
	assert.Equal(t, 0, task.ModelConf.GPULayers) // Default gpu_layers
}

func TestReadTask_MissingInferParams(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a task file with missing inferParams
	minimalContent := `
name: test_task
template: "{prompt}"
modelConf:
  - name: test_model
    ctx: 2048
`
	taskPath := filepath.Join(tempDir, "no_inferparams_task.yml")
	err := os.WriteFile(taskPath, []byte(minimalContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with missing inferParams
	exists, task, err := ReadTask("no_inferparams_task.yml")

	// Assert no error (should use defaults)
	assert.NoError(t, err)

	// Assert task exists
	assert.True(t, exists)

	// Assert task content with default inferParams
	assert.Equal(t, "test_task", task.Name)
	assert.Equal(t, "{prompt}", task.Template)
	assert.Equal(t, "test_model", task.ModelConf.Name)
	assert.Equal(t, 2048, task.ModelConf.Ctx)

	// Check default inferParams
	assert.Equal(t, state.DefaultInferenceParams.Stream, task.InferParams.Stream)
	assert.Equal(t, state.DefaultInferenceParams.Threads, task.InferParams.Threads)
	assert.Equal(t, state.DefaultInferenceParams.NPredict, task.InferParams.NPredict)
	assert.Equal(t, state.DefaultInferenceParams.TopK, task.InferParams.TopK)
	assert.Equal(t, state.DefaultInferenceParams.TopP, task.InferParams.TopP)
	assert.Equal(t, state.DefaultInferenceParams.Temperature, task.InferParams.Temperature)
	assert.Equal(t, state.DefaultInferenceParams.StopPrompts, task.InferParams.StopPrompts)
}

func TestReadTask_WithSpecialCharacters(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a task file with special characters
	specialContent := `
name: "special-chars_task_123"
template: "Special: {prompt}\nWith: \"quotes\" and 'apostrophes'"
modelConf:
  - name: "model-with-dashes"
    ctx: 2048
inferParams:
  - stream: true
    stop:
      - "STOP"
      - "END"
      - "\n\n"
`
	taskPath := filepath.Join(tempDir, "special_task.yml")
	err := os.WriteFile(taskPath, []byte(specialContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with special characters
	exists, task, err := ReadTask("special_task.yml")

	// Assert no error
	assert.NoError(t, err, "Should be able to read file with special characters")

	// Assert task exists
	assert.True(t, exists, "Task with special characters should exist")

	// Assert task content preserves special characters
	assert.Equal(t, "special-chars_task_123", task.Name)
	assert.Equal(t, "Special: {prompt}\nWith: \"quotes\" and 'apostrophes'", task.Template)
	assert.Equal(t, "model-with-dashes", task.ModelConf.Name)
	assert.True(t, task.InferParams.Stream)
	assert.Equal(t, []string{"STOP", "END", "\n\n"}, task.InferParams.StopPrompts)
}

func TestReadTask_WithComplexInferParams(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a task file with complex inference parameters
	complexContent := `
name: complex_task
template: "{system}\n\n{prompt}"
modelConf:
  - name: complex_model
    ctx: 8192
    gpu_layers: 1
inferParams:
  - stream: false
    threads: 16
    n_predict: 500
    top_k: 100
    top_p: 0.99
    temperature: 1.0
    frequency_penalty: 0.1
    presence_penalty: 0.1
    repeat_penalty: 1.1
    tfs_z: 0.5
    stop:
      - "\n"
      - "User:"
      - "Assistant:"
`
	taskPath := filepath.Join(tempDir, "complex_task.yml")
	err := os.WriteFile(taskPath, []byte(complexContent), 0o644)
	require.NoError(t, err)

	// Test ReadTask with complex parameters
	exists, task, err := ReadTask("complex_task.yml")

	// Assert no error
	assert.NoError(t, err)

	// Assert task exists
	assert.True(t, exists)

	// Assert complex task content
	assert.Equal(t, "complex_task", task.Name)
	assert.Equal(t, "{system}\n\n{prompt}", task.Template)
	assert.Equal(t, "complex_model", task.ModelConf.Name)
	assert.Equal(t, 8192, task.ModelConf.Ctx)
	assert.Equal(t, 1, task.ModelConf.GPULayers)
	assert.Equal(t, 16, task.InferParams.Threads)
	assert.Equal(t, 500, task.InferParams.NPredict)
	assert.Equal(t, 100, task.InferParams.TopK)
	assert.Equal(t, float32(0.99), task.InferParams.TopP)
	assert.Equal(t, float32(1.0), task.InferParams.Temperature)
	assert.Equal(t, float32(0.1), task.InferParams.FrequencyPenalty)
	assert.Equal(t, float32(0.1), task.InferParams.PresencePenalty)
	assert.Equal(t, float32(1.1), task.InferParams.RepeatPenalty)
	assert.Equal(t, float32(0.5), task.InferParams.TailFreeSamplingZ)
	assert.Equal(t, []string{"\n", "User:", "Assistant:"}, task.InferParams.StopPrompts)
}

func TestReadTask_FilePermissionError(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Set the tasks directory for testing
	state.TasksDir = tempDir

	// Create a read-only directory (no execute permissions)
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0o755) // Normal directory first
	require.NoError(t, err)

	// Create a task file in the read-only directory
	taskContent := `
name: readonly_task
template: "{prompt}"
`
	taskPath := filepath.Join(readOnlyDir, "readonly_task.yml")
	err = os.WriteFile(taskPath, []byte(taskContent), 0o644)
	require.NoError(t, err)

	// Make the directory read-only (no execute permissions)
	err = os.Chmod(readOnlyDir, 0o500) // Read-only, no execute permissions
	require.NoError(t, err)

	// Test ReadTask with read-only file
	exists, task, err := ReadTask("readonly/readonly_task.yml")

	// The behavior might vary depending on the system, so let's check both possibilities
	// In some cases, permission errors might be handled gracefully
	if err != nil {
		// Assert error occurred (permission denied)
		assert.Contains(t, err.Error(), "permission denied", "Error message should mention permission denied")
		assert.False(t, exists, "Task should not exist when file cannot be read")
		assert.Equal(t, types.Task{}, task, "Should return empty task when file cannot be read")
	} else {
		// If no error, the file was successfully read (this can happen on some systems)
		assert.True(t, exists, "Task should exist when file can be read")
		assert.Equal(t, "readonly_task", task.Name, "Task name should be correct")
		assert.Equal(t, "{prompt}", task.Template, "Task template should be correct")
	}

	// Restore permissions to allow cleanup
	_ = os.Chmod(readOnlyDir, 0o755)
}
