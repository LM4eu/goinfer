// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/LM4eu/goinfer/errors"
	"github.com/LM4eu/goinfer/state"
	"github.com/LM4eu/goinfer/types"
	"github.com/labstack/echo/v4"
)

// inferHandler handles infer requests.
func inferHandler(c echo.Context) error {
	// Initialize context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Check if infer is already running using ProxyManager
	if proxyManager.IsInferring() {
		fmt.Println("Infer already running")
		return c.NoContent(http.StatusAccepted)
	}

	// Bind request parameters
	reqMap := echo.Map{}
	err := c.Bind(&reqMap)
	if err != nil {
		return errors.HandleValidationError(c, errors.ErrInvalidFormat)
	}

	// Parse infer parameters directly
	query, err := parseInferQuery(reqMap)
	if err != nil {
		return errors.HandleValidationError(c, errors.ErrInvalidParams)
	}

	// Setup streaming response if needed
	if query.Params.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}

	// Execute infer directly (no retry)
	result, err := execute(c, ctx, query)
	if err != nil {
		return err
	}

	// Handle the infer result
	if state.Verbose {
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
func parseInferQuery(m echo.Map) (*types.InferQuery, error) {
	req := &types.InferQuery{
		Prompt: "",
		Model:  types.DefaultModel,
		Params: types.DefaultInferParams,
	}

	// Check required prompt parameter
	if _, ok := m["prompt"]; !ok {
		return req, errors.ErrPromptRequired
	}

	// Parse simple parameters directly
	if val, ok := m["prompt"].(string); ok {
		req.Prompt = val
	} else {
		return req, errors.ErrInvalidPrompt
	}

	if val, ok := m["model"].(string); ok {
		req.Model.Name = val
	}

	if val, ok := m["ctx"].(int); ok {
		req.Model.Ctx = val
	}

	if val, ok := m["stream"].(bool); ok {
		req.Params.Stream = val
	}

	if val, ok := m["temperature"].(float64); ok {
		req.Params.Sampling.Temperature = float32(val)
	}

	if val, ok := m["min_p"].(float64); ok {
		req.Params.Sampling.MinP = float32(val)
	}

	if val, ok := m["top_p"].(float64); ok {
		req.Params.Sampling.TopP = float32(val)
	}

	if val, ok := m["presence_penalty"].(float64); ok {
		req.Params.Sampling.PresencePenalty = float32(val)
	}

	if val, ok := m["frequency_penalty"].(float64); ok {
		req.Params.Sampling.FrequencyPenalty = float32(val)
	}

	if val, ok := m["repeat_penalty"].(float64); ok {
		req.Params.Sampling.RepeatPenalty = float32(val)
	}

	if val, ok := m["tfs"].(float64); ok {
		req.Params.Sampling.TailFreeSamplingZ = float32(val)
	}

	if val, ok := m["top_k"].(int); ok {
		req.Params.Sampling.TopK = val
	}

	if val, ok := m["max_tokens"].(int); ok {
		req.Params.Generation.MaxTokens = val
	}

	// Parse stop prompts array
	err := populateStopPrompts(m, &req.Params.Generation)
	if err != nil {
		return req, err
	}

	// Parse media byte arrays
	if v, ok := m["images"]; ok {
		if sliceImg, okImg := v.([]any); okImg && len(sliceImg) > 0 {
			req.Params.Media.Images = make([]byte, len(sliceImg))
			for i, val := range sliceImg {
				imgByte, okImg := val.(byte)
				if !okImg {
					return req, fmt.Errorf("images[%d] must be a byte", i)
				}
				req.Params.Media.Images[i] = imgByte
			}
		}
	}

	if v, ok := m["audios"]; ok {
		if sliceAud, okAud := v.([]any); okAud && len(sliceAud) > 0 {
			req.Params.Media.Audios = make([]byte, len(sliceAud))
			for i, val := range sliceAud {
				audioByte, okAud := val.(byte)
				if !okAud {
					return req, fmt.Errorf("audios[%d] must be a byte", i)
				}
				req.Params.Media.Audios[i] = audioByte
			}
		}
	}

	return req, nil
}

// execute executes inference using ProxyManager.
func execute(c echo.Context, ctx context.Context, query *types.InferQuery) (*types.StreamedMsg, error) {
	// Execute infer through ProxyManager
	resultChan := make(chan types.StreamedMsg)
	errorChan := make(chan types.StreamedMsg)
	defer close(resultChan)
	defer close(errorChan)

	err := proxyManager.ForwardInference(ctx, query, c, resultChan, errorChan)
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInference, "PROXY_FORWARD_FAILED", "proxy manager forward inference failed")
	}

	// Process response from ProxyManager
	select {
	case response, ok := <-resultChan:
		if ok {
			return &response, nil
		}
		return nil, errors.ErrChannelClosed

	case message, ok := <-errorChan:
		if ok {
			if message.MsgType == types.ErrorMsgType {
				return nil, errors.Wrap(errors.ErrInferenceFailed, errors.TypeInference, "INFERENCE_ERROR", "infer error: "+message.Content)
			}
			return nil, errors.Wrap(errors.ErrInferenceFailed, errors.TypeInference, "INFERENCE_ERROR", fmt.Sprintf("infer error: %v", message))
		}
		return nil, errors.ErrChannelClosed

	case <-ctx.Done():
		// Client canceled request
		state.ContinueInferringController = false
		return nil, errors.ErrClientCanceled
	}
}

// abortHandler aborts ongoing inference using ProxyManager.
func abortHandler(c echo.Context) error {
	err := proxyManager.AbortInference()
	if err != nil {
		fmt.Printf("INF: %v\n", err)
		return c.NoContent(http.StatusAccepted)
	}

	if state.Verbose {
		fmt.Println("INF: Aborting inference")
	}

	return c.NoContent(http.StatusNoContent)
}

// populateStopPrompts extracts and validates the "stop" parameter from the request map.
func populateStopPrompts(m echo.Map, gen *types.Generation) error {
	v, ok := m["stop"]
	if !ok {
		return nil
	}
	slice, ok := v.([]any)
	if !ok {
		return errors.Wrap(errors.ErrInvalidParams, errors.TypeValidation, "STOP_INVALID_TYPE", "stop must be an array")
	}
	if len(slice) > 10 {
		return errors.Wrap(errors.ErrInvalidParams, errors.TypeValidation, "STOP_TOO_LARGE", "stop array too large (max 10)")
	}
	if len(slice) == 0 {
		return nil
	}
	gen.StopPrompts = make([]string, len(slice))
	for i, val := range slice {
		str, ok := val.(string)
		if !ok {
			return errors.Wrap(errors.ErrInvalidParams, errors.TypeValidation, "STOP_INVALID_ELEMENT", fmt.Sprintf("stop[%d] must be a string", i))
		}
		gen.StopPrompts[i] = str
	}
	return nil
}
