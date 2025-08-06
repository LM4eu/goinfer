package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTasks_FlatDirectory(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create test task files
	taskFiles := []struct {
		name    string
		content string
	}{
		{"test_task.yml", "name: test_task\ntemplate: {prompt}"},
		{"another_task.yml", "name: another_task\ntemplate: {system}\n\n{prompt}"},
		{"third_task.yml", "name: third_task\ntemplate: Custom template"},
	}
	
	for _, tf := range taskFiles {
		filePath := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(filePath, []byte(tf.content), 0644)
		require.NoError(t, err)
	}
	
	// Test ReadTasks
	nodes, err := ReadTasks(tempDir)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert all tasks are found
	assert.Len(t, nodes, 3)
	
	// Check each task node (order may vary due to sorting)
	taskNames := []string{"test_task", "another_task", "third_task"}
	for _, expectedName := range taskNames {
		found := false
		for _, node := range nodes {
			if node.Label == expectedName {
				found = true
				assert.Equal(t, expectedName+".yml", node.Path)
				assert.Empty(t, node.Children) // No children in flat structure
				break
			}
		}
		assert.True(t, found, "Expected task %s not found", expectedName)
	}
}

func TestReadTasks_NestedDirectories(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2", "nested")
	err := os.MkdirAll(subDir1, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(subDir2, 0755)
	require.NoError(t, err)
	
	// Create test task files
	taskFiles := []struct {
		path    string
		content string
	}{
		{"root_task.yml", "name: root_task\ntemplate: {prompt}"},
		{"subdir1/sub_task.yml", "name: sub_task\ntemplate: {system}\n\n{prompt}"},
		{"subdir2/nested/deep_task.yml", "name: deep_task\ntemplate: Deep template"},
	}
	
	for _, tf := range taskFiles {
		filePath := filepath.Join(tempDir, tf.path)
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filePath, []byte(tf.content), 0644)
		require.NoError(t, err)
	}
	
	// Test ReadTasks
	nodes, err := ReadTasks(tempDir)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert root level directories - should return directories containing tasks
	assert.Len(t, nodes, 3) // root_task, subdir1, and subdir2
	
	// Check root_task
	assert.Equal(t, "root_task", nodes[0].Label)
	assert.Equal(t, "root_task.yml", nodes[0].Path)
	assert.Empty(t, nodes[0].Children)
	
	// Check subdir1
	assert.Equal(t, "subdir1", nodes[1].Label)
	assert.Equal(t, "subdir1", nodes[1].Path)
	assert.Len(t, nodes[1].Children, 1)
	assert.Equal(t, "sub_task", nodes[1].Children[0].Label)
	assert.Equal(t, "subdir1/sub_task.yml", nodes[1].Children[0].Path)
	
	// Check subdir2
	assert.Equal(t, "subdir2", nodes[2].Label)
	assert.Equal(t, "subdir2", nodes[2].Path)
	assert.Len(t, nodes[2].Children, 1)
	assert.Equal(t, "nested", nodes[2].Children[0].Label)
	assert.Equal(t, "subdir2/nested", nodes[2].Children[0].Path)
	assert.Len(t, nodes[2].Children[0].Children, 1)
	assert.Equal(t, "deep_task", nodes[2].Children[0].Children[0].Label)
	assert.Equal(t, "subdir2/nested/deep_task.yml", nodes[2].Children[0].Children[0].Path)
}

func TestReadTasks_EmptyDirectory(t *testing.T) {
	// Create a temporary empty directory
	tempDir := t.TempDir()
	
	// Test ReadTasks on empty directory
	nodes, err := ReadTasks(tempDir)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert empty result
	assert.Empty(t, nodes)
}

func TestReadTasks_NonExistentDirectory(t *testing.T) {
	// Test ReadTasks with non-existent directory
	nonExistentDir := "/path/that/does/not/exist"
	nodes, err := ReadTasks(nonExistentDir)
	
	// Assert error occurred
	assert.Error(t, err)
	
	// Assert empty result
	assert.Empty(t, nodes)
}

func TestReadTasks_WithNonYMLFiles(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create test files (mix of yml and non-yml)
	files := []struct {
		name    string
		content string
		isYML   bool
	}{
		{"task1.yml", "name: task1\ntemplate: {prompt}", true},
		{"task2.yml", "name: task2\ntemplate: {system}", true},
		{"readme.md", "# This is a readme", false},
		{"config.json", "{\"key\": \"value\"}", false},
		{"script.sh", "echo 'hello'", false},
		{"task3.yml", "name: task3\ntemplate: Custom", true},
	}
	
	for _, f := range files {
		filePath := filepath.Join(tempDir, f.name)
		err := os.WriteFile(filePath, []byte(f.content), 0644)
		require.NoError(t, err)
	}
	
	// Test ReadTasks
	nodes, err := ReadTasks(tempDir)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert only yml files are processed
	assert.Len(t, nodes, 3)
	
	// Check each task node
	taskNames := []string{"task1", "task2", "task3"}
	for i, expectedName := range taskNames {
		assert.Equal(t, expectedName, nodes[i].Label)
		assert.Equal(t, expectedName+".yml", nodes[i].Path)
		assert.Empty(t, nodes[i].Children)
	}
}

