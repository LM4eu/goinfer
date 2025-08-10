package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoInferConf(t *testing.T) {
	// Test creating a GoInferConf
	conf := GoInferConf{
		ModelsDir:  "/models",
		WebServer:  WebServerConf{EnableApiOpenAi: true, Origins: []string{"localhost", "example.com"}, ApiKey: "test-key"},
		OpenAiConf: OpenAiConf{},
	}

	// Test field values
	assert.Equal(t, "/models", conf.ModelsDir)
	assert.Equal(t, []string{"localhost", "example.com"}, conf.WebServer.Origins)
	assert.Equal(t, "test-key", conf.WebServer.ApiKey)
	assert.True(t, conf.WebServer.EnableApiOpenAi)
}

func TestTask(t *testing.T) {
	// Test creating a Task
	task := InferQuery{
		Prompt:    "Please read this prompt",
		ModelConf: ModelConf{Name: "test-model", Ctx: 2048},
		InferParams: InferParams{
			TopK:        40,
			TopP:        0.95,
			MinP:        0.05,
			Temperature: 0.2,
		},
	}

	// Test field values
	assert.Equal(t, "Please read this prompt", task.Prompt)
	assert.Equal(t, "test-model", task.ModelConf.Name)
	assert.Equal(t, 2048, task.ModelConf.Ctx)
	assert.Equal(t, 40, task.InferParams.TopK)
	assert.Equal(t, float32(0.95), task.InferParams.TopP)
	assert.Equal(t, float32(0.2), task.InferParams.Temperature)
}

func TestModelConf(t *testing.T) {
	// Test creating a ModelConf
	modelConf := ModelConf{
		Name: "test-model",
		Ctx:  2048,
	}

	// Test field values
	assert.Equal(t, "test-model", modelConf.Name)
	assert.Equal(t, 2048, modelConf.Ctx)

	// Test creating a ModelConf with minimal values
	minimalModelConf := ModelConf{Name: "minimal-model"}
	assert.Equal(t, "minimal-model", minimalModelConf.Name)
	assert.Equal(t, 0, minimalModelConf.Ctx)
}

func TestTemplateInfo(t *testing.T) {
	// Test creating a TemplateInfo
	templateInfo := TemplateInfo{
		Name: "test-template",
		Ctx:  2048,
	}

	// Test field values
	assert.Equal(t, "test-template", templateInfo.Name)
	assert.Equal(t, 2048, templateInfo.Ctx)
}

func TestMsgTypeConstants(t *testing.T) {
	// Test MsgType constants
	assert.Equal(t, "token", string(TokenMsgType))
	assert.Equal(t, "system", string(SystemMsgType))
	assert.Equal(t, "error", string(ErrorMsgType))
}

func TestStreamedMessage(t *testing.T) {
	// Test creating a StreamedMessage
	data := map[string]any{
		"model":     "test-model",
		"timestamp": 1234567890,
	}

	streamedMsg := StreamedMessage{
		Content: "test content",
		Num:     10,
		MsgType: TokenMsgType,
		Data:    data,
	}

	// Test field values
	assert.Equal(t, "test content", streamedMsg.Content)
	assert.Equal(t, 10, streamedMsg.Num)
	assert.Equal(t, TokenMsgType, streamedMsg.MsgType)
	assert.Equal(t, data, streamedMsg.Data)

	// Test creating a StreamedMessage without data
	minimalStreamedMsg := StreamedMessage{
		Content: "minimal content",
		MsgType: SystemMsgType,
	}
	assert.Equal(t, "minimal content", minimalStreamedMsg.Content)
	assert.Equal(t, 0, minimalStreamedMsg.Num)
	assert.Equal(t, SystemMsgType, minimalStreamedMsg.MsgType)
	assert.Nil(t, minimalStreamedMsg.Data)
}

func TestApiTypeConstants(t *testing.T) {
	// Test ApiType constants
	assert.Equal(t, "llama", string(Llama))
	assert.Equal(t, "openai", string(OpenAi))
}

func TestInferenceStatsJSONMarshaling(t *testing.T) {
	// Test JSON marshaling and unmarshaling for InferenceStats
	stats := InferenceStats{
		ThinkingTime:       1.5,
		ThinkingTimeFormat: "seconds",
		EmitTime:           0.5,
		EmitTimeFormat:     "seconds",
		TotalTime:          2.0,
		TotalTimeFormat:    "seconds",
		TokensPerSecond:    10.5,
		TotalTokens:        21,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(stats)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshalledStats InferenceStats
	err = json.Unmarshal(jsonData, &unmarshalledStats)
	require.NoError(t, err)

	// Verify the unmarshalled data matches the original
	assert.Equal(t, stats, unmarshalledStats)
}

func TestInferenceResultJSONMarshaling(t *testing.T) {
	// Test JSON marshaling and unmarshaling for InferenceResult
	result := InferenceResult{
		Text: "test result",
		Stats: InferenceStats{
			TotalTokens: 10,
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(result)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshalledResult InferenceResult
	err = json.Unmarshal(jsonData, &unmarshalledResult)
	require.NoError(t, err)

	// Verify the unmarshalled data matches the original
	assert.Equal(t, result, unmarshalledResult)
}

func TestTaskJSONMarshaling(t *testing.T) {
	// Test JSON marshaling and unmarshaling for Task
	task := InferQuery{
		Prompt:      "test-template",
		ModelConf:   ModelConf{Name: "test-model"},
		InferParams: DefaultInferParams,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(task)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshalledTask InferQuery
	err = json.Unmarshal(jsonData, &unmarshalledTask)
	require.NoError(t, err)

	// Verify the unmarshalled data matches the original
	assert.Equal(t, task, unmarshalledTask)
}
