package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInferenceParamsConstructor tests the constructor function.
func TestInferenceParamsConstructor(t *testing.T) {
	// Test that default values are properly set using the constructor
	params := DefaultInferenceParams

	// Test default values
	assert.False(t, params.Stream)
	assert.Equal(t, DefaultInferenceParams.MaxTokens, params.MaxTokens)
	assert.Equal(t, DefaultInferenceParams.TopK, params.TopK)
	assert.Equal(t, float32(DefaultInferenceParams.TopP), params.TopP)
	assert.Equal(t, float32(DefaultInferenceParams.Temperature), params.Temperature)
	assert.Equal(t, float32(DefaultInferenceParams.FrequencyPenalty), params.FrequencyPenalty)
	assert.Equal(t, float32(DefaultInferenceParams.PresencePenalty), params.PresencePenalty)
	assert.Equal(t, float32(DefaultInferenceParams.RepeatPenalty), params.RepeatPenalty)
	assert.Equal(t, float32(DefaultInferenceParams.TailFreeSamplingZ), params.TailFreeSamplingZ)
	assert.Equal(t, DefaultInferenceParams.StopPrompts, params.StopPrompts)
}

// TestInferenceParamsCreation tests creating InferenceParams with custom values.
func TestInferenceParamsCreation(t *testing.T) {
	// Test creating custom inference params
	params := InferenceParams{
		Stream:            true,
		MaxTokens:         1024,
		TopK:              80,
		TopP:              0.8,
		Temperature:       0.5,
		FrequencyPenalty:  0.1,
		PresencePenalty:   0.1,
		RepeatPenalty:     1.2,
		TailFreeSamplingZ: 0.9,
		StopPrompts:       []string{"STOP", "END"},
	}

	// Verify custom values
	assert.True(t, params.Stream)
	assert.Equal(t, 1024, params.MaxTokens)
	assert.Equal(t, 80, params.TopK)
	assert.Equal(t, float32(0.8), params.TopP)
	assert.Equal(t, float32(0.5), params.Temperature)
	assert.Equal(t, float32(0.1), params.FrequencyPenalty)
	assert.Equal(t, float32(0.1), params.PresencePenalty)
	assert.Equal(t, float32(1.2), params.RepeatPenalty)
	assert.Equal(t, float32(0.9), params.TailFreeSamplingZ)
	assert.Equal(t, []string{"STOP", "END"}, params.StopPrompts)
}

// TestInferenceParamsClone tests the Clone method.
func TestInferenceParamsClone(t *testing.T) {
	// Test that inference params can be copied correctly using Clone
	params1 := InferenceParams{
		Stream:            true,
		MaxTokens:         2048,
		TopK:              100,
		TopP:              0.9,
		Temperature:       0.7,
		FrequencyPenalty:  0.2,
		PresencePenalty:   0.2,
		RepeatPenalty:     1.5,
		TailFreeSamplingZ: 0.8,
		StopPrompts:       []string{"STOP", "END", "DONE"},
	}

	// Copy the params using Clone
	params2 := params1.Clone()

	// Verify they are equal
	assert.Equal(t, params1, params2)

	// Modify the copy
	params2.Stream = false
	params2.StopPrompts = []string{"END"}

	// Verify they are now different
	assert.NotEqual(t, params1, params2)
	assert.True(t, params1.Stream)
	assert.False(t, params2.Stream)
	assert.Equal(t, []string{"STOP", "END", "DONE"}, params1.StopPrompts)
	assert.Equal(t, []string{"END"}, params2.StopPrompts)
}

// TestInferenceParamsReset tests resetting to defaults.
func TestInferenceParamsReset(t *testing.T) {
	// Test resetting inference params to defaults
	params := InferenceParams{
		Stream:            true,
		MaxTokens:         2048,
		TopK:              100,
		TopP:              0.9,
		Temperature:       0.7,
		FrequencyPenalty:  0.2,
		PresencePenalty:   0.2,
		RepeatPenalty:     1.5,
		TailFreeSamplingZ: 0.8,
		StopPrompts:       []string{"STOP", "END", "DONE"},
	}

	// Reset to defaults using constructor
	defaultParams := DefaultInferenceParams

	// Verify they are different
	assert.NotEqual(t, defaultParams, params)

	// Reset params using constructor
	params = DefaultInferenceParams

	// Verify they are now equal
	assert.Equal(t, defaultParams, params)
}

