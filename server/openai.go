package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func parseParams(m echo.Map) (string, string, string, types.InferenceParams, error) {
	params := lm.DefaultInferenceParams
	v, ok := m["model"]
	if !ok {
		return "", "", "", params, errors.New("provide a model")
	}
	model := v.(string)
	v, ok = m["messages"]
	if !ok {
		return "", "", "", params, errors.New("provide a messages array")
	}
	qmsgs := v.([]interface{})
	prompt := ""
	template := state.OpenAiConf.Template
	for _, m := range qmsgs {
		el := m.(map[string]interface{})
		role := el["role"].(string)
		content := el["content"].(string)
		if role == "system" {
			template = strings.Replace(template, "{system}", content, 1)
		} else if role == "user" {
			prompt = content
		}
	}
	v, ok = m["temperature"]
	if ok {
		params.Temperature = float32(v.(float64))
	}
	v, ok = m["top_p"]
	if ok {
		params.TopP = float32(v.(float64))
	}
	v, ok = m["top_k"]
	if ok {
		params.TopK = int(v.(float64))
	}
	v, ok = m["max_tokens"]
	if ok {
		params.Tokens = int(v.(float64))
	}
	v, ok = m["stop"]
	if ok {
		params.StopPrompts = strings.Join(v.([]string), ",")
	}
	v, ok = m["presence_penalty"]
	if ok {
		params.PresencePenalty = float32(v.(float64))
	}
	v, ok = m["frequency_penalty"]
	if ok {
		params.FrequencyPenalty = float32(v.(float64))
	}
	v, ok = m["repeat_penalty"]
	if ok {
		params.RepeatPenalty = float32(v.(float64))
	}
	v, ok = m["tfs_z"]
	if ok {
		params.TailFreeSamplingZ = float32(v.(float64))
	}
	return model, prompt, template, params, nil
}

// Create an Openai api for /v1/completion
func CreateCompletionHandler(c echo.Context) error {
	if state.IsInfering {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	fmt.Println("COMPLETION", m)
	model, prompt, template, params, err := parseParams(m)
	if err != nil {
		panic(err)
	}
	if state.LoadedModel != model {
		lm.LoadModel(model, lm.DefaultModelParams)
	}
	res, err := lm.Infer(prompt, template, params, c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	endres := types.OpenAiChatCompletion{
		ID:      "0",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []types.OpenAiChoice{
			{
				Index: 0,
				Message: types.OpenAiMessage{
					Role:    "assistant",
					Content: res.Text,
				},
				FinishReason: "stop",
			},
		},
		Usage: types.OpenAiUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      res.TotalTokens,
		},
	}
	fmt.Println("RES", endres)
	return c.JSON(http.StatusOK, endres)
}
