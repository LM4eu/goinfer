// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package gie (Go Infer Error) implements the GoinferError to be MCP compliant,
// producing the error object as specified in JSON-RPC 2.0:
// https://www.jsonrpc.org/specification#error_object
//
//	code must be a number
//	 -32768 to -32000  Reserved for pre-defined errors:
//	 -32700            Parse error, the server received an invalid JSON, or had an issue while parsing the JSON text
//	 -32603            Internal JSON-RPC error
//	 -32602            Invalid method parameters
//	 -32601            Method not found, the method does not exist or is not available
//	 -32600            Invalid Request, the JSON sent is not a valid Request object
//	 -32099 to -32000  Implementation-defined server-errors
//
//	message  string providing a short description of the error (one concise single sentence).
//
//	data     optional, any type, additional information about the error.
package gie

import (
	"errors"
	"fmt"
	"strconv"
)

type (
	// ErrorCode represents the type of error.
	ErrorCode int

	// GoinferError is a structured error that includes type, code, and message.
	GoinferError struct {
		Cause   any       `json:"data,omitempty"` // Cause is serialized "data" in HTTP error response (JSON-RPC 2.0)
		Message string    `json:"message,omitempty"`
		Code    ErrorCode `json:"code,omitempty"`
	}
)

const (
	// Invalid indicates validation errors.
	Invalid ErrorCode = iota + -32149
	// ConfigErr indicates configuration errors.
	ConfigErr
	// InferErr indicates inference-related errors.
	InferErr
	// ServerErr indicates server-related errors.
	ServerErr
	// Timeout indicates timeout-related errors.
	Timeout
	// NotFound indicates resource not found errors.
	NotFound
)

var (
	// Validation errors.

	// ErrInvalidPrompt indicates an invalid prompt error.
	ErrInvalidPrompt = New(Invalid, "invalid prompt (mandatory)")
	// ErrInvalidFormat indicates an invalid request format error.
	ErrInvalidFormat = New(Invalid, "invalid request format")
	// ErrInvalidParams indicates invalid parameter values error.
	ErrInvalidParams = New(Invalid, "invalid parameter values")

	// Configuration errors.

	// ErrConfigValidation indicates a configuration validation error.
	ErrConfigValidation = New(ConfigErr, "config validation failed")
	// ErrAPIKeyMissing indicates a missing API key error.
	ErrAPIKeyMissing = New(ConfigErr, "API key is missing")
	// ErrInvalidAPIKey indicates an invalid API key format error.
	ErrInvalidAPIKey = New(ConfigErr, "invalid API key format")

	// Inference errors.

	// ErrChanClosed indicates a channel closed unexpectedly error.
	ErrChanClosed = New(InferErr, "channel closed unexpectedly")
	// ErrClientCanceled indicates a request canceled by client error.
	ErrClientCanceled = New(InferErr, "request canceled by client")

	// Timeout errors.

	// ErrReqTimeout indicates a request timeout error.
	ErrReqTimeout = New(Timeout, "request timeout")
)

// New creates a New GoinferError.
func New(code ErrorCode, message string) *GoinferError {
	return &GoinferError{
		Code:    code,
		Message: message,
	}
}

func NewWithData(code ErrorCode, message string, data any) *GoinferError {
	return &GoinferError{
		Code:    code,
		Message: message,
		Cause:   data,
	}
}

// Wrap wraps an existing error with an GoinferError.
func Wrap(err error, code ErrorCode, message string) *GoinferError {
	var giErr *GoinferError
	if errors.As(err, &giErr) {
		// merge errors when appropriate
		if giErr.Cause == "" {
			giErr.Cause = err
			return giErr
		} else if len(message) < 11 {
			giErr.Message = message + ": " + giErr.Message
			return giErr
		}
	}

	return &GoinferError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// Error implements the error interface.
func (e *GoinferError) Error() string {
	str := e.Message + " (" + strconv.Itoa(int(e.Code)) + ")"
	if e.Cause != nil {
		str += " cause: " + fmt.Sprint(e.Cause)
	}
	return str
}

// Unwrap returns the underlying error for error unwrapping.
func (e *GoinferError) Unwrap() error {
	err, ok := e.Cause.(error)
	if ok {
		return err
	}
	return errors.New(fmt.Sprint(e.Cause))
}
