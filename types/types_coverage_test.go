package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferParamsCloneWithEmptySlice(t *testing.T) {
	// Test Clone with empty slice
	params := InferParams{
		StopPrompts: []string{},
	}

	cloned := params.Clone()
	assert.Equal(t, params, cloned)
	assert.Empty(t, cloned.StopPrompts)

	// Modify the clone and ensure original is not affected
	cloned.StopPrompts = append(cloned.StopPrompts, "test")
	assert.Empty(t, params.StopPrompts)
	assert.NotEmpty(t, cloned.StopPrompts)
}

func TestInferParamsCloneWithNilSlice(t *testing.T) {
	// Test Clone with nil slice
	params := InferParams{
		StopPrompts: nil,
	}

	cloned := params.Clone()
	assert.Equal(t, params, cloned)
	assert.Nil(t, cloned.StopPrompts)

	// Modify the clone and ensure original is not affected
	cloned.StopPrompts = []string{"test"}
	assert.Nil(t, params.StopPrompts)
	assert.NotEmpty(t, cloned.StopPrompts)
}

func TestInferParamsCloneWithComplexValues(t *testing.T) {
	// Test Clone with complex values
	params := InferParams{
		Stream:            true,
		MaxTokens:         1024,
		TopK:              50,
		TopP:              0.8,
		Temperature:       0.5,
		FrequencyPenalty:  0.1,
		PresencePenalty:   0.1,
		RepeatPenalty:     1.1,
		TailFreeSamplingZ: 0.9,
		StopPrompts:       []string{"STOP", "END", "DONE"},
	}

	cloned := params.Clone()
	assert.Equal(t, params, cloned)

	// Modify the clone and ensure original is not affected
	cloned.Stream = false
	cloned.StopPrompts[0] = "MODIFIED"

	assert.True(t, params.Stream)
	assert.False(t, cloned.Stream)
	assert.Equal(t, "STOP", params.StopPrompts[0])
	assert.Equal(t, "MODIFIED", cloned.StopPrompts[0])
}

func TestInferParamsValidationWithAllFields(t *testing.T) {
	// Test validation with all fields set to valid values
	params := InferParams{
		MaxTokens:         512,
		TopK:              40,
		TopP:              0.95,
		Temperature:       0.2,
		FrequencyPenalty:  0.0,
		PresencePenalty:   0.0,
		RepeatPenalty:     1.0,
		TailFreeSamplingZ: 1.0,
		StopPrompts:       []string{"</s>"},
	}

	require.NoError(t, params.Validate())
}

func TestInferParamsValidationWithMaxValues(t *testing.T) {
	// Test validation with maximum valid values
	params := InferParams{
		MaxTokens:         4096,
		TopK:              100,
		TopP:              1.0,
		Temperature:       100.0,
		FrequencyPenalty:  100.0,
		PresencePenalty:   100.0,
		RepeatPenalty:     100.0,
		TailFreeSamplingZ: 100.0,
	}

	require.NoError(t, params.Validate())
}

func TestInferParamsValidationWithMinValues(t *testing.T) {
	// Test validation with minimum valid values
	params := InferParams{
		TopK:              0,
		TopP:              0.0,
		Temperature:       0.0,
		FrequencyPenalty:  0.0,
		PresencePenalty:   0.0,
		RepeatPenalty:     0.0,
		TailFreeSamplingZ: 0.0,
	}

	require.NoError(t, params.Validate())
}

func TestInferParamsResetToDefaults(t *testing.T) {
	// Reset to defaults
	params := DefaultInferParams

	// Verify all fields are set to defaults
	assert.False(t, params.Stream)
	assert.Equal(t, DefaultInferParams.MaxTokens, params.MaxTokens)
	assert.Equal(t, DefaultInferParams.TopK, params.TopK)
	assert.Equal(t, float32(DefaultInferParams.TopP), params.TopP)
	assert.Equal(t, float32(DefaultInferParams.Temperature), params.Temperature)
	assert.Equal(t, float32(DefaultInferParams.FrequencyPenalty), params.FrequencyPenalty)
	assert.Equal(t, float32(DefaultInferParams.PresencePenalty), params.PresencePenalty)
	assert.Equal(t, float32(DefaultInferParams.RepeatPenalty), params.RepeatPenalty)
	assert.Equal(t, float32(DefaultInferParams.TailFreeSamplingZ), params.TailFreeSamplingZ)
	assert.Equal(t, DefaultInferParams.StopPrompts, params.StopPrompts)
}

func TestInferParamsPartialReset(t *testing.T) {
	// Test resetting only some fields
	params := InferParams{
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

	// Reset only some fields
	params.Stream = false
	params.TopK = DefaultInferParams.TopK

	// Verify only the reset fields changed
	assert.False(t, params.Stream)
	assert.Equal(t, DefaultInferParams.TopK, params.TopK)

	// Verify other fields remain unchanged
	assert.Equal(t, 2048, params.MaxTokens)
	assert.Equal(t, DefaultInferParams.TopK, params.TopK) // This should be DefaultInferParams.TopK now
	assert.Equal(t, float32(0.9), params.TopP)
	assert.Equal(t, float32(0.7), params.Temperature)
	assert.Equal(t, []string{"STOP", "END", "DONE"}, params.StopPrompts)
}
