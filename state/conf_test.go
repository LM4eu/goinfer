package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultInferenceParams(t *testing.T) {
	// Test that DefaultInferenceParams has expected values
	assert.Equal(t, false, DefaultInferenceParams.Stream)
	assert.Equal(t, 4, DefaultInferenceParams.Threads)
	assert.Equal(t, 512, DefaultInferenceParams.NPredict)
	assert.Equal(t, 40, DefaultInferenceParams.TopK)
	assert.Equal(t, float32(0.95), DefaultInferenceParams.TopP)
	assert.Equal(t, float32(0.2), DefaultInferenceParams.Temperature)
	assert.Equal(t, float32(0.0), DefaultInferenceParams.FrequencyPenalty)
	assert.Equal(t, float32(0.0), DefaultInferenceParams.PresencePenalty)
	assert.Equal(t, float32(1.0), DefaultInferenceParams.RepeatPenalty)
	assert.Equal(t, float32(1.0), DefaultInferenceParams.TailFreeSamplingZ)
	assert.Equal(t, []string{"</s>"}, DefaultInferenceParams.StopPrompts)
}

func TestDefaultModelOptions(t *testing.T) {
	// Test that DefaultModelOptions has expected values
	assert.Equal(t, 2048, DefaultModelOptions.ContextSize)
	assert.Equal(t, 0, DefaultModelOptions.Seed)
	assert.Equal(t, false, DefaultModelOptions.F16Memory)
	assert.Equal(t, false, DefaultModelOptions.MLock)
	assert.Equal(t, true, DefaultModelOptions.MMap)
	assert.Equal(t, false, DefaultModelOptions.Embeddings)
	assert.Equal(t, false, DefaultModelOptions.LowVRAM)
	assert.Equal(t, 512, DefaultModelOptions.NBatch)
	assert.Equal(t, float32(10000), DefaultModelOptions.FreqRopeBase)
	assert.Equal(t, float32(1.0), DefaultModelOptions.FreqRopeScale)
	assert.Equal(t, 0, DefaultModelOptions.NGPULayers)
}

func TestDefaultModelConf(t *testing.T) {
	// Test that DefaultModelConf has expected values
	assert.Equal(t, "", DefaultModelConf.Name)
	assert.Equal(t, 2048, DefaultModelConf.Ctx)
	assert.Equal(t, 0, DefaultModelConf.GPULayers)
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
	// Test that modifying DefaultModelOptions doesn't affect other instances
	originalOptions := DefaultModelOptions
	
	// Modify the default options
	DefaultModelOptions.ContextSize = 4096
	DefaultModelOptions.NGPULayers = 2
	
	// Create a new instance and verify it has the modified values
	newOptions := DefaultModelOptions
	assert.Equal(t, 4096, newOptions.ContextSize)
	assert.Equal(t, 2, newOptions.NGPULayers)
	
	// Restore original values
	DefaultModelOptions = originalOptions
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
	customParams.Temperature = 0.5
	
	// Verify other fields retain default values
	assert.Equal(t, true, customParams.Stream)
	assert.Equal(t, 16, customParams.Threads)
	assert.Equal(t, 1024, customParams.NPredict)
	assert.Equal(t, 80, customParams.TopK)
	assert.Equal(t, float32(0.8), customParams.TopP)
	assert.Equal(t, float32(0.5), customParams.Temperature)
	assert.Equal(t, float32(0.0), customParams.FrequencyPenalty) // Default value
	assert.Equal(t, float32(0.0), customParams.PresencePenalty)  // Default value
	assert.Equal(t, float32(1.0), customParams.RepeatPenalty)    // Default value
	assert.Equal(t, float32(1.0), customParams.TailFreeSamplingZ) // Default value
	assert.Equal(t, []string{"</s>"}, customParams.StopPrompts) // Default value
}

func TestDefaultModelOptionsWithCustomValues(t *testing.T) {
	// Test creating custom model options based on defaults
	customOptions := DefaultModelOptions
	
	// Modify only specific fields
	customOptions.ContextSize = 8192
	customOptions.Seed = 42
	customOptions.NBatch = 1024
	customOptions.NGPULayers = 4
	customOptions.FreqRopeBase = 20000
	customOptions.FreqRopeScale = 0.8
	
	// Verify other fields retain default values
	assert.Equal(t, 8192, customOptions.ContextSize)
	assert.Equal(t, 42, customOptions.Seed)
	assert.Equal(t, 1024, customOptions.NBatch)
	assert.Equal(t, 4, customOptions.NGPULayers)
	assert.Equal(t, float32(20000), customOptions.FreqRopeBase)
	assert.Equal(t, float32(0.8), customOptions.FreqRopeScale)
	assert.Equal(t, false, customOptions.F16Memory) // Default value
	assert.Equal(t, false, customOptions.MLock)     // Default value
	assert.Equal(t, true, customOptions.MMap)       // Default value
	assert.Equal(t, false, customOptions.Embeddings) // Default value
	assert.Equal(t, false, customOptions.LowVRAM)   // Default value
}

func TestDefaultModelConfWithCustomValues(t *testing.T) {
	// Test creating custom model conf based on defaults
	customConf := DefaultModelConf
	
	// Modify only specific fields
	customConf.Name = "custom_model"
	customConf.Ctx = 4096
	customConf.GPULayers = 1
	
	// Verify other fields retain default values
	assert.Equal(t, "custom_model", customConf.Name)
	assert.Equal(t, 4096, customConf.Ctx)
	assert.Equal(t, 1, customConf.GPULayers)
}
