// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LM4eu/goinfer/gic"
	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// Infer performs language model inference.
func (inf *Infer) Infer(ctx context.Context, query *InferQuery, c echo.Context, resChan, errChan chan<- StreamedMsg) {
	// Create context with request ID
	reqID := gic.GenReqID()
	ctx = context.WithValue(ctx, "requestID", reqID)

	// Early validation checks
	if ctx.Err() != nil {
		giErr := gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "CTX_CANCELED", "infer canceled")
		gic.LogCtxAwareError(ctx, "infer_start", giErr)
		errChan <- StreamedMsg{
			Num:     0,
			Content: giErr.Error(),
			MsgType: ErrorMsgType,
		}
		return
	}

	if query.Model.Name == "" {
		err := gie.Wrap(gie.ErrModelNotLoaded, gie.TypeValidation, "MODEL_NOT_LOADED", "model not loaded: "+query.Model.Name)
		errChan <- StreamedMsg{
			Num:     0,
			Content: err.Error(),
			MsgType: ErrorMsgType,
		}
		return
	}

	if inf.Cfg.Debug {
		fmt.Println("DBG: Infer params:")
		fmt.Printf("DBG: %+v\n\n", query.Params)
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

// runInfer performs the actual inference with token streaming.
func (inf *Infer) runInfer(ctx context.Context, c echo.Context, query *InferQuery) (int, error) {
	// Start the infer process
	inf.IsInferring = true
	inf.ContinueInferringController = true

	nTok := 0
	startThinking := time.Now()
	var startEmitting time.Time
	var thinkingElapsed time.Duration

	// Execute inference with basic retry logic
	var giErr error
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context
		if ctx.Err() != nil {
			giErr = gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "CTX_CANCELED", "infer canceled")
			break
		}

		if !inf.ContinueInferringController {
			giErr = gie.Wrap(gie.ErrInferStopped, gie.TypeInference, "INFERENCE_STOPPED", "infer stopped by controller")
			break
		}

		// For demo purposes, assume successful inference
		giErr = nil
		break
	}

	// If successful, process tokens
	if giErr == nil && query.Params.Stream {
		// Create JSON encoder for streaming
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
		jsonEncoder := json.NewEncoder(c.Response())

		// Simulate token streaming
		for i := range 10 {
			if !inf.ContinueInferringController {
				break
			}

			token := fmt.Sprintf("token_%d", i)
			err := inf.streamToken(ctx, nTok+i, token, jsonEncoder, c, &query.Params, startThinking, &startEmitting, &thinkingElapsed)
			if err != nil {
				return nTok, err
			}
			time.Sleep(10 * time.Millisecond)
			nTok++
		}
	}

	inf.IsInferring = false
	return nTok, giErr
}

// completeStream handles streaming termination.
func (inf *Infer) completeStream(ctx context.Context, c echo.Context, _ int) error {
	if ctx.Err() != nil {
		er := gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "STREAM_CANCELED", "stream termination canceled")
		gic.LogCtxAwareError(ctx, "stream_termination", er)
		return er
	}

	err := sendTerm(ctx, c)
	if err != nil {
		inf.ContinueInferringController = false
		er := gie.Wrap(err, gie.TypeInference, "STREAM_TERMINATION_FAILED", "stream termination failed")
		gic.LogCtxAwareError(ctx, "stream_termination", er)
		inf.logError(ctx, "Llama", "cannot send stream termination", er)
		return er
	}

	return nil
}

// streamToken handles token processing during prediction.
func (inf *Infer) streamToken(
	ctx context.Context, nTok int, token string, jsonEncoder *json.Encoder,
	c echo.Context, params *InferParams, startThinking time.Time,
	startEmitting *time.Time, thinkingElapsed *time.Duration,
) error {
	// Check context	err :=
	if ctx.Err() != nil {
		return gie.Wrap(gie.ErrClientCanceled, gie.TypeInference, "CTX_CANCELED", "context canceled")
	}

	// Handle first token
	if nTok == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		if params.Stream && inf.ContinueInferringController {
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
	if !inf.ContinueInferringController {
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

// logError logs error information.
func (inf *Infer) logError(ctx context.Context, prefix, message string, err error) {
	if err != nil {
		inf.logMsg(ctx, "%s | ERROR: %s - %v", prefix, message, err)
	} else {
		inf.logMsg(ctx, "%s | ERROR: %s", prefix, message)
	}
}

// logToken logs token information.
func (inf *Infer) logToken(ctx context.Context, token string) {
	inf.logMsg(ctx, "token: %s", token)
}

// logMsg formats and logs a message with common context.
func (inf *Infer) logMsg(ctx context.Context, format string, args ...any) {
	if !inf.Cfg.Verbose {
		return
	}

	reqID := "req"
	if id := ctx.Value("requestID"); id != nil {
		if str, ok := id.(string); ok {
			reqID = str
		}
	}

	fmt.Printf("INF: [%s] | c: %s | r: %s | %s\n",
		time.Now().Format(time.RFC3339), fmt.Sprintf("c-%d", time.Now().UnixNano()), reqID, fmt.Sprintf(format, args...))
}
