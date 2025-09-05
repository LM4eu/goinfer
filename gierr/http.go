// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package gierr

import (
	"errors"

	"github.com/labstack/echo/v4"
)

// HTTPResponse represents a standardized HTTP error response.
type HTTPResponse struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ErrorToEchoResponse converts an AppError to an Echo error response.
func ErrorToEchoResponse(c echo.Context, err error) error {
	var appErr *GoInferError
	if errors.As(err, &appErr) {
		httpErr := ErrorToHTTP(appErr)
		response := HTTPResponse{
			Type:    string(httpErr.Type),
			Code:    httpErr.Code,
			Message: httpErr.Message,
		}
		if appErr.Cause != nil {
			response.Details = appErr.Cause.Error()
		}
		return c.JSON(httpErr.StatusCode, response)
	}

	// If not an AppError, wrap it and return internal server error
	wrappedErr := Wrap(err, TypeServer, "INTERNAL_ERROR", "internal server error")
	httpErr := ErrorToHTTP(wrappedErr)
	response := HTTPResponse{
		Type:    string(httpErr.Type),
		Code:    httpErr.Code,
		Message: httpErr.Message,
		Details: err.Error(),
	}
	return c.JSON(httpErr.StatusCode, response)
}

// handleError centralizes error handling for HTTP responses.
func handleError(c echo.Context, err error, expectedType ErrorType, wrapCode, wrapMsg string) error {
	var appErr *GoInferError
	if errors.As(err, &appErr) && appErr.Type == expectedType {
		httpErr := ErrorToHTTP(appErr)
		resp := HTTPResponse{
			Type:    string(httpErr.Type),
			Code:    httpErr.Code,
			Message: httpErr.Message,
		}
		if appErr.Cause != nil {
			resp.Details = appErr.Cause.Error()
		}
		return c.JSON(httpErr.StatusCode, resp)
	}
	// Not the expected type: wrap and forward
	wrapped := Wrap(err, expectedType, wrapCode, wrapMsg)
	return ErrorToEchoResponse(c, wrapped)
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

// ErrorHandler is a centralized error handler for Echo middleware.
func ErrorHandler(err error, c echo.Context) error {
	// Log the error for debugging
	c.Logger().Error(err)

	// Return a standardized error response
	return ErrorToEchoResponse(c, err)
}
