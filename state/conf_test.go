package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultInferenceParams(t *testing.T) {
	// Test that DefaultInferenceParams has expected values
	assert.False(t, DefaultInferenceParams.Stream)
	assert.Equal(t, 4, DefaultInferenceParams.Threads)
	assert.Equal(t, 512, DefaultInferenceParams.NPredict)
	assert.Equal(t, 40, DefaultInferenceParams.TopK)
	assert.Equal(t, float32(0.05), DefaultInferenceParams.MinP)
	assert.Equal(t, float32(0.95), DefaultInferenceParams.TopP)
	assert.Equal(t, float32(0.2), DefaultInferenceParams.Temperature)
	assert.Equal(t, float32(0.0), DefaultInferenceParams.FrequencyPenalty)
	assert.Equal(t, float32(0.0), DefaultInferenceParams.PresencePenalty)
	assert.Equal(t, float32(1.0), DefaultInferenceParams.RepeatPenalty)
	assert.Equal(t, float32(1.0), DefaultInferenceParams.TailFreeSamplingZ)
	assert.Equal(t, []string{"</s>"}, DefaultInferenceParams.StopPrompts)
}

func TestDefaultModelOptions(t *testing.T) {
	// Test that DefaultModelConf has expected values
	assert.Equal(t, 2048, DefaultModelConf.Ctx)
}

func TestDefaultModelConf(t *testing.T) {
	// Test that DefaultModelConf has expected values
	assert.Empty(t, DefaultModelConf.Name)
	assert.Equal(t, 2048, DefaultModelConf.Ctx)
}

func TestDefaultInferenceParamsImmutability(t *testing.T) {
	// Test that modifying DefaultInferenceParams doesn't affect other instances
	originalParams := DefaultInferenceParams

	// Modify the default params
	DefaultInferenceParams.Stream = true
	DefaultInferenceParams.Threads = 8

	// Create a new instance and verify it has the modified values
	newParams := DefaultInferenceParams
	assert.True(t, newParams.Stream)
	assert.Equal(t, 8, newParams.Threads)

	// Restore original values
	DefaultInferenceParams = originalParams
}

func TestDefaultModelOptionsImmutability(t *testing.T) {
	// Test that modifying DefaultModelConf doesn't affect other instances
	originalOptions := DefaultModelConf

	// Modify the default options
	DefaultModelConf.Ctx = 4096

	// Create a new instance and verify it has the modified values
	newOptions := DefaultModelConf
	assert.Equal(t, 4096, newOptions.Ctx)

	// Restore original values
	DefaultModelConf = originalOptions
}

func TestDefaultModelConfImmutability(t *testing.T) {
	// Test that modifying DefaultModelConf doesn't affect other instances
	originalConf := DefaultModelConf

	// Modify the default conf
	DefaultModelConf.Name = "test_model"
	DefaultModelConf.Ctx = 8192

	// Create a new instance and verify it has the modified values
	newConf := DefaultModelConf
	assert.Equal(t, "test_model", newConf.Name)
	assert.Equal(t, 8192, newConf.Ctx)

	// Restore original values
	DefaultModelConf = originalConf
}

func TestDefaultInferenceParamsWithCustomValues(t *testing.T) {
	// Test creating custom inference params based on defaults
	customParams := DefaultInferenceParams

	// Modify only specific fields
	customParams.Stream = true
	customParams.Threads = 16
	customParams.NPredict = 1024
	customParams.TopK = 80
	customParams.TopP = 0.8
	customParams.MinP = 0.2
	customParams.Temperature = 0.5

	// Verify other fields retain default values
	assert.True(t, customParams.Stream)
	assert.Equal(t, 16, customParams.Threads)
	assert.Equal(t, 1024, customParams.NPredict)
	assert.Equal(t, 80, customParams.TopK)
	assert.Equal(t, float32(0.2), customParams.MinP)
	assert.Equal(t, float32(0.8), customParams.TopP)
	assert.Equal(t, float32(0.5), customParams.Temperature)
	assert.Equal(t, float32(0.0), customParams.FrequencyPenalty)  // Default value
	assert.Equal(t, float32(0.0), customParams.PresencePenalty)   // Default value
	assert.Equal(t, float32(1.0), customParams.RepeatPenalty)     // Default value
	assert.Equal(t, float32(1.0), customParams.TailFreeSamplingZ) // Default value
	assert.Equal(t, []string{"</s>"}, customParams.StopPrompts)   // Default value
}

func TestDefaultModelOptionsWithCustomValues(t *testing.T) {
	// Test creating custom model options based on defaults
	customOptions := DefaultModelConf

	// Modify only specific fields
	customOptions.Ctx = 8192

	// Verify other fields retain default values
	assert.Equal(t, 8192, customOptions.Ctx)
}

func TestDefaultModelConfWithCustomValues(t *testing.T) {
	// Test creating custom model conf based on defaults
	customConf := DefaultModelConf

	// Modify only specific fields
	customConf.Name = "custom_model"
	customConf.Ctx = 4096

	// Verify other fields retain default values
	assert.Equal(t, "custom_model", customConf.Name)
	assert.Equal(t, 4096, customConf.Ctx)
}
