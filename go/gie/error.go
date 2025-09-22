// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package gie is the Go Infer Error package.
package gie

type (
	// ErrorType represents the type of error.
	ErrorType string

	// GoInferError is a structured error that includes type, code, and message.
	GoInferError struct {
		Cause   error     `json:"details,omitempty"` // Cause is serialized "details" in HTTP error response (JSON)
		Type    ErrorType `json:"type,omitempty"`
		Code    string    `json:"code,omitempty"`
		Message string    `json:"message,omitempty"`
	}
)

const (
	// TypeValidation indicates validation errors.
	TypeValidation ErrorType = "validation"
	// TypeConfiguration indicates configuration errors.
	TypeConfiguration ErrorType = "config"
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
	ErrPromptRequired = newGIE(TypeValidation, "PROMPT_REQUIRED", "prompt is required")
	ErrInvalidPrompt  = newGIE(TypeValidation, "INVALID_PROMPT", "prompt must be a string")
	ErrModelNotLoaded = newGIE(TypeValidation, "MODEL_NOT_LOADED", "model not loaded")
	ErrInvalidFormat  = newGIE(TypeValidation, "INVALID_FORMAT", "invalid request format")
	ErrInvalidParams  = newGIE(TypeValidation, "INVALID_PARAMS", "invalid parameter values")

	// Configuration errors.
	ErrConfigLoadFailed   = newGIE(TypeConfiguration, "CONFIG_LOAD_FAILED", "failed to load configuration")
	ErrConfigValidation   = newGIE(TypeConfiguration, "CONFIG_VALIDATION", "configuration validation failed")
	ErrAPIKeyMissing      = newGIE(TypeConfiguration, "API_KEY_MISSING", "API key is missing")
	ErrInvalidAPIKey      = newGIE(TypeConfiguration, "INVALID_API_KEY", "invalid API key format")
	ErrModelFilesNotFound = newGIE(TypeConfiguration, "MODEL_FILES_NOT_FOUND", "no model files found")
	ErrSwapConfigFailed   = newGIE(TypeConfiguration, "SWAP_CONFIG_FAILED", "failed to configure proxy")

	// Inference errors.
	ErrInferRunning    = newGIE(TypeInference, "INFERENCE_RUNNING", "infer already running")
	ErrInferNotRunning = newGIE(TypeInference, "INFERENCE_NOT_RUNNING", "no inference running, nothing to abort")
	ErrInferFailed     = newGIE(TypeInference, "INFERENCE_FAILED", "infer failed")
	ErrInferCanceled   = newGIE(TypeInference, "INFERENCE_CANCELED", "infer canceled")
	ErrInferStopped    = newGIE(TypeInference, "INFERENCE_STOPPED", "infer stopped by controller")
	ErrChanClosed      = newGIE(TypeInference, "CHANNEL_CLOSED", "channel closed unexpectedly")
	ErrClientCanceled  = newGIE(TypeInference, "CLIENT_CANCELED", "request canceled by client")

	// Server errors.
	ErrServerStart    = newGIE(TypeServer, "SERVER_START_FAILED", "failed to start server")
	ErrServerShutdown = newGIE(TypeServer, "SERVER_SHUTDOWN_FAILED", "failed to shutdown server")
	ErrSwapShutdown   = newGIE(TypeServer, "SWAP_SHUTDOWN_FAILED", "failed to shutdown proxy")

	// Timeout errors.
	ErrReqTimeout    = newGIE(TypeTimeout, "REQUEST_TIMEOUT", "request timeout")
	ErrStreamTimeout = newGIE(TypeTimeout, "STREAM_TIMEOUT", "stream timeout")

	// NotFound errors.
	ErrModelNotFound    = newGIE(TypeNotFound, "MODEL_NOT_FOUND", "model not found")
	ErrResourceNotFound = newGIE(TypeNotFound, "RESOURCE_NOT_FOUND", "resource not found")

	// Unauthorized errors.
	ErrUnauthorized       = newGIE(TypeUnauthorized, "UNAUTHORIZED", "unauthorized")
	ErrInvalidCredentials = newGIE(TypeUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
)

// newGIE creates a new GoInferError.
func newGIE(errType ErrorType, code, message string) *GoInferError {
	return &GoInferError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with an GoInferError.
func Wrap(err error, errType ErrorType, code, message string) *GoInferError {
	return &GoInferError{
		Type:    errType,
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// Error implements the error interface.
func (e *GoInferError) Error() string {
	str := e.Code + ": " + e.Message
	if e.Cause != nil {
		str += " cause=" + e.Cause.Error()
	}
	return str
}

// Unwrap returns the underlying error for error unwrapping.
func (e *GoInferError) Unwrap() error {
	return e.Cause
}
