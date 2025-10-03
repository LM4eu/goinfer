// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package gie (Go Infer Error) implements the gie.Error to be MCP compliant,
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
//	msg  string providing a short description of the error (one concise single sentence).
//
//	data     optional, any type, additional information about the error.
package gie

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"time"
)

type (
	// Error implements the error structure defined in JSON-RPC 2.0.
	Error struct {
		Message string `json:"msg,omitempty"`
		Data    Data   `json:"data,omitempty"`
		Code    Code   `json:"code,omitempty"`
	}

	Data struct {
		Cause  any          `                          json:"cause,omitempty"`
		Source *slog.Source `                          json:"source,omitempty"`
		Attrs  []slog.Attr  `attrs:"details,omitempty"`
	}

	// Code represents the type of error.
	Code int
)

const (
	// Invalid indicates validation errors.
	Invalid Code = iota + -32149
	// ConfigErr indicates configuration errors.
	ConfigErr
	// InferErr indicates inference-related errors.
	InferErr
	// UserAbort occurs when /abort is requested.
	UserAbort
	// ServerErr indicates server-related errors.
	ServerErr
	// Timeout indicates timeout-related errors.
	Timeout
	// NotFound indicates resource not found errors.
	NotFound
)

// New creates a new gie.Error.
func New(code Code, msg string, args ...any) *Error {
	return wrap(nil, code, msg, args...)
}

// Wrap an existing error.
func Wrap(err error, code Code, msg string, args ...any) *Error {
	return wrap(err, code, msg, args...)
}

//nolint:revive // wrap is the common function for New() and Wrap().
func wrap(err error, code Code, msg string, args ...any) *Error {
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip 3 calls in the callstack: [runtime.Callers, wrap, New/Wrap]
	record := slog.NewRecord(time.Now(), 0, "", pcs[0])
	record.Add(args...)

	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(a slog.Attr) bool { attrs = append(attrs, a); return true })

	return &Error{
		Code:    code,
		Message: msg,
		Data: Data{
			Attrs:  attrs,
			Source: record.Source(),
			Cause:  err,
		},
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	str := e.Message + " (" + strconv.Itoa(int(e.Code)) + ")"
	for _, a := range e.Data.Attrs {
		str += " " + a.Key + "=" + a.Value.String()
	}
	if e.Data.Cause != nil {
		str += " cause: " + fmt.Sprint(e.Data.Cause)
	}
	if e.Data.Source != nil {
		str += " in " + e.Data.Source.Function +
			" " + e.Data.Source.File +
			":" + strconv.Itoa(e.Data.Source.Line)
	}
	return str
}

// Unwrap returns the underlying error for error unwrapping.
func (e *Error) Unwrap() error {
	if e.Data.Cause == nil {
		return nil
	}
	err, ok := e.Data.Cause.(error)
	if ok {
		return err
	}
	return errors.New(fmt.Sprint(e.Data.Cause))
}
