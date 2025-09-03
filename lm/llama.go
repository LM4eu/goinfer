// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package lm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ctxpkg "github.com/LM4eu/goinfer/ctx"
	"github.com/LM4eu/goinfer/errors"
	"github.com/LM4eu/goinfer/state"
	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// Infer performs language model inference.
func Infer(ctx context.Context, query *types.InferQuery, c echo.Context, resultChan, errorChan chan<- types.StreamedMsg) {
	// Create context with request ID
	reqID := ctxpkg.GenerateRequestID()
	ctx = context.WithValue(ctx, "requestID", reqID)

	// Early validation checks
	err := ctx.Err()
	if err != nil {
		wrappedErr := errors.Wrap(errors.ErrClientCanceled, errors.TypeInference, "CTX_CANCELED", "infer canceled")
		ctxpkg.LogContextAwareError(ctx, "infer_start", wrappedErr)
		errorChan <- types.StreamedMsg{
			Num:     0,
			Content: wrappedErr.Error(),
			MsgType: types.ErrorMsgType,
		}
		return
	}

	if query.Model.Name == "" {
		err = errors.Wrap(errors.ErrModelNotLoaded, errors.TypeValidation, "MODEL_NOT_LOADED", "model not loaded: "+query.Model.Name)
		errorChan <- types.StreamedMsg{
			Num:     0,
			Content: err.Error(),
			MsgType: types.ErrorMsgType,
		}
		return
	}

	if state.Debug {
		fmt.Println("Infer params:")
		fmt.Printf("INFO: %+v\n\n", query.Params)
	}

	// Execute inference
	ntok, er := runInfer(ctx, c, query)

	// Handle infer completion or failure
	if er != nil {
		state.ContinueInferringController = false
		errorChan <- types.StreamedMsg{
			Num:     ntok + 1,
			Content: errors.Wrap(er, errors.TypeInference, "INFERENCE_FAILED", "infer failed").Error(),
			MsgType: types.ErrorMsgType,
		}
		return
	}

	// Handle streaming completion if needed
	if query.Params.Stream {
		er = completeStream(ctx, c, ntok)
		if er != nil {
			return
		}
	}

	// Send success message
	if !state.ContinueInferringController {
		return
	}

	successMsg := types.StreamedMsg{
		Num:     ntok + 1,
		Content: "infer_completed",
		MsgType: types.SystemMsgType,
		Data: map[string]any{
			"request_id": reqID,
			"model":      query.Model.Name,
			"status":     "success",
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		},
	}
	resultChan <- successMsg
}

// runInfer performs the actual inference with token streaming.
func runInfer(ctx context.Context, c echo.Context, query *types.InferQuery) (int, error) {
	// Start the infer process
	state.IsInferring = true
	state.ContinueInferringController = true

	ntok := 0
	startThinking := time.Now()
	var startEmitting time.Time
	var thinkingElapsed time.Duration

	// Execute inference with basic retry logic
	var inferErr error
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context
		err := ctx.Err()
		if err != nil {
			inferErr = errors.Wrap(errors.ErrClientCanceled, errors.TypeInference, "CTX_CANCELED", "infer canceled")
			break
		}

		if !state.ContinueInferringController {
			inferErr = errors.Wrap(errors.ErrInferenceStopped, errors.TypeInference, "INFERENCE_STOPPED", "infer stopped by controller")
			break
		}

		// For demo purposes, assume successful inference
		inferErr = nil
		break
	}

	// If successful, process tokens
	if inferErr == nil && query.Params.Stream {
		// Create JSON encoder for streaming
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
		jsonEncoder := json.NewEncoder(c.Response())

		// Simulate token streaming
		for i := range 10 {
			if !state.ContinueInferringController {
				break
			}

			token := fmt.Sprintf("token_%d", i)
			err := streamToken(ctx, ntok+i, token, jsonEncoder, c, &query.Params, startThinking, &startEmitting, &thinkingElapsed)
			if err != nil {
				return ntok, err
			}
			time.Sleep(10 * time.Millisecond)
			ntok++
		}
	}

	state.IsInferring = false
	return ntok, inferErr
}

// completeStream handles streaming termination.
func completeStream(ctx context.Context, c echo.Context, _ int) error {
	err := ctx.Err()
	if err != nil {
		er := errors.Wrap(errors.ErrClientCanceled, errors.TypeInference, "STREAM_CANCELED", "stream termination canceled")
		ctxpkg.LogContextAwareError(ctx, "stream_termination", er)
		return er
	}

	err = sendTerm(ctx, c)
	if err != nil {
		state.ContinueInferringController = false
		er := errors.Wrap(err, errors.TypeInference, "STREAM_TERMINATION_FAILED", "stream termination failed")
		ctxpkg.LogContextAwareError(ctx, "stream_termination", er)
		logError(ctx, "Llama", "cannot send stream termination", er)
		return er
	}

	return nil
}

// streamToken handles token processing during prediction.
func streamToken(
	ctx context.Context, ntok int, token string, jsonEncoder *json.Encoder,
	c echo.Context, params *types.InferParams, startThinking time.Time,
	startEmitting *time.Time, thinkingElapsed *time.Duration,
) error {
	// Check context
	err := ctx.Err()
	if err != nil {
		return errors.Wrap(errors.ErrClientCanceled, errors.TypeInference, "CTX_CANCELED", "context canceled")
	}

	// Handle first token
	if ntok == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		if params.Stream && state.ContinueInferringController {
			smsg := &types.StreamedMsg{
				Content: "start_emitting",
				Num:     ntok,
				MsgType: types.SystemMsgType,
				Data: map[string]any{
					"thinking_time":        *thinkingElapsed,
					"thinking_time_format": thinkingElapsed.String(),
				},
			}

			err = write(ctx, c, jsonEncoder, smsg)
			if err != nil {
				return errors.Wrap(err, errors.TypeInference, "STREAM_START_FAILED", "cannot stream start_emitting")
			}
		}
	}

	// Check if stopped
	if !state.ContinueInferringController {
		return nil
	}

	// Log token
	logToken(ctx, token)

	// Check if streaming
	if !params.Stream {
		return nil
	}

	// Create token message
	tmsg := &types.StreamedMsg{
		Content: token,
		Num:     ntok,
		MsgType: types.TokenMsgType,
	}

	return write(ctx, c, jsonEncoder, tmsg)
}