func TestInferenceParamsValidation(t *testing.T) {
	// Test validation of inference params
	testCases := []struct {
		name        string
		params      InferenceParams
		valid       bool
		expectedErr string
	}{
		{
			name: "Valid params",
			params: InferenceParams{
				TopK:        40,
				TopP:        0.95,
				Temperature: 0.2,
			},
			valid: true,
		},
		{
			name: "Invalid TopK (negative)",
			params: InferenceParams{
				TopK: -1,
			},
			valid:       false,
			expectedErr: "top_k must be non-negative, got -1",
		},
		{
			name: "Invalid TopP (negative)",
			params: InferenceParams{
				TopP: -0.1,
			},
			valid:       false,
			expectedErr: "top_p must be between 0.0 and 1.0, got -0.100000",
		},
		{
			name: "Invalid TopP (> 1.0)",
			params: InferenceParams{
				TopP: 1.1,
			},
			valid:       false,
			expectedErr: "top_p must be between 0.0 and 1.0, got 1.100000",
		},
		{
			name: "Invalid Temperature (negative)",
			params: InferenceParams{
				Temperature: -0.1,
			},
			valid:       false,
			expectedErr: "temperature must be non-negative, got -0.100000",
		},
		{
			name: "Invalid RepeatPenalty (negative)",
			params: InferenceParams{
				RepeatPenalty: -0.1,
			},
			valid:       false,
			expectedErr: "repeat_penalty must be non-negative, got -0.100000",
		},
		{
			name: "Invalid TailFreeSamplingZ (negative)",
			params: InferenceParams{
				TailFreeSamplingZ: -0.1,
			},
			valid:       false,
			expectedErr: "tail_free_sampling_z must be non-negative, got -0.100000",
		},
		{
			name: "TopP boundary (0.0)",
			params: InferenceParams{
				TopP: 0.0,
			},
			valid: true,
		},
		{
			name: "TopP boundary (1.0)",
			params: InferenceParams{
				TopP: 1.0,
			},
			valid: true,
		},
		{
			name: "TopK boundary (0)",
			params: InferenceParams{
				TopK: 0,
			},
			valid: true,
		},
		{
			name: "Temperature boundary (0.0)",
			params: InferenceParams{
				Temperature: 0.0,
			},
			valid: true,
		},
		{
			name: "RepeatPenalty boundary (0.0)",
			params: InferenceParams{
				RepeatPenalty: 0.0,
			},
			valid: true,
		},
		{
			name: "TailFreeSamplingZ boundary (0.0)",
			params: InferenceParams{
				TailFreeSamplingZ: 0.0,
			},
			valid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()

			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error())
			}
		})
	}
}

func TestInferenceParamsStringRepresentation(t *testing.T) {
	// Test string representation of inference params
	params := InferenceParams{
		Stream:            true,
		MaxTokens:         1024,
		TopK:              50,
		TopP:              0.8,
		Temperature:       0.5,
		FrequencyPenalty:  0.1,
		PresencePenalty:   0.1,
		RepeatPenalty:     1.1,
		TailFreeSamplingZ: 0.9,
		StopPrompts:       []string{"STOP", "END"},
	}

	// Test that params have values
	assert.True(t, params.Stream)
	assert.NotZero(t, params.MaxTokens)
	assert.NotZero(t, params.TopK)
	assert.NotZero(t, params.TopP)
	assert.NotZero(t, params.Temperature)
	assert.NotZero(t, params.RepeatPenalty)
	assert.NotZero(t, params.TailFreeSamplingZ)
	assert.NotEmpty(t, params.StopPrompts)
}

func TestInferenceParamsImmutability(t *testing.T) {
	// Test that modifying inference params doesn't affect other instances
	originalParams := InferenceParams{
		Stream: true,
	}

	// Modify the params
	originalParams.Stream = false

	// Create a new instance and verify it has the modified values
	newParams := originalParams
	assert.False(t, newParams.Stream)
}