func TestReadTasks_WithDeeplyNestedStructure(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create deeply nested directories
	deepPath := filepath.Join(tempDir, "level1", "level2", "level3", "level4")
	err := os.MkdirAll(deepPath, 0755)
	require.NoError(t, err)
	
	// Create test task files at different levels
	taskFiles := []struct {
		path    string
		content string
	}{
		{"root.yml", "name: root\ntemplate: Root template"},
		{"level1/l1.yml", "name: l1\ntemplate: Level 1"},
		{"level1/level2/l2.yml", "name: l2\ntemplate: Level 2"},
		{"level1/level2/level3/l3.yml", "name: l3\ntemplate: Level 3"},
		{"level1/level2/level3/level4/l4.yml", "name: l4\ntemplate: Level 4"},
	}
	
	for _, tf := range taskFiles {
		filePath := filepath.Join(tempDir, tf.path)
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filePath, []byte(tf.content), 0644)
		require.NoError(t, err)
	}
	
	// Test ReadTasks
	nodes, err := ReadTasks(tempDir)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Should return directory structure
	assert.Len(t, nodes, 2) // l1 and level2
	
	// Check l1
	assert.Equal(t, "l1", nodes[0].Label)
	assert.Equal(t, "level1/l1.yml", nodes[0].Path)
	assert.Empty(t, nodes[0].Children)
	
	// Check level2
	assert.Equal(t, "level2", nodes[1].Label)
	assert.Equal(t, "level1/level2", nodes[1].Path)
	assert.Len(t, nodes[1].Children, 2)
	
	// Check l2 (child of level2)
	assert.Equal(t, "l2", nodes[1].Children[0].Label)
	assert.Equal(t, "level1/level2/l2.yml", nodes[1].Children[0].Path)
	assert.Empty(t, nodes[1].Children[0].Children)
	
	// Check level3 (child of level2)
	assert.Equal(t, "level3", nodes[1].Children[1].Label)
	assert.Equal(t, "level1/level2/level3", nodes[1].Children[1].Path)
	assert.Len(t, nodes[1].Children[1].Children, 2)
	
	// Check l3 (child of level3)
	assert.Equal(t, "l3", nodes[1].Children[1].Children[0].Label)
	assert.Equal(t, "level1/level2/level3/l3.yml", nodes[1].Children[1].Children[0].Path)
	assert.Empty(t, nodes[1].Children[1].Children[0].Children)
	
	// Check level4 (child of level3)
	assert.Equal(t, "level4", nodes[1].Children[1].Children[1].Label)
	assert.Equal(t, "level1/level2/level3/level4", nodes[1].Children[1].Children[1].Path)
	assert.Len(t, nodes[1].Children[1].Children[1].Children, 1)
	
	// Check l4 (child of level4)
	assert.Equal(t, "l4", nodes[1].Children[1].Children[1].Children[0].Label)
	assert.Equal(t, "level1/level2/level3/level4/l4.yml", nodes[1].Children[1].Children[1].Children[0].Path)
	assert.Empty(t, nodes[1].Children[1].Children[1].Children[0].Children)
}

func TestReadTasks_WithSpecialCharacters(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create test task files with special characters
	taskFiles := []struct {
		name    string
		content string
	}{
		{"task-with-dashes.yml", "name: task-with-dashes\ntemplate: {prompt}"},
		{"task_with_underscores.yml", "name: task_with_underscores\ntemplate: {system}"},
		{"task.with.dots.yml", "name: task.with.dots\ntemplate: Custom"},
		{"task with spaces.yml", "name: task with spaces\ntemplate: Space template"},
	}
	
	for _, tf := range taskFiles {
		filePath := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(filePath, []byte(tf.content), 0644)
		require.NoError(t, err)
	}
	
	// Test ReadTasks
	nodes, err := ReadTasks(tempDir)
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert all tasks are found
	assert.Len(t, nodes, 4)
	
	// Check each task node preserves special characters (order may vary)
	expectedNames := []string{"task-with-dashes", "task_with_underscores", "task.with.dots", "task with spaces"}
	foundNames := make(map[string]bool)
	
	for _, node := range nodes {
		foundNames[node.Label] = true
		assert.Equal(t, node.Label+".yml", node.Path)
		assert.Empty(t, node.Children)
	}
	
	// Verify all expected names were found
	for _, expectedName := range expectedNames {
		assert.True(t, foundNames[expectedName], "Expected task %s not found", expectedName)
	}
}
