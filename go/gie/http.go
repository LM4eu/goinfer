// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package gie

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// HandleErrorMiddleware is a centralized error handler for Echo middleware.
func HandleErrorMiddleware(err error, c echo.Context) error {
	// Log the error for debugging
	c.Logger().Error(err)

	// Return a standardized error response
	var giErr *Error
	if !errors.As(err, &giErr) {
		// If not an GoinferError, wrap it and return internal server error
		giErr = Wrap(err, ServerErr, "internal server error")
	}
	return c.JSON(statusCode(giErr.Code), giErr)
}

// statusCode deduce the HTTP status code from an ErrorType.
func statusCode(errType Code) int {
	switch errType {
	case Invalid:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	case Timeout:
		return http.StatusRequestTimeout
	case UserAbort:
		return http.StatusNoContent
	case ConfigErr, InferErr, ServerErr:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}
