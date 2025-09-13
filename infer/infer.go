// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package infer implements a lightweight LLM proxy with streaming support.
package infer

import (
	"context"
	"fmt"
	"net/http"
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
		fmt.Println("Infer already running")
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
	query, err := parseInferQuery(reqMap)
	if err != nil {
		return gie.HandleValidationError(c, gie.ErrInvalidParams)
	}

	// Setup streaming response if needed
	if query.Params.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}

	// Execute infer directly (no retry)
	result, err := inf.execute(c, ctx, query)
	if err != nil {
		// Use the generic inference error handler to avoid exposing internal error messages.
		return gie.HandleInferenceError(c, err)
	}

	// Handle the infer result
	if inf.Cfg.Verbose {
		fmt.Println("INF: -------- result ----------")
		for key, value := range result.Data {
			fmt.Printf("INF: %s: %v\n", key, value)
		}
		fmt.Println("INF: --------------------------")
	}

	if !query.Params.Stream {
		return c.JSON(http.StatusOK, result.Data)
	}
	return nil
}

// parseInferQuery parses infer parameters from echo.Map directly.
func parseInferQuery(m echo.Map) (*InferQuery, error) {
	query := &InferQuery{
		Prompt: "",
		Model:  DefaultModel,
		Params: DefaultInferParams,
	}

	// Parse required prompt parameter
	if val, ok := m["prompt"].(string); ok {
		query.Prompt = val
	} else {
		return query, gie.ErrInvalidPrompt
	}

	if val, ok := m["model"].(string); ok {
		query.Model.Name = val
	}

	query.Model.Ctx = getInt(m, "ctx")

	if val, ok := m["stream"].(bool); ok {
		query.Params.Stream = val
	}

	query.Params.Sampling.Temperature = getFloat(m, "temperature")
	query.Params.Sampling.MinP = getFloat(m, "min_p")
	query.Params.Sampling.TopP = getFloat(m, "top_p")
	query.Params.Sampling.PresencePenalty = getFloat(m, "presence_penalty")
	query.Params.Sampling.FrequencyPenalty = getFloat(m, "frequency_penalty")
	query.Params.Sampling.RepeatPenalty = getFloat(m, "repeat_penalty")
	query.Params.Sampling.TailFreeSamplingZ = getFloat(m, "tfs")
	query.Params.Sampling.TopK = getInt(m, "top_k")
	query.Params.Generation.MaxTokens = getInt(m, "max_tokens")

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

func getInt(m echo.Map, param string) int {
	if v, ok := m[param]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		default:
			fmt.Printf("Expected int (or float64) but received %s=%v", param, m[param])
		}
	}
	return 0
}

func getFloat(m echo.Map, param string) float32 {
	if v, ok := m[param]; ok {
		switch val := v.(type) {
		case int:
			return float32(val)
		case float64:
			return float32(val)
		default:
			fmt.Printf("Expected float64 (or int) but received %s=%v", param, m[param])
		}
	}
	return 0
}

// execute inference.
func (inf *Infer) execute(c echo.Context, ctx context.Context, query *InferQuery) (*StreamedMsg, error) {
	// Execute infer through Infer
	resChan := make(chan StreamedMsg)
	errChan := make(chan StreamedMsg)
	defer close(resChan)
	defer close(errChan)

	err := inf.forwardInference(ctx, query, c, resChan, errChan)
	if err != nil {
		return nil, gie.Wrap(err, gie.TypeInference, "PROXY_FORWARD_FAILED", "proxy manager forward inference failed")
	}

	// Process response
	select {
	case res, ok := <-resChan:
		if ok {
			return &res, nil
		}
		return nil, gie.ErrChanClosed

	case err, ok := <-errChan:
		if ok {
			if err.MsgType == ErrorMsgType {
				return nil, gie.Wrap(gie.ErrInferFailed, gie.TypeInference, "INFERENCE_ERROR", "infer error: "+err.Content)
			}
			return nil, gie.Wrap(gie.ErrInferFailed, gie.TypeInference, "INFERENCE_ERROR", fmt.Sprintf("infer error: %v", err))
		}
		return nil, gie.ErrChanClosed

	case <-ctx.Done():
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
		fmt.Printf("INF: %v\n", err)
		return c.NoContent(http.StatusAccepted)
	}

	if inf.Cfg.Verbose {
		fmt.Println("INF: Aborting inference")
	}

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
			return gie.Wrap(gie.ErrInvalidParams, gie.TypeValidation, "STOP_INVALID_ELEMENT", fmt.Sprintf("stop[%d] must be a string", i))
		}
		gen.StopPrompts[i] = str
	}
	return nil
}
