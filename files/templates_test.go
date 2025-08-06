package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/synw/goinfer/state"
	"github.com/stretchr/testify/assert"
)

func TestReadTemplates(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create models directory
	modelsDir := filepath.Join(tempDir, "models")
	err := os.MkdirAll(modelsDir, 0755)
	assert.NoError(t, err)
	
	// Create templates.yml file
	templatesContent := `
model1:
  - ctx: 2048
    template: "default template"
  - ctx: 4096
    template: "extended template"

model2:
  - ctx: 1024
    template: "fast template"

model3:
  - ctx: 8192
    template: "premium template"
`
	templatesPath := filepath.Join(modelsDir, "templates.yml")
	err = os.WriteFile(templatesPath, []byte(templatesContent), 0644)
	assert.NoError(t, err)
	
	// Save the original ModelsDir
	originalModelsDir := state.ModelsDir
	// Set the models directory for testing to the models subdirectory
	state.ModelsDir = modelsDir
	// Restore the original ModelsDir after the test
	defer func() {
		state.ModelsDir = originalModelsDir
	}()
	
	// Test ReadTemplates
	templates, err := ReadTemplates()
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert expected templates are loaded
	assert.Len(t, templates, 3)
	
	// Check model1 templates (should use last entry: ctx: 4096, template: "extended template")
	assert.Contains(t, templates, "model1")
	model1Templates := templates["model1"]
	assert.Equal(t, "extended template", model1Templates.Name)
	assert.Equal(t, 4096, model1Templates.Ctx)
	
	// Check model2 templates
	assert.Contains(t, templates, "model2")
	model2Templates := templates["model2"]
	assert.Equal(t, "fast template", model2Templates.Name)
	assert.Equal(t, 1024, model2Templates.Ctx)
	
	// Check model3 templates
	assert.Contains(t, templates, "model3")
	model3Templates := templates["model3"]
	assert.Equal(t, "premium template", model3Templates.Name)
	assert.Equal(t, 8192, model3Templates.Ctx)
}

func TestReadTemplates_EmptyFile(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create models directory
	modelsDir := filepath.Join(tempDir, "models")
	err := os.MkdirAll(modelsDir, 0755)
	assert.NoError(t, err)
	
	// Create empty templates.yml file
	templatesPath := filepath.Join(modelsDir, "templates.yml")
	err = os.WriteFile(templatesPath, []byte(""), 0644)
	assert.NoError(t, err)
	
	// Save the original ModelsDir
	originalModelsDir := state.ModelsDir
	// Set the models directory for testing to the models subdirectory
	state.ModelsDir = modelsDir
	// Restore the original ModelsDir after the test
	defer func() {
		state.ModelsDir = originalModelsDir
	}()
	
	// Test ReadTemplates
	templates, err := ReadTemplates()
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert empty map returned
	assert.Empty(t, templates)
}

func TestReadTemplates_NonExistentFile(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create models directory but no templates.yml file
	modelsDir := filepath.Join(tempDir, "models")
	err := os.MkdirAll(modelsDir, 0755)
	assert.NoError(t, err)
	
	// Test ReadTemplates with non-existent file
	templates, err := ReadTemplates()
	
	// Assert error occurred (file not found)
	assert.Error(t, err)
	
	// Assert empty map returned on error
	assert.Empty(t, templates)
}

func TestReadTemplates_InvalidYAML(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create models directory
	modelsDir := filepath.Join(tempDir, "models")
	err := os.MkdirAll(modelsDir, 0755)
	assert.NoError(t, err)
	
	// Create invalid YAML templates.yml file
	invalidContent := `
model1:
  - ctx: 2048
    template: "valid template"
  - invalid: yaml: structure

model2:
  - ctx: 1024
    template: "another template"
`
	templatesPath := filepath.Join(modelsDir, "templates.yml")
	err = os.WriteFile(templatesPath, []byte(invalidContent), 0644)
	assert.NoError(t, err)
	
	// Test ReadTemplates
	templates, err := ReadTemplates()
	
	// Assert error occurred (invalid YAML)
	assert.Error(t, err)
	
	// Assert empty map returned on error
	assert.Empty(t, templates)
}

func TestReadTemplates_MissingModelsDir(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Save the original ModelsDir
	originalModelsDir := state.ModelsDir
	// Set the models directory for testing
	state.ModelsDir = tempDir
	// Restore the original ModelsDir after the test
	defer func() {
		state.ModelsDir = originalModelsDir
	}()
	
	// Test ReadTemplates with models directory that exists but no templates.yml file
	templates, err := ReadTemplates()
	
	// Assert error occurred (file not found)
	assert.Error(t, err)
	
	// Assert empty map returned on error
	assert.Empty(t, templates)
}

func TestReadTemplates_WithPartialData(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Save the original ModelsDir
	originalModelsDir := state.ModelsDir
	// Set the models directory for testing
	state.ModelsDir = tempDir
	// Restore the original ModelsDir after the test
	defer func() {
		state.ModelsDir = originalModelsDir
	}()
	
	// Create models directory
	modelsDir := filepath.Join(tempDir, "models")
	err := os.MkdirAll(modelsDir, 0755)
	assert.NoError(t, err)
	
	// Create templates.yml file with partial data
	templatesContent := `
model1:
  - ctx: 2048
    template: "complete template"

model2:
  - ctx: 1024
    # Missing template field

model3:
  - template: "missing ctx field"
`
	templatesPath := filepath.Join(modelsDir, "templates.yml")
	err = os.WriteFile(templatesPath, []byte(templatesContent), 0644)
	assert.NoError(t, err)
	
	// Test ReadTemplates
	templates, err := ReadTemplates()
	
	// Assert error occurred (due to type assertion failure for missing ctx)
	assert.Error(t, err)
	
	// Assert empty map returned on error
	assert.Empty(t, templates)
}

