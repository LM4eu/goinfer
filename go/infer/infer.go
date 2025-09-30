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

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// inferHandler handles infer requests.
func (inf *Infer) inferHandler(c echo.Context) error {
	// Parse infer parameters directly
	query, err := parseInferQuery(c)
	if err != nil {
		return gie.HandleValidationError(c, gie.ErrInvalidParams)
	}

	if query == nil {
		return gie.New(gie.InferErr, "invalid infer query")
	}

	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager configured")
	}

	ginCtx := echo2gin(c)
	inf.ProxyMan.ProxyOAIHandler(ginCtx)
	return nil
}

// parseInferQuery parses infer parameters from echo.Map directly.
func parseInferQuery(c echo.Context) (*InferQuery, error) {
	// Bind request parameters
	m := echo.Map{}
	err := c.Bind(&m)
	if err != nil {
		return nil, gie.HandleValidationError(c, gie.ErrInvalidFormat)
	}

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

	ctx := c.Request().Context()

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
	err = populateStopPrompts(m, &query.Params.Generation)
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
		return gie.New(gie.Invalid, "invalid parameter type: stop must be an array")
	}
	if len(slice) > 10 {
		return gie.New(gie.Invalid, "invalid size: stop array too large (max 10) got="+strconv.Itoa(len(slice)))
	}
	if len(slice) == 0 {
		return nil
	}
	gen.StopPrompts = make([]string, len(slice))
	for i, val := range slice {
		str, ok := val.(string)
		if !ok {
			return gie.New(gie.Invalid, "invalid parameter type: stop["+strconv.Itoa(i)+"] must be a string")
		}
		gen.StopPrompts[i] = str
	}
	return nil
}
