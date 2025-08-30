// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/LM4eu/goinfer/ctx"
	"github.com/labstack/echo/v4"
)

// CreateErrorResponse creates a standardized error response with context.
func CreateErrorResponse(c echo.Context, err error, code, suggestion string) error {
	return c.JSON(http.StatusBadRequest, echo.Map{
		"error":      err.Error(),
		"code":       code,
		"request_id": ctx.GenerateRequestID(),
		"timestamp":  strconv.FormatInt(time.Now().Unix(), 10),
		"suggestion": suggestion,
	})
}

// CreateTimeoutResponse creates a standardized timeout response.
func CreateTimeoutResponse(c echo.Context, suggestion string) error {
	timeoutErr := errors.New("request timeout after 30 seconds")
	return c.JSON(http.StatusRequestTimeout, echo.Map{
		"error":      timeoutErr.Error(),
		"code":       "REQUEST_TIMEOUT",
		"request_id": ctx.GenerateRequestID(),
		"timestamp":  strconv.FormatInt(time.Now().Unix(), 10),
		"suggestion": suggestion,
	})
}

// CreateCancelResponse creates a standardized cancellation response.
func CreateCancelResponse(c echo.Context) error {
	return c.JSON(http.StatusNoContent, echo.Map{
		"message":    "Request canceled",
		"code":       "REQUEST_CANCELED",
		"request_id": ctx.GenerateRequestID(),
		"timestamp":  strconv.FormatInt(time.Now().Unix(), 10),
	})
}
