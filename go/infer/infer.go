// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package infer implements a lightweight LLM proxy with streaming support.
package infer

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// inferHandler handles infer requests.
func (inf *Infer) inferHandler(c echo.Context) error {
	// Initialize context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Check if infer is already running using Infer
	inf.mu.Lock()
	if inf.IsInferring {
		slog.Info("Infer already running")
		inf.mu.Unlock()
		return c.NoContent(http.StatusAccepted)
	}
	inf.mu.Unlock()

	// Bind request parameters
	reqMap := echo.Map{}
	err := c.Bind(&reqMap)
	if err != nil {
		return gie.HandleValidationError(c, gie.ErrInvalidFormat)
	}

	// Parse infer parameters directly
	query, err := parseInferQuery(ctx, reqMap)
	if err != nil {
		return gie.HandleValidationError(c, gie.ErrInvalidParams)
	}

	// Setup streaming response if needed
	if query.Params.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}

	// Execute infer directly (no retry)
	result, err := inf.execute(c, query)
	if err != nil {
		// Use the generic inference error handler to avoid exposing internal error messages.
		return gie.HandleInferenceError(c, err)
	}

	// Handle the infer result
	slog.Debug("-----------------------------")
	for key, value := range result.Data {
		slog.Debug("result", "key", key, "value", value)
	}
	slog.Debug("-----------------------------")

	if !query.Params.Stream {
		return c.JSON(http.StatusOK, result.Data)
	}
	return nil
}

// parseInferQuery parses infer parameters from echo.Map directly.
func parseInferQuery(ctx context.Context, m echo.Map) (*InferQuery, error) {
	query := &InferQuery{
		Prompt: "",
		Model:  DefaultModel,
		Params: DefaultInferParams,
	}

	// Parse required prompt parameter
	val, ok := m["prompt"].(string)
	if !ok {
		return query, gie.ErrInvalidPrompt
	}
	query.Prompt = val

	if val, ok := m["model"].(string); ok {
		query.Model.Name = val
	}

	query.Model.Ctx = getInt(ctx, m, "ctx")

	if val, ok := m["stream"].(bool); ok {
		query.Params.Stream = val
	}

	query.Params.Sampling.Temperature = getFloat(ctx, m, "temperature")
	query.Params.Sampling.MinP = getFloat(ctx, m, "min_p")
	query.Params.Sampling.TopP = getFloat(ctx, m, "top_p")
	query.Params.Sampling.PresencePenalty = getFloat(ctx, m, "presence_penalty")
	query.Params.Sampling.FrequencyPenalty = getFloat(ctx, m, "frequency_penalty")
	query.Params.Sampling.RepeatPenalty = getFloat(ctx, m, "repeat_penalty")
	query.Params.Sampling.TailFreeSamplingZ = getFloat(ctx, m, "tfs")
	query.Params.Sampling.TopK = getInt(ctx, m, "top_k")
	query.Params.Generation.MaxTokens = getInt(ctx, m, "max_tokens")

	// Parse stop prompts array
	err := populateStopPrompts(m, &query.Params.Generation)
	if err != nil {
		return query, err
	}

	// Parse media arrays
	if images, ok := m["images"].([]string); ok {
		query.Params.Media.Images = images
	}
	if audios, ok := m["audios"].([]string); ok {
		query.Params.Media.Audios = audios
	}

	return query, nil
}

func getInt(ctx context.Context, m echo.Map, param string) int {
	v, ok := m[param]
	if ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		default:
			slog.WarnContext(ctx, "expected int (or float64) but received", "param", param, "value", m[param])
		}
	}
	return 0
}

func getFloat(ctx context.Context, m echo.Map, param string) float32 {
	v, ok := m[param]
	if ok {
		switch val := v.(type) {
		case int:
			return float32(val)
		case float64:
			return float32(val)
		default:
			slog.WarnContext(ctx, "expected float64 (or int) but received", "param", param, "value", m[param])
		}
	}
	return 0
}

// execute inference.
func (inf *Infer) execute(c echo.Context, query *InferQuery) (*StreamedMsg, error) {
	// Execute infer through Infer
	resChan := make(chan StreamedMsg)
	errChan := make(chan StreamedMsg)
	defer close(resChan)
	defer close(errChan)

	err := inf.forwardInference(c.Request().Context(), query, c, resChan, errChan)
	if err != nil {
		return nil, gie.Wrap(err, gie.TypeInference, "PROXY_FORWARD_FAILED", "proxy manager forward inference failed")
	}

	// Process response
	select {
	case resCh, ok := <-resChan:
		if ok {
			return &resCh, nil
		}
		return nil, gie.ErrChanClosed

	case errCh, ok := <-errChan:
		if ok {
			if errCh.MsgType == ErrorMsgType {
				return nil, gie.Wrap(gie.ErrInferFailed, gie.TypeInference, "INFERENCE_ERROR", "infer error: "+errCh.Content)
			}
			return nil, gie.Wrap(gie.ErrInferFailed, gie.TypeInference, "INFERENCE_ERROR", errCh.Content)
		}
		return nil, gie.ErrChanClosed

	case <-c.Request().Context().Done():
		// Client canceled request
		inf.mu.Lock()
		inf.ContinueInferringController = false
		inf.mu.Unlock()
		return nil, gie.ErrClientCanceled
	}
}

// abortHandler aborts ongoing inference.
func (inf *Infer) abortHandler(c echo.Context) error {
	err := inf.abortInference()
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "abortInference", "error", err)
		return c.NoContent(http.StatusAccepted)
	}

	slog.DebugContext(c.Request().Context(), "Aborting inference")

	return c.NoContent(http.StatusNoContent)
}

// populateStopPrompts extracts and validates the "stop" parameter from the request map.
func populateStopPrompts(m echo.Map, gen *Generation) error {
	v, ok := m["stop"]
	if !ok {
		return nil
	}
	slice, ok := v.([]any)
	if !ok {
		return gie.Wrap(gie.ErrInvalidParams, gie.TypeValidation, "STOP_INVALID_TYPE", "stop must be an array")
	}
	if len(slice) > 10 {
		return gie.Wrap(gie.ErrInvalidParams, gie.TypeValidation, "STOP_TOO_LARGE", "stop array too large (max 10)")
	}
	if len(slice) == 0 {
		return nil
	}
	gen.StopPrompts = make([]string, len(slice))
	for i, val := range slice {
		str, ok := val.(string)
		if !ok {
			return gie.Wrap(gie.ErrInvalidParams, gie.TypeValidation, "STOP_INVALID_ELEMENT", "stop["+strconv.Itoa(i)+"] must be a string")
		}
		gen.StopPrompts[i] = str
	}
	return nil
}
