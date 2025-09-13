// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// Message represents a chat message in OpenAI format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// handleChatCompletions handles OpenAI compatible chat completion requests.
func (inf *Infer) handleChatCompletions(c echo.Context) error {
	// Parse the OpenAI request into an InferQuery.
	query, err := parseOpenAIRequest(c)
	if err != nil {
		return gie.HandleValidationError(c, gie.Wrap(err, gie.TypeValidation, "OPENAI_PARSE_ERROR", "failed to parse OpenAI request"))
	}

	// Reuse the existing inference flow through Infer.
	// Create channels for streaming results.
	resChan := make(chan StreamedMsg)
	errChan := make(chan StreamedMsg)

	// Execute inference with request context.
	err = inf.forwardInference(c.Request().Context(), query, c, resChan, errChan)
	if err != nil {
		return gie.HandleInferenceError(c, gie.Wrap(err, gie.TypeInference, "PROXY_FORWARD_FAILED", "proxy manager forward inference failed"))
	}

	// Wait for the first result or error to respond.
	select {
	case res := <-resChan:
		return c.JSON(http.StatusOK, res.Data)
	case err := <-errChan:
		return gie.HandleInferenceError(c, gie.Wrap(gie.ErrInferFailed, gie.TypeInference, "INFERENCE_ERROR", err.Content))
	}
}

// parseOpenAIRequest converts an OpenAI chat completion request into an InferQuery.
func parseOpenAIRequest(c echo.Context) (*InferQuery, error) {
	// The OpenAI API expects a JSON body with fields such as model, messages, temperature, etc.
	// For simplicity we map a subset of these fields to the internal InferQuery.
	var req struct {
		Model       string    `json:"model"`
		Prompt      string    `json:"prompt"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature"`
		MaxTokens   int       `json:"max_tokens"`
		Stream      bool      `json:"stream"`
	}
	err := c.Bind(&req)
	if err != nil {
		return nil, gie.Wrap(err, gie.TypeValidation, "OPENAI_BIND_ERROR", "failed to bind OpenAI request")
	}

	// Determine prompt: if messages provided, concatenate contents, else use fallback prompt.
	prompt := req.Prompt
	if len(req.Messages) > 0 {
		var builder strings.Builder
		for i, msg := range req.Messages {
			if msg.Role == "" {
				return nil, gie.Wrap(gie.ErrInvalidParams, gie.TypeValidation, "INVALID_MESSAGE_ROLE", fmt.Sprintf("message %d missing role", i))
			}
			if msg.Content == "" {
				return nil, gie.Wrap(gie.ErrInvalidParams, gie.TypeValidation, "INVALID_MESSAGE_CONTENT", fmt.Sprintf("message %d missing content", i))
			}
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(msg.Content)
		}
		prompt = builder.String()
	}

	query := &InferQuery{
		Prompt: prompt,
		Model:  Model{Name: req.Model},
		Params: DefaultInferParams,
	}
	if req.Temperature != 0 {
		query.Params.Sampling.Temperature = float32(req.Temperature)
	}
	if req.MaxTokens != 0 {
		query.Params.Generation.MaxTokens = req.MaxTokens
	}
	query.Params.Stream = req.Stream

	return query, nil
}
