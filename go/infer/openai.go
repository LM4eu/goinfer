// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"log/slog"
	"net/http"
	"strconv"
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
		return gie.HandleValidationError(c, gie.Wrap(err, gie.Invalid, "failed to parse OpenAI request"))
	}

	// Reuse the existing inference flow through Infer.
	// Create channels for streaming results.
	resChan := make(chan StreamedMsg)
	errChan := make(chan StreamedMsg)

	// Execute inference with request context.
	err = inf.forwardInference(c.Request().Context(), query, c, resChan, errChan)
	if err != nil {
		return gie.HandleInferenceError(c, gie.Wrap(err, gie.InferErr, "proxy manager forward inference failed"))
	}

	// Wait for the first result or error to respond.
	select {
	case res := <-resChan:
		return c.JSON(http.StatusOK, res.Data)
	case err := <-errChan:
		return gie.HandleInferenceError(c, err.Error)
	}
}

// parseOpenAIRequest converts an OpenAI chat completion request into an InferQuery.
func parseOpenAIRequest(c echo.Context) (*Query, error) {
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
		slog.ErrorContext(c.Request().Context(), "Failed to bind OpenAI request", "error", err)
		return nil, gie.Wrap(err, gie.Invalid, "failed to bind OpenAI request")
	}

	// Determine prompt: if messages provided, concatenate contents, else use fallback prompt.
	prompt := req.Prompt
	if len(req.Messages) > 0 {
		var builder strings.Builder
		for i, msg := range req.Messages {
			if msg.Role == "" {
				return nil, gie.New(gie.Invalid, "invalid message "+strconv.Itoa(i)+" missing role")
			}
			if msg.Content == "" {
				return nil, gie.New(gie.Invalid, "invalid message "+strconv.Itoa(i)+" missing content")
			}
			if i > 0 {
				_, _ = builder.WriteString("\n")
			}
			_, _ = builder.WriteString(msg.Content)
		}
		prompt = builder.String()
	}

	query := defaultQuery
	query.Prompt = prompt
	if req.Temperature != 0 {
		query.Temperature = float32(req.Temperature)
	}
	if req.MaxTokens != 0 {
		query.MaxTokens = req.MaxTokens
	}
	query.Stream = req.Stream

	return &query, nil
}
