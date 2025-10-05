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
	"fmt"
	"runtime"
	"strconv"
	"time"
)

type (
	// Error implements the error structure defined in JSON-RPC 2.0.
	Error struct {
		Data    Data   `json:"data,omitempty"`
		Message string `json:"msg,omitempty"`
		Code    Code   `json:"code,omitempty"`
	}

	Data struct {
		Time     time.Time      `json:"time,omitempty"`
		Cause    error          `json:"cause,omitempty"`
		Params   map[string]any `json:"params,omitempty"`
		Function string         `json:"function,omitempty"`
		FileLine string         `json:"file_line,omitempty"`
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
func wrap(cause error, code Code, msg string, args ...any) *Error {
	err := &Error{
		Code:    code,
		Message: msg,
		Data: Data{
			Time:  time.Now(),
			Cause: cause,
		},
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip 3 calls in the callstack: [runtime.Callers, wrap, New/Wrap]
	if pcs[0] != 0 {
		fs := runtime.CallersFrames([]uintptr{pcs[0]})
		f, _ := fs.Next()
		err.Data.Function = f.Function
		err.Data.FileLine = f.File
		if f.Line != 0 {
			err.Data.FileLine += ":" + strconv.Itoa(f.Line)
		}
	}

	err.Data.Params = make(map[string]any, (len(args)+1)/2)
	for len(args) > 0 {
		var key string
		var val any
		key, val, args = getPairRest(args)
		err.Data.Params[key] = val
	}

	return err
}

func getPairRest(args []any) (key string, val any, rest []any) {
	if len(args) == 1 {
		return "!BADKEY", args[0], nil
	}
	key, ok := args[0].(string)
	if !ok {
		key = fmt.Sprint(args[0])
	}
	return key, args[1], args[2:]
}

// Error implements the error interface.
func (e *Error) Error() string {
	str := e.Message + " (" + strconv.Itoa(int(e.Code)) + ")"
	for key, val := range e.Data.Params {
		str += " " + key + "=" + fmt.Sprint(val)
	}
	if e.Data.Cause != nil {
		str += " cause: " + e.Data.Cause.Error()
	}
	if e.Data.Function != "" {
		str += " in " + e.Data.Function
	}
	if e.Data.FileLine != "" {
		str += " " + e.Data.FileLine
	}
	if !e.Data.Time.IsZero() {
		str += " " + e.Data.Time.Format("2006-01-02 15:04:05.999")
	}
	return str
}

// Unwrap returns the underlying error for error unwrapping.
func (e *Error) Unwrap() error {
	return e.Data.Cause
}
