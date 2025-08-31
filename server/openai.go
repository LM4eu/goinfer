// Package server provides HTTP handlers for the GoInfer proxy, including an OpenAI‑compatible API.
package server

import (
	"net/http"

	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// OpenAIChatRequest mirrors the OpenAI /v1/chat/completions payload.
type OpenAIChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Temperature *float32      `json:"temperature,omitempty"`
	TopP        *float32      `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatMessage represents a single message in the OpenAI chat format.
type ChatMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // prompt text
}

// OpenAIChatResponse is the response format sent back to the client.
type OpenAIChatResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"` // "chat.completion"
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIChatChoice   `json:"choices"`
	Usage   *OpenAIChatUsage     `json:"usage,omitempty"`
}

// OpenAIChatChoice represents a single choice in the response.
type OpenAIChatChoice struct {
	Index   int `json:"index"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// OpenAIChatUsage contains token usage statistics.
type OpenAIChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openAIChatHandler converts an OpenAI‑style request into an internal InferQuery,
// runs the inference via lm.Infer, and streams or returns the final result.
func openAIChatHandler(echoCtx echo.Context) error {
	// Parse request body.
	var req OpenAIChatRequest
	if err := echoCtx.Bind(&req); err != nil {
		return echoCtx.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid JSON payload",
		})
	}

	// Build internal InferQuery.
	inferQry := &types.InferQuery{
		Prompt: "",
		Model:  types.Model{Name: req.Model},
		Params: types.DefaultInferParams,
	}
	// Concatenate messages into a single prompt.
	for _, m := range req.Messages {
		inferQry.Prompt += m.Content + "\n"
	}
	if req.MaxTokens != nil {
		inferQry.Params.Generation.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		inferQry.Params.Sampling.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		inferQry.Params.Sampling.TopP = *req.TopP
	}
	inferQry.Params.Stream = req.Stream

	// Execute inference using the existing helper.
	result, err := execute(echoCtx, echoCtx.Request().Context(), inferQry)
	if err != nil {
		return echoCtx.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Non‑streaming response.
	if !req.Stream {
		resp := OpenAIChatResponse{
			ID:      "chatcmpl-" + result.Content, // placeholder ID
			Object:  "chat.completion",
			Created: result.Data["timestamp"].(int64),
			Model:   inferQry.Model.Name,
			Choices: []OpenAIChatChoice{
				{
					Index: 0,
					Message: struct {
						Role    string "json:\"role\""
						Content string "json:\"content\""
					}{
						Role:    "assistant",
						Content: result.Content,
					},
					FinishReason: "stop",
				},
			},
		}
		if usage, ok := result.Data["usage"]; ok {
			resp.Usage = usage.(*OpenAIChatUsage)
		}
		return echoCtx.JSON(http.StatusOK, resp)
	}

	// Streaming case – the execute helper already writes JSON chunks,
	// so we simply return nil to let Echo finish the response.
	return nil
}