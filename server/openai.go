// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package server

import (
	"net/http"

	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// handleChatCompletions handles OpenAI compatible chat completion requests.
func (pi *ProxyInfer) handleChatCompletions(c echo.Context) error {
	// Parse the OpenAI request into an InferQuery.
	query, err := parseOpenAIRequest(c)
	if err != nil {
		return gie.HandleValidationError(c, gie.Wrap(err, gie.TypeValidation, "OPENAI_PARSE_ERROR", "failed to parse OpenAI request"))
	}

	// Reuse the existing inference flow through ProxyInfer.
	// Create channels for streaming results.
	resChan := make(chan types.StreamedMsg)
	errChan := make(chan types.StreamedMsg)

	// Execute inference with request context using ProxyInfer.
	err = pi.forwardInference(c.Request().Context(), query, c, resChan, errChan)
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
func parseOpenAIRequest(c echo.Context) (*types.InferQuery, error) {
	// The OpenAI API expects a JSON body with fields such as model, messages, temperature, etc.
	// For simplicity we map a subset of these fields to the internal InferQuery.
	var req struct {
		Model       string  `json:"model"`
		Prompt      string  `json:"prompt"` // fallback if messages not used
		Temperature float64 `json:"temperature"`
		MaxTokens   int     `json:"max_tokens"`
		Stream      bool    `json:"stream"`
	}
	err := c.Bind(&req)
	if err != nil {
		return nil, gie.Wrap(err, gie.TypeValidation, "OPENAI_BIND_ERROR", "failed to bind OpenAI request")
	}

	query := &types.InferQuery{
		Prompt: req.Prompt,
		Model:  types.Model{Name: req.Model},
		Params: types.DefaultInferParams,
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
