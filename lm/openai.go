// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package lm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/LM4eu/goinfer/state"
	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

type (
	// OpenAI response structures (reduced).
	ChatCompletion struct {
		ID      string   `json:"id"`
		Object  string   `json:"object"`
		Model   string   `json:"model"`
		Choices []Choice `json:"choices"`
		Usage   struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Created int64 `json:"created"`
	}

	Choice struct {
		Message      string `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	}
)

// inferOpenAI performs OpenAI model inference.
func inferOpenAI(ctx context.Context, query types.InferQuery, c echo.Context, resultChan chan<- ChatCompletion, errorChan chan<- error) {
	// Check if context is already canceled
	err := ctx.Err()
	if err != nil {
		errorChan <- fmt.Errorf("context canceled at start of inference: %w", err)
		return
	}

	// Create response directly
	result := ChatCompletion{
		ID:      "1",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   query.Model.Name,
		Choices: []Choice{
			{
				Index:        0,
				Message:      "OpenAI response",
				FinishReason: "stop",
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      1,
		},
	}

	resultChan <- result
}

// streamTokenOpenAI streams tokens to the client.
func streamTokenOpenAI(ctx context.Context, ntok int, token string, jsonEncoder *json.Encoder, c echo.Context, query types.InferQuery, startThinking time.Time, thinkingElapsed *time.Duration, startEmitting *time.Time) error {
	// Check context
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("context canceled: %w", err)
	}

	// Handle first token
	if ntok == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		err := ctx.Err()
		if err != nil {
			return fmt.Errorf("context canceled: %w", err)
		}

		if query.Params.Stream && state.ContinueInferringController {
			// Create start message
			smsg := ChatCompletion{
				ID:      strconv.Itoa(ntok),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   query.Model.Name,
				Choices: []Choice{
					{
						Index:        ntok,
						Message:      "start_emitting",
						FinishReason: "",
					},
				},
			}

			// Convert to unified message
			unifiedMsg := &types.StreamedMsg{
				Content: "start_emitting",
				Num:     ntok,
				MsgType: types.SystemMsgType,
				Data: map[string]any{
					"openai_system": smsg,
				},
			}

			err = write(ctx, c, jsonEncoder, unifiedMsg)
			if err != nil {
				logError(ctx, "OpenAI", "cannot emit start message", err)
				state.ContinueInferringController = false
				return fmt.Errorf("failed to send start emitting message: %w", err)
			}

			time.Sleep(2 * time.Millisecond)
		}
	}

	// Check if stopped
	if !state.ContinueInferringController {
		return nil
	}

	// Log token
	logToken(ctx, token)

	// Check if streaming
	if !query.Params.Stream {
		return nil
	}

	// Create token message
	tmsg := ChatCompletion{
		ID:      strconv.Itoa(ntok),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   query.Model.Name,
		Choices: []Choice{
			{
				Index:        ntok,
				Message:      token,
				FinishReason: "",
			},
		},
	}

	// Create unified message
	unifiedMsg := &types.StreamedMsg{
		Content: tmsg.Choices[0].Message,
		Num:     ntok,
		MsgType: types.TokenMsgType,
		Data: map[string]any{
			"openai_delta": tmsg,
		},
	}

	return write(ctx, c, jsonEncoder, unifiedMsg)
}

// // sendOpenAITerm sends termination message.
// func sendOpenAITerm(ctx context.Context, c echo.Context) error {
// 	// Check context
// 	err := ctx.Err()
// 	if err != nil {
// 		logError(ctx, "OpenAI", "context canceled during stream termination", err)
// 		return fmt.Errorf("context canceled: %w", err)
// 	}
//
// 	// Create termination message
// 	unifiedMsg := &types.StreamedMsg{
// 		Content: "[DONE]",
// 		Num:     -1,
// 		MsgType: types.SystemMsgType,
// 		Data: map[string]any{
// 			"openai_termination": true,
// 		},
// 	}
//
// 	// Send termination message
// 	err = write(ctx, c, nil, unifiedMsg)
// 	if err != nil {
// 		logError(ctx, "OpenAI", "failed to send stream termination", err)
// 		return fmt.Errorf("failed to send stream termination: %w", err)
// 	}
//
// 	return nil
// }
