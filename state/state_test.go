package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/types"
)

func TestStateInitialization(t *testing.T) {
	// Test that state variables have expected default values
	assert.Empty(t, ModelsDir)
	assert.False(t, IsModelLoaded)
	assert.Empty(t, LoadedModel)
	assert.Equal(t, DefaultModelOptions, ModelOptions)
	assert.True(t, ContinueInferringController)
	assert.False(t, IsInferring)
	assert.True(t, IsVerbose)
	assert.False(t, IsDebug)
	assert.Equal(t, types.OpenAiConf{}, OpenAiConf)

	// Test that Lm is initialized as empty LLama
	var expectedLm llama.LLama
	assert.Equal(t, expectedLm, Lm)
}

func TestStateModification(t *testing.T) {
	// Test modifying state variables
	originalModelsDir := ModelsDir
	originalIsModelLoaded := IsModelLoaded
	originalLoadedModel := LoadedModel
	originalModelOptions := ModelOptions
	originalContinueInferringController := ContinueInferringController
	originalIsInferring := IsInferring
	originalIsVerbose := IsVerbose
	originalIsDebug := IsDebug
	originalOpenAiConf := OpenAiConf

	// Modify state variables
	ModelsDir = "/test/models"
	IsModelLoaded = true
	LoadedModel = "test_model"
	ModelOptions = llama.ModelOptions{ContextSize: 4096}
	ContinueInferringController = false
	IsInferring = true
	IsVerbose = false
	IsDebug = true
	OpenAiConf = types.OpenAiConf{Threads: 8, Template: "custom template"}

	// Assert modifications
	assert.Equal(t, "/test/models", ModelsDir)
	assert.True(t, IsModelLoaded)
	assert.Equal(t, "test_model", LoadedModel)
	assert.Equal(t, llama.ModelOptions{ContextSize: 4096}, ModelOptions)
	assert.False(t, ContinueInferringController)
	assert.True(t, IsInferring)
	assert.False(t, IsVerbose)
	assert.True(t, IsDebug)
	assert.Equal(t, types.OpenAiConf{Threads: 8, Template: "custom template"}, OpenAiConf)

	// Restore original values
	ModelsDir = originalModelsDir
	IsModelLoaded = originalIsModelLoaded
	LoadedModel = originalLoadedModel
	ModelOptions = originalModelOptions
	ContinueInferringController = originalContinueInferringController
	IsInferring = originalIsInferring
	IsVerbose = originalIsVerbose
	IsDebug = originalIsDebug
	OpenAiConf = originalOpenAiConf
}

func TestStateConcurrentAccess(t *testing.T) {
	// Test concurrent access to state variables with proper synchronization
	// Set initial values
	IsModelLoaded = false
	ContinueInferringController = true
	IsInferring = false

	// Use channels to synchronize goroutines
	writeDone := make(chan bool)
	readDone := make(chan bool)

	// Goroutine 1: Modify state
	go func() {
		IsModelLoaded = true
		LoadedModel = "concurrent_model"
		ContinueInferringController = false
		writeDone <- true
	}()

	// Wait for write to complete before reading
	<-writeDone

	// Goroutine 2: Read state
	go func() {
		modelLoaded := IsModelLoaded
		loadedModel := LoadedModel
		continueInferring := ContinueInferringController

		// Assert values after modification
		assert.True(t, modelLoaded)
		assert.Equal(t, "concurrent_model", loadedModel)
		assert.False(t, continueInferring)
		readDone <- true
	}()

	// Wait for read to complete
	<-readDone
}

func TestStateModelOptions(t *testing.T) {
	// Test ModelOptions state variable
	originalModelOptions := ModelOptions

	// Modify ModelOptions
	ModelOptions = llama.ModelOptions{
		ContextSize:   8192,
		Seed:          42,
		NBatch:        1024,
		F16Memory:     true,
		MLock:         true,
		MMap:          false,
		LowVRAM:       true,
		Embeddings:    true,
		NUMA:          true,
		NGPULayers:    2,
		MainGPU:       "0",
		TensorSplit:   "1,1",
		FreqRopeBase:  10000,
		FreqRopeScale: 1.0,
	}

	// Assert modification
	assert.Equal(t, llama.ModelOptions{
		ContextSize:   8192,
		Seed:          42,
		NBatch:        1024,
		F16Memory:     true,
		MLock:         true,
		MMap:          false,
		LowVRAM:       true,
		Embeddings:    true,
		NUMA:          true,
		NGPULayers:    2,
		MainGPU:       "0",
		TensorSplit:   "1,1",
		FreqRopeBase:  10000,
		FreqRopeScale: 1.0,
	}, ModelOptions)

	// Restore original value
	ModelOptions = originalModelOptions
}

