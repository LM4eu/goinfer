// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package gie

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorHandler is a centralized error handler for Echo middleware.
func ErrorHandler(err error, c echo.Context) error {
	// Log the error for debugging
	c.Logger().Error(err)

	// Return a standardized error response
	return errorToEchoResponse(c, err)
}

// HandleValidationError handles validation errors with proper HTTP status.
func HandleValidationError(c echo.Context, err error) error {
	return handleError(c, err, TypeValidation, "VALIDATION_ERROR", "validation failed")
}

// HandleInferenceError handles inference-related errors.
func HandleInferenceError(c echo.Context, err error) error {
	return handleError(c, err, TypeInference, "INFERENCE_ERROR", "inference failed")
}

// HandleConfigError handles configuration-related errors.
func HandleConfigError(c echo.Context, err error) error {
	return handleError(c, err, TypeConfiguration, "CONFIG_ERROR", "configuration error")
}

// HandleUnauthorizedError handles authentication errors.
func HandleUnauthorizedError(c echo.Context, err error) error {
	return handleError(c, err, TypeUnauthorized, "UNAUTHORIZED", "unauthorized access")
}

// HandleNotFoundError handles resource not found errors.
func HandleNotFoundError(c echo.Context, err error) error {
	return handleError(c, err, TypeNotFound, "NOT_FOUND", "resource not found")
}

// HandleTimeoutError handles timeout-related errors.
func HandleTimeoutError(c echo.Context, err error) error {
	return handleError(c, err, TypeTimeout, "TIMEOUT_ERROR", "request timeout")
}

// HandleServerError handles server-related errors.
func HandleServerError(c echo.Context, err error) error {
	return handleError(c, err, TypeServer, "SERVER_ERROR", "internal server error")
}

// handleError centralizes error handling for HTTP responses.
func handleError(c echo.Context, err error, expectedType ErrorType, wrapCode, wrapMsg string) error {
	var giErr *GoInferError
	if errors.As(err, &giErr) && giErr.Type == expectedType {
		return c.JSON(statusCode(giErr.Type), giErr)
	}
	// Not the expected type: wrap and forward
	wrapped := Wrap(err, expectedType, wrapCode, wrapMsg)
	return errorToEchoResponse(c, wrapped)
}

// errorToEchoResponse converts an GoInferError to an Echo error response.
func errorToEchoResponse(c echo.Context, err error) error {
	var giErr *GoInferError
	if !errors.As(err, &giErr) {
		// If not an GoInferError, wrap it and return internal server error
		giErr = Wrap(err, TypeServer, "INTERNAL_ERROR", "internal server error")
	}
	return c.JSON(statusCode(giErr.Type), giErr)
}

// statusCode deduce the HTTP status code from an ErrorType.
func statusCode(errType ErrorType) int {
	switch errType {
	case TypeValidation:
		return http.StatusBadRequest
	case TypeUnauthorized:
		return http.StatusUnauthorized
	case TypeNotFound:
		return http.StatusNotFound
	case TypeTimeout:
		return http.StatusRequestTimeout
	case TypeConfiguration, TypeInference, TypeServer:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
