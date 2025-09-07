// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT
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
	if inf.IsInferring {
		fmt.Println("Infer already running")
		return c.NoContent(http.StatusAccepted)
	}

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
		return err
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
	req := &InferQuery{
		Prompt: "",
		Model:  DefaultModel,
		Params: DefaultInferParams,
	}

	// Check required prompt parameter
	if _, ok := m["prompt"]; !ok {
		return req, gie.ErrPromptRequired
	}

	// Parse simple parameters directly
	if val, ok := m["prompt"].(string); ok {
		req.Prompt = val
	} else {
		return req, gie.ErrInvalidPrompt
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
		inf.ContinueInferringController = false
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
