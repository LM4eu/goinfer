// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package infer implements a lightweight LLM proxy with streaming support.
package infer

import (
	"context"
	"encoding/json"
	"io"

	"log/slog"
	"net/http"

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
func parseInferQuery(c echo.Context) (*Query, error) {
	query := defaultQuery

	err := json.NewDecoder(c.Request().Body).Decode(&query)
	if err == io.EOF {
		return nil, gie.New(gie.NotFound, "the infer request is empty")
	} else if err != nil {
		return nil, gie.Wrap(err, gie.Invalid, "expect JSON format (see documentation)")
	}

	// Parse required prompt parameter
	if query.Prompt == "" {
		return nil, gie.ErrInvalidPrompt
	}

	return &query, nil
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
