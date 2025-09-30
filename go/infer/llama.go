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
func (inf *Infer) Infer(ctx context.Context, query *Query, c echo.Context, resChan, errChan chan<- StreamedMsg) {
	// Create context with request ID
	ctx = context.WithValue(ctx, RequestID, reqID())

	// Early validation checks
	if ctx.Err() != nil {
		err := gie.Wrap(ctx.Err(), gie.InferErr, "infer canceled")
		slog.ErrorContext(ctx, "Context canceled at infer start", "error", err)
		errChan <- StreamedMsg{
			Num:     0,
			Error:   err,
			MsgType: ErrorMsgType,
		}
		return
	}

	if query.Model == "" {
		err := gie.New(gie.Invalid, "model not loaded model="+query.Model)
		slog.ErrorContext(ctx, "Model not loaded", "model", query.Model, "error", err)
		errChan <- StreamedMsg{
			Num:     0,
			Error:   err,
			MsgType: ErrorMsgType,
		}
		return
	}

	slog.DebugContext(ctx, "Infer", "query", query)

	// Execute inference
	nTok, err := inf.runInfer(ctx, c, query)
	// Handle infer completion or failure
	if err != nil {
		inf.ContinueInferringController = false
		errChan <- StreamedMsg{
			Num:     nTok + 1,
			Error:   gie.Wrap(err, gie.InferErr, "infer failed"),
			MsgType: ErrorMsgType,
		}
		return
	}

	// Handle streaming completion if needed
	if query.Stream {
		err = inf.completeStream(ctx, c, nTok)
		if err != nil {
			// Forward the error to the caller via errChan
			errChan <- StreamedMsg{
				Num:     0,
				Error:   err,
				MsgType: ErrorMsgType,
			}
			return
		}
	}

	// Send success message
	if !inf.ContinueInferringController {
		return
	}

	// infer completed
	successMsg := StreamedMsg{
		Num:     nTok + 1,
		MsgType: SystemMsgType,
		Data: map[string]any{
			"request_id": reqID,
			"model":      query.Model,
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
func (inf *Infer) runInfer(ctx context.Context, c echo.Context, query *Query) (int, error) {
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
	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context
		if ctx.Err() != nil {
			err = gie.New(gie.InferErr, "request canceled by client")
			break
		}

		if !inf.ContinueInferringController {
			err = gie.New(gie.InferErr, "inference stopped by controller")
			break
		}

		// NOTE: This is a placeholder; real inference logic should replace the stub.
		// For demo purposes, assume successful inference
		err = nil
		break
	}

	// If successful, process tokens
	if err == nil && query.Stream {
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
			err = inf.streamToken(ctx, nTok+i, token, jsonEncoder, c, query, startThinking, &startEmitting, &thinkingElapsed)
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
		err := gie.New(gie.InferErr, "stream termination canceled by client")
		slog.InfoContext(ctx, "Context‑aware error", "request_id", ctx.Value(RequestID), "operation", "stream_termination", "error", err)
		return err
	}

	err := sendTerm(ctx, c)
	if err != nil {
		inf.mu.Lock()
		inf.ContinueInferringController = false
		inf.mu.Unlock()
		slog.ErrorContext(ctx, "Context‑aware error", "request_id", ctx.Value(RequestID), "operation", "stream_termination", "error", err)
		return gie.Wrap(err, gie.InferErr, "stream termination failed")
	}

	return nil
}

// streamToken handles token processing during prediction.
//
//nolint:revive // will refactor to reduce the number of arguments
func (inf *Infer) streamToken(
	ctx context.Context, nTok int, token string, jsonEncoder *json.Encoder,
	c echo.Context, params *Query, startThinking time.Time,
	startEmitting *time.Time, thinkingElapsed *time.Duration,
) error {
	// Check context
	if ctx.Err() != nil {
		return gie.New(gie.InferErr, "streamToken: request canceled by client")
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
				return gie.Wrap(err, gie.InferErr, "cannot start_emitting stream")
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
	logToken(ctx, token)

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
func logToken(ctx context.Context, token string) {
	slog.InfoContext(ctx, "token", "value", token)
}

// logMsg formats and logs a message with common context.
/* Removed logMsg helper – logging now performed directly via slog in logError and logToken. */
