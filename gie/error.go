// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// gie is the Go Infer Error package.
package gie

import (
	"fmt"
)

type (
	// ErrorType represents the type of error.
	ErrorType string

	// GogiError is a structured error that includes type, code, and message.
	GogiError struct {
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
)

// New creates a new GogiError.
func New(errType ErrorType, code string, message string) *GogiError {
	return &GogiError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with an GogiError.
func Wrap(err error, errType ErrorType, code, message string) *GogiError {
	return &GogiError{
		Type:    errType,
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// Error implements the error interface.
func (e *GogiError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error unwrapping.
func (e *GogiError) Unwrap() error {
	return e.Cause
}
