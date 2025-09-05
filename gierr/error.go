// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package gierr

import (
	"fmt"
	"net/http"
)

// ErrorType represents the type of error.
type (
	ErrorType string

	// GoInferError is a structured error that includes type, code, and message.
	GoInferError struct {
		Cause   error     `json:"cause,omitempty"`
		Type    ErrorType `json:"type"`
		Code    string    `json:"code"`
		Message string    `json:"message"`
	}

	// HTTPError represents an error with HTTP status code.
	HTTPError struct {
		*GoInferError

		StatusCode int
	}
)

const (
	// TypeValidation indicates validation errors.
	TypeValidation ErrorType = "validation"
	// TypeConfiguration indicates configuration errors.
	TypeConfiguration ErrorType = "configuration"
	// TypeInference indicates inference-related errors.
	TypeInference ErrorType = "inference"
	// TypeServer indicates server-related errors.
	TypeServer ErrorType = "server"
	// TypeTimeout indicates timeout-related errors.
	TypeTimeout ErrorType = "timeout"
	// TypeNotFound indicates resource not found errors.
	TypeNotFound ErrorType = "not_found"
	// TypeUnauthorized indicates authentication errors.
	TypeUnauthorized ErrorType = "unauthorized"
)

var (
	// Validation errors.
	ErrPromptRequired = New(TypeValidation, "PROMPT_REQUIRED", "prompt is required")
	ErrInvalidPrompt  = New(TypeValidation, "INVALID_PROMPT", "prompt must be a string")
	ErrModelNotLoaded = New(TypeValidation, "MODEL_NOT_LOADED", "model not loaded")
	ErrInvalidFormat  = New(TypeValidation, "INVALID_FORMAT", "invalid request format")
	ErrInvalidParams  = New(TypeValidation, "INVALID_PARAMS", "invalid parameter values")

	// Configuration errors.
	ErrConfigLoadFailed   = New(TypeConfiguration, "CONFIG_LOAD_FAILED", "failed to load configuration")
	ErrConfigValidation   = New(TypeConfiguration, "CONFIG_VALIDATION", "configuration validation failed")
	ErrAPIKeyMissing      = New(TypeConfiguration, "API_KEY_MISSING", "API key is missing")
	ErrInvalidAPIKey      = New(TypeConfiguration, "INVALID_API_KEY", "invalid API key format")
	ErrModelFilesNotFound = New(TypeConfiguration, "MODEL_FILES_NOT_FOUND", "no model files found")
	ErrProxyConfigFailed  = New(TypeConfiguration, "PROXY_CONFIG_FAILED", "failed to configure proxy")

	// Inference errors.
	ErrInferenceRunning    = New(TypeInference, "INFERENCE_RUNNING", "infer already running")
	ErrInferenceNotRunning = New(TypeInference, "INFERENCE_NOT_RUNNING", "no inference running, nothing to abort")
	ErrInferenceFailed     = New(TypeInference, "INFERENCE_FAILED", "infer failed")
	ErrInferenceCanceled   = New(TypeInference, "INFERENCE_CANCELED", "infer canceled")
	ErrInferenceStopped    = New(TypeInference, "INFERENCE_STOPPED", "infer stopped by controller")
	ErrChannelClosed       = New(TypeInference, "CHANNEL_CLOSED", "channel closed unexpectedly")
	ErrClientCanceled      = New(TypeInference, "CLIENT_CANCELED", "request canceled by client")

	// Server errors.
	ErrServerStart    = New(TypeServer, "SERVER_START_FAILED", "failed to start server")
	ErrServerShutdown = New(TypeServer, "SERVER_SHUTDOWN_FAILED", "failed to shutdown server")
	ErrProxyShutdown  = New(TypeServer, "PROXY_SHUTDOWN_FAILED", "failed to shutdown proxy")

	// Timeout errors.
	ErrRequestTimeout = New(TypeTimeout, "REQUEST_TIMEOUT", "request timeout")
	ErrStreamTimeout  = New(TypeTimeout, "STREAM_TIMEOUT", "stream timeout")

	// NotFound errors.
	ErrModelNotFound    = New(TypeNotFound, "MODEL_NOT_FOUND", "model not found")
	ErrResourceNotFound = New(TypeNotFound, "RESOURCE_NOT_FOUND", "resource not found")

	// Unauthorized errors.
	ErrUnauthorized       = New(TypeUnauthorized, "UNAUTHORIZED", "unauthorized")
	ErrInvalidCredentials = New(TypeUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")

	// Common HTTP error mappings.

	HTTPBadRequest = func(err *GoInferError) *HTTPError {
		return NewHTTPError(err.Type, err.Code, err.Message, http.StatusBadRequest)
	}
	HTTPUnauthorized = func(err *GoInferError) *HTTPError {
		return NewHTTPError(err.Type, err.Code, err.Message, http.StatusUnauthorized)
	}
	HTTPNotFound = func(err *GoInferError) *HTTPError {
		return NewHTTPError(err.Type, err.Code, err.Message, http.StatusNotFound)
	}
	HTTPStatusRequestTimeout = func(err *GoInferError) *HTTPError {
		return NewHTTPError(err.Type, err.Code, err.Message, http.StatusRequestTimeout)
	}
	HTTPInternalServerError = func(err *GoInferError) *HTTPError {
		return NewHTTPError(err.Type, err.Code, err.Message, http.StatusInternalServerError)
	}
)

// ErrorToHTTP converts an AppError to an HTTPError with appropriate status code.
func ErrorToHTTP(err *GoInferError) *HTTPError {
	switch err.Type {
	case TypeValidation:
		return HTTPBadRequest(err)
	case TypeUnauthorized:
		return HTTPUnauthorized(err)
	case TypeNotFound:
		return HTTPNotFound(err)
	case TypeTimeout:
		return HTTPStatusRequestTimeout(err)
	case TypeConfiguration, TypeInference, TypeServer:
		return HTTPInternalServerError(err)
	default:
		return HTTPInternalServerError(err)
	}
}

// Error implements the error interface.
func (e *GoInferError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error unwrapping.
func (e *GoInferError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError.
func New(errType ErrorType, code, message string) *GoInferError {
	return &GoInferError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with an AppError.
func Wrap(err error, errType ErrorType, code, message string) *GoInferError {
	return &GoInferError{
		Type:    errType,
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// NewHTTPError creates a new HTTPError.
func NewHTTPError(errType ErrorType, code, message string, statusCode int) *HTTPError {
	return &HTTPError{
		GoInferError: New(errType, code, message),
		StatusCode:   statusCode,
	}
}

// WrapHTTPError wraps an existing error with an HTTPError.
func WrapHTTPError(err error, errType ErrorType, code, message string, statusCode int) *HTTPError {
	return &HTTPError{
		GoInferError: Wrap(err, errType, code, message),
		StatusCode:   statusCode,
	}
}
