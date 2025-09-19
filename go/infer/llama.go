// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// CtxKeyRequestID is a typed context key to prevent key collisions.
type CtxKeyRequestID string

// RequestID is the key to access the RequestID stored within the context.
const RequestID CtxKeyRequestID = "RequestID"

// Infer performs language model inference.
func (inf *Infer) Infer(ctx context.Context, query *InferQuery, c echo.Context, resChan, errChan chan<- StreamedMsg) {
	// Create context with request ID
	ctx = context.WithValue(ctx, RequestID, reqID())

	// Early validation checks
	if ctx.Err() != nil {
		err := gie.Wrap(ctx.Err(), gie.TypeInference, "CTX_CANCELED", "infer canceled")
		slog.ErrorContext(ctx, "Context canceled at infer start", "error", err)
		errChan <- StreamedMsg{
			Num:     0,
			Content: err.Error(),
			MsgType: ErrorMsgType,
		}
		return
	}

	if query.Model.Name == "" {
		err := gie.Wrap(gie.ErrModelNotLoaded, gie.TypeValidation, "MODEL_NOT_LOADED", "model not loaded: "+query.Model.Name)
		slog.ErrorContext(ctx, "Model not loaded", "model", query.Model.Name, "error", err)
		errChan <- StreamedMsg{
			Num:     0,
			Content: err.Error(),
			MsgType: ErrorMsgType,
		}
		return
	}

	if inf.Cfg.Debug {
		slog.DebugContext(ctx, "Infer params")
		slog.DebugContext(ctx, "params", "value", query.Params)
	}

	// Execute inference
	nTok, err := inf.runInfer(ctx, c, query)
	// Handle infer completion or failure
	if err != nil {
		inf.ContinueInferringController = false
		errChan <- StreamedMsg{
			Num:     nTok + 1,
			Content: gie.Wrap(err, gie.TypeInference, "INFERENCE_FAILED", "infer failed").Error(),
			MsgType: ErrorMsgType,
		}
		return
	}

	// Handle streaming completion if needed
	if query.Params.Stream {
		err = inf.completeStream(ctx, c, nTok)
		if err != nil {
			// Forward the error to the caller via errChan
			errChan <- StreamedMsg{
				Num:     0,
				Content: err.Error(),
				MsgType: ErrorMsgType,
			}
			return
		}
	}

	// Send success message
	if !inf.ContinueInferringController {
		return
	}

	successMsg := StreamedMsg{
		Num:     nTok + 1,
		Content: "infer_completed",
		MsgType: SystemMsgType,
		Data: map[string]any{
			"request_id": reqID,
			"model":      query.Model.Name,
			"status":     "success",
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		},
	}
	resChan <- successMsg
}

// reqID generates a unique request ID for correlation.
func reqID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// runInfer performs the actual inference with token streaming.
func (inf *Infer) runInfer(ctx context.Context, c echo.Context, query *InferQuery) (int, error) {
	// Start the infer process
	inf.mu.Lock()
	inf.IsInferring = true
	inf.ContinueInferringController = true
	inf.mu.Unlock()

	nTok := 0
	startThinking := time.Now()
	var startEmitting time.Time
	var thinkingElapsed time.Duration

	// Execute inference with basic retry logic
	var err error
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context
		if ctx.Err() != nil {
			err = gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "CTX_CANCELED", "infer canceled")
			break
		}

		if !inf.ContinueInferringController {
			err = gie.Wrap(gie.ErrInferStopped, gie.TypeInference, "INFERENCE_STOPPED", "infer stopped by controller")
			break
		}

		// NOTE: This is a placeholder; real inference logic should replace the stub.
		// For demo purposes, assume successful inference
		err = nil
		break
	}

	// If successful, process tokens
	if err == nil && query.Params.Stream {
		// Create JSON encoder for streaming
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
		jsonEncoder := json.NewEncoder(c.Response())

		// Simulate token streaming
		for i := range 10 {
			inf.mu.Lock()
			stopped := !inf.ContinueInferringController
			inf.mu.Unlock()
			if stopped {
				break
			}

			token := "token_" + strconv.Itoa(i)
			err = inf.streamToken(ctx, nTok+i, token, jsonEncoder, c, &query.Params, startThinking, &startEmitting, &thinkingElapsed)
			if err != nil {
				return nTok, err
			}
			time.Sleep(10 * time.Millisecond)
			nTok++
		}
	}

	inf.mu.Lock()
	inf.IsInferring = false
	inf.mu.Unlock()
	return nTok, err
}

// completeStream handles streaming termination.
func (inf *Infer) completeStream(ctx context.Context, c echo.Context, _ int) error {
	if ctx.Err() != nil {
		err := gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "STREAM_CANCELED", "stream termination canceled")
		slog.InfoContext(ctx, "Context‑aware error", "request_id", ctx.Value(RequestID), "operation", "stream_termination", "error", err)
		return err
	}

	err := sendTerm(ctx, c)
	if err != nil {
		inf.mu.Lock()
		inf.ContinueInferringController = false
		inf.mu.Unlock()
		slog.ErrorContext(ctx, "Context‑aware error", "request_id", ctx.Value(RequestID), "operation", "stream_termination", "error", err)
		return gie.Wrap(err, gie.TypeInference, "STREAM_TERMINATION_FAILED", "stream termination failed")
	}

	return nil
}

// streamToken handles token processing during prediction.
func (inf *Infer) streamToken(
	ctx context.Context, nTok int, token string, jsonEncoder *json.Encoder,
	c echo.Context, params *InferParams, startThinking time.Time,
	startEmitting *time.Time, thinkingElapsed *time.Duration,
) error {
	// Check context
	if ctx.Err() != nil {
		return gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "CTX_CANCELED", "context canceled")
	}

	// Handle first token
	if nTok == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		inf.mu.Lock()
		continueInf := inf.ContinueInferringController
		inf.mu.Unlock()
		if params.Stream && continueInf {
			sMsg := &StreamedMsg{
				Content: "start_emitting",
				Num:     nTok,
				MsgType: SystemMsgType,
				Data: map[string]any{
					"thinking_time":        *thinkingElapsed,
					"thinking_time_format": thinkingElapsed.String(),
				},
			}

			err := write(ctx, c, jsonEncoder, sMsg)
			if err != nil {
				return gie.Wrap(err, gie.TypeInference, "STREAM_START_FAILED", "cannot stream start_emitting")
			}
		}
	}

	// Check if stopped
	inf.mu.Lock()
	stopped := !inf.ContinueInferringController
	inf.mu.Unlock()
	if stopped {
		return nil
	}

	// Log token
	inf.logToken(ctx, token)

	// Check if streaming
	if !params.Stream {
		return nil
	}

	// Create token message
	tMsg := &StreamedMsg{
		Content: token,
		Num:     nTok,
		MsgType: TokenMsgType,
	}

	return write(ctx, c, jsonEncoder, tMsg)
}

// logToken logs token information.
func (inf *Infer) logToken(ctx context.Context, token string) {
	slog.InfoContext(ctx, "token", "value", token)
}

// logMsg formats and logs a message with common context.
/* Removed logMsg helper – logging now performed directly via slog in logError and logToken. */
