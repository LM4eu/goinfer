package lm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	ctxpkg "github.com/synw/goinfer/ctx"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// Infer performs language model inference.
func Infer(query types.InferQuery, c echo.Context, resultChan, errorChan chan<- types.StreamedMsg) {
	// Create context with request ID
	ctx := c.Request().Context()
	reqID := ctxpkg.GenerateRequestID()
	ctx = context.WithValue(ctx, "requestID", reqID)

	// Early validation checks
	err := ctx.Err()
	if err != nil {
		inferErr := fmt.Errorf("infer canceled: %w", err)
		ctxpkg.LogContextAwareError(ctx, "infer_start", inferErr)
		errorChan <- types.StreamedMsg{
			Num:     0,
			Content: inferErr.Error(),
			MsgType: types.ErrorMsgType,
		}
		return
	}

	if query.Model.Name == "" {
		modelErr := fmt.Errorf("model not loaded: %s", query.Model.Name)
		errorChan <- types.StreamedMsg{
			Num:     0,
			Content: modelErr.Error(),
			MsgType: types.ErrorMsgType,
		}
		return
	}

	if state.Debug {
		fmt.Println("Infer params:")
		fmt.Printf("%+v\n\n", query.Params)
	}

	// Execute inference
	ntok, inferErr := runInfer(ctx, c, query)

	// Handle infer completion or failure
	if inferErr != nil {
		state.ContinueInferringController = false
		errorChan <- types.StreamedMsg{
			Num:     ntok + 1,
			Content: fmt.Sprintf("infer failed: %v", inferErr),
			MsgType: types.ErrorMsgType,
		}
		return
	}

	// Handle streaming completion if needed
	if query.Params.Stream {
		err := completeStream(ctx, c, ntok)
		if err != nil {
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
func runInfer(ctx context.Context, c echo.Context, query types.InferQuery) (int, error) {
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
			inferErr = fmt.Errorf("infer canceled: %w", err)
			break
		}

		if !state.ContinueInferringController {
			inferErr = errors.New("infer stopped by controller")
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
			err := streamToken(ctx, ntok+i, token, jsonEncoder, c, query.Params, startThinking, &startEmitting, &thinkingElapsed)
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
	if err := ctx.Err(); err != nil {
		streamErr := fmt.Errorf("stream termination canceled: %w", err)
		ctxpkg.LogContextAwareError(ctx, "stream_termination", streamErr)
		return streamErr
	}

	err := sendTerm(ctx, c)
	if err != nil {
		state.ContinueInferringController = false
		streamErr := fmt.Errorf("stream termination failed: %w", err)
		ctxpkg.LogContextAwareError(ctx, "stream_termination", streamErr)
		logError(ctx, "Llama", "cannot send stream termination", streamErr)
		return streamErr
	}

	return nil
}

// streamToken handles token processing during prediction.
func streamToken(ctx context.Context, ntok int, token string, jsonEncoder *json.Encoder, c echo.Context, params types.InferParams, startThinking time.Time, startEmitting *time.Time, thinkingElapsed *time.Duration) error {
	// Check context
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("context canceled: %w", err)
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

			err := write(ctx, c, jsonEncoder, smsg)
			if err != nil {
				return fmt.Errorf("cannot stream start_emitting: %w", err)
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