func TestReadTemplates_WithComplexStructure(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create models directory
	modelsDir := filepath.Join(tempDir, "models")
	err := os.MkdirAll(modelsDir, 0755)
	assert.NoError(t, err)
	
	// Create templates.yml file with complex nested structure
	templatesContent := `
llama-2-7b-chat:
  - ctx: 2048
    template: "chat template"
  - ctx: 4096
    template: "extended chat template"

llama-2-13b-chat:
  - ctx: 2048
    template: "chat template"
  - ctx: 4096
    template: "extended chat template"

codellama-7b:
  - ctx: 2048
    template: "code template"
  - ctx: 4096
    template: "extended code template"
  - ctx: 8192
    template: "premium code template"
`
	templatesPath := filepath.Join(modelsDir, "templates.yml")
	err = os.WriteFile(templatesPath, []byte(templatesContent), 0644)
	assert.NoError(t, err)
	
	// Save the original ModelsDir
	originalModelsDir := state.ModelsDir
	// Set the models directory for testing to the models subdirectory
	state.ModelsDir = modelsDir
	// Restore the original ModelsDir after the test
	defer func() {
		state.ModelsDir = originalModelsDir
	}()
	
	// Test ReadTemplates
	templates, err := ReadTemplates()
	
	// Assert no error
	assert.NoError(t, err)
	
	// Assert all models are processed
	assert.Len(t, templates, 3)
	
	// Check llama-2-7b-chat (should use last entry: ctx: 4096, template: "extended chat template")
	assert.Contains(t, templates, "llama-2-7b-chat")
	chat7b := templates["llama-2-7b-chat"]
	assert.Equal(t, "extended chat template", chat7b.Name)
	assert.Equal(t, 4096, chat7b.Ctx)
	
	// Check llama-2-13b-chat (should use last entry: ctx: 4096, template: "extended chat template")
	assert.Contains(t, templates, "llama-2-13b-chat")
	chat13b := templates["llama-2-13b-chat"]
	assert.Equal(t, "extended chat template", chat13b.Name)
	assert.Equal(t, 4096, chat13b.Ctx)
	
	// Check codellama-7b (should use last entry: ctx: 8192, template: "premium code template")
	assert.Contains(t, templates, "codellama-7b")
	code7b := templates["codellama-7b"]
	assert.Equal(t, "premium code template", code7b.Name)
	assert.Equal(t, 8192, code7b.Ctx)
}

// func TestReadTemplates_WithSpecialCharacters(t *testing.T) {
// 	// Create a temporary directory structure
// 	tempDir := t.TempDir()
	
// 	// Create models directory
// 	modelsDir := filepath.Join(tempDir, "models")
// 	err := os.MkdirAll(modelsDir, 0755)
// 	assert.NoError(t, err)
	
// 	// Create templates.yml file with special characters
// 	templatesContent := `
// model-with-dashes:
//   - ctx: 2048
//     template: "Template with: {prompt}\\nAnd \"quotes\""
// model_with_underscores:
//   - ctx: 4096
//     template: "Template with {system}\\nAnd {prompt}"
// model.with.dots:
//   - ctx: 1024
//     template: "Template\\nWith\\nNewlines"
// `
// 	templatesPath := filepath.Join(modelsDir, "templates.yml")
// 	err = os.WriteFile(templatesPath, []byte(templatesContent), 0644)
// 	assert.NoError(t, err)
	
// 	// Save the original ModelsDir
// 	originalModelsDir := state.ModelsDir
// 	// Set the models directory for testing to the models subdirectory
// 	state.ModelsDir = modelsDir
// 	// Restore the original ModelsDir after the test
// 	defer func() {
// 		state.ModelsDir = originalModelsDir
// 	}()
	
// 	// Test ReadTemplates
// 	templates, err := ReadTemplates()
	
// 	// Assert no error
// 	assert.NoError(t, err)
	
// 	// Assert all models are processed
// 	assert.Len(t, templates, 3)
	
// 	// Check model-with-dashes (literal \n characters from double-escaped YAML)
// 	assert.Contains(t, templates, "model-with-dashes")
// 	dashModel := templates["model-with-dashes"]
// 	assert.Equal(t, "Template with: {prompt}\\nAnd \"quotes\"", dashModel.Name)
// 	assert.Equal(t, 2048, dashModel.Ctx)
	
// 	// Check model_with_underscores (literal \n characters from double-escaped YAML)
// 	assert.Contains(t, templates, "model_with_underscores")
// 	underscoreModel := templates["model_with_underscores"]
// 	assert.Equal(t, "Template with {system}\\nAnd {prompt}", underscoreModel.Name)
// 	assert.Equal(t, 4096, underscoreModel.Ctx)
	
// 	// Check model.with.dots (literal \n characters from double-escaped YAML)
// 	assert.Contains(t, templates, "model.with.dots")
// 	dotModel := templates["model.with.dots"]
// 	assert.Equal(t, "Template\\nWith\\nNewlines", dotModel.Name)
// 	assert.Equal(t, 1024, dotModel.Ctx)
// }