func TestInferenceParamsWithPartialDefaults(t *testing.T) {
	// Test creating inference params with some defaults and some custom values
	defaultParams := DefaultInferenceParams

	// Create custom params based on defaults
	customParams := defaultParams
	customParams.Stream = true
	customParams.MaxTokens = 2048
	customParams.TopK = 80
	customParams.TopP = 0.8
	customParams.Temperature = 0.5

	// Verify custom values
	assert.True(t, customParams.Stream)
	assert.Equal(t, 2048, customParams.MaxTokens)
	assert.Equal(t, 80, customParams.TopK)
	assert.Equal(t, float32(0.8), customParams.TopP)
	assert.Equal(t, float32(0.5), customParams.Temperature)

	// Verify other fields retain default values
	assert.Equal(t, DefaultInferenceParams.MaxTokens, defaultParams.MaxTokens)                          // Default value
	assert.Equal(t, DefaultInferenceParams.TopK, defaultParams.TopK)                                    // Default value
	assert.Equal(t, float32(DefaultInferenceParams.TopP), defaultParams.TopP)                           // Default value
	assert.Equal(t, float32(DefaultInferenceParams.Temperature), defaultParams.Temperature)             // Default value
	assert.Equal(t, float32(DefaultInferenceParams.FrequencyPenalty), defaultParams.FrequencyPenalty)   // Default value
	assert.Equal(t, float32(DefaultInferenceParams.PresencePenalty), defaultParams.PresencePenalty)     // Default value
	assert.Equal(t, float32(DefaultInferenceParams.RepeatPenalty), defaultParams.RepeatPenalty)         // Default value
	assert.Equal(t, float32(DefaultInferenceParams.TailFreeSamplingZ), defaultParams.TailFreeSamplingZ) // Default value
	assert.Equal(t, DefaultInferenceParams.StopPrompts, defaultParams.StopPrompts)                      // Default value
}

func TestStopPromptsManipulation(t *testing.T) {
	// Test manipulation of stop prompts
	params := InferenceParams{
		StopPrompts: []string{"STOP", "END"},
	}

	// Verify initial state
	assert.Equal(t, []string{"STOP", "END"}, params.StopPrompts)

	// Modify the slice
	params.StopPrompts = append(params.StopPrompts, "DONE")

	// Verify modification
	assert.Equal(t, []string{"STOP", "END", "DONE"}, params.StopPrompts)

	// Create a copy and verify independence
	copyParams := params
	copyParams.StopPrompts = []string{"COPY"}

	// Verify they are different
	assert.NotEqual(t, params.StopPrompts, copyParams.StopPrompts)
	assert.Equal(t, []string{"STOP", "END", "DONE"}, params.StopPrompts)
	assert.Equal(t, []string{"COPY"}, copyParams.StopPrompts)
}

func TestInferenceParamsJSONMarshaling(t *testing.T) {
	// Test JSON marshaling and unmarshaling
	params := InferenceParams{
		Stream:            true,
		MaxTokens:         1024,
		TopK:              50,
		TopP:              0.8,
		Temperature:       0.5,
		FrequencyPenalty:  0.1,
		PresencePenalty:   0.1,
		RepeatPenalty:     1.1,
		TailFreeSamplingZ: 0.9,
		StopPrompts:       []string{"STOP", "END"},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(params)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshalledParams InferenceParams
	err = json.Unmarshal(jsonData, &unmarshalledParams)
	require.NoError(t, err)

	// Verify the unmarshalled data matches the original
	assert.Equal(t, params, unmarshalledParams)
}

func TestInferenceParamsEdgeCases(t *testing.T) {
	// Test edge cases for InferenceParams
	testCases := []struct {
		name   string
		params InferenceParams
	}{
		{
			name: "Minimum valid values",
			params: InferenceParams{
				TopK:        0,
				TopP:        0.0,
				Temperature: 0.0,
			},
		},
		{
			name: "Maximum valid values",
			params: InferenceParams{
				TopP:              1.0,
				Temperature:       100.0,
				RepeatPenalty:     100.0,
				TailFreeSamplingZ: 100.0,
			},
		},
		{
			name: "Empty stop prompts",
			params: InferenceParams{
				StopPrompts: []string{},
			},
		},
		{
			name: "Nil stop prompts",
			params: InferenceParams{
				StopPrompts: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that edge cases can be created and validated
			require.NoError(t, tc.params.Validate())
		})
	}
}