func TestStateInferenceFlags(t *testing.T) {
	// Test inference-related state flags
	originalIsInferring := IsInferring
	originalContinueInferringController := ContinueInferringController

	// Test inference start
	IsInferring = true
	ContinueInferringController = true

	assert.True(t, IsInferring)
	assert.True(t, ContinueInferringController)

	// Test inference abort
	ContinueInferringController = false

	assert.True(t, IsInferring)
	assert.False(t, ContinueInferringController)

	// Test inference end
	IsInferring = false

	assert.False(t, IsInferring)
	assert.False(t, ContinueInferringController)

	// Restore original values
	IsInferring = originalIsInferring
	ContinueInferringController = originalContinueInferringController
}

func TestStateDebugAndVerboseFlags(t *testing.T) {
	// Test debug and verbose flags
	originalIsVerbose := IsVerbose
	originalIsDebug := IsDebug

	// Test verbose mode
	IsVerbose = true
	IsDebug = false

	assert.True(t, IsVerbose)
	assert.False(t, IsDebug)

	// Test debug mode
	IsVerbose = false
	IsDebug = true

	assert.False(t, IsVerbose)
	assert.True(t, IsDebug)

	// Test both modes
	IsVerbose = true
	IsDebug = true

	assert.True(t, IsVerbose)
	assert.True(t, IsDebug)

	// Test neither mode
	IsVerbose = false
	IsDebug = false

	assert.False(t, IsVerbose)
	assert.False(t, IsDebug)

	// Restore original values
	IsVerbose = originalIsVerbose
	IsDebug = originalIsDebug
}

func TestStateOpenAiConfiguration(t *testing.T) {
	// Test OpenAiConf state variable
	originalOpenAiConf := OpenAiConf

	// Modify OpenAiConf
	OpenAiConf = types.OpenAiConf{
		Threads:  16,
		Template: "custom openai template",
	}

	// Assert modification
	assert.Equal(t, types.OpenAiConf{
		Threads:  16,
		Template: "custom openai template",
	}, OpenAiConf)

	// Restore original value
	OpenAiConf = originalOpenAiConf
}

func TestStateModelLoadedState(t *testing.T) {
	// Test model loaded state
	originalIsModelLoaded := IsModelLoaded
	originalLoadedModel := LoadedModel
	originalModelOptions := ModelOptions

	// Test model loaded state
	IsModelLoaded = true
	LoadedModel = "test_model.bin"
	ModelOptions = llama.ModelOptions{ContextSize: 2048}

	assert.True(t, IsModelLoaded)
	assert.Equal(t, "test_model.bin", LoadedModel)
	assert.Equal(t, llama.ModelOptions{ContextSize: 2048}, ModelOptions)

	// Test model unloaded state
	IsModelLoaded = false
	LoadedModel = ""
	ModelOptions = llama.ModelOptions{}

	assert.False(t, IsModelLoaded)
	assert.Empty(t, LoadedModel)
	assert.Equal(t, llama.ModelOptions{}, ModelOptions)

	// Restore original values
	IsModelLoaded = originalIsModelLoaded
	LoadedModel = originalLoadedModel
	ModelOptions = originalModelOptions
}

func TestStateLLamaInstance(t *testing.T) {
	// Test Lm state variable
	originalLm := Lm

	// Test setting LLama instance
	var testLm llama.LLama = "test_model_path"
	Lm = testLm

	assert.Equal(t, testLm, Lm)

	// Test clearing LLama instance
	Lm = llama.LLama("")

	assert.Equal(t, llama.LLama(""), Lm)

	// Restore original value
	Lm = originalLm
}
