// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// gic is the Go Infer Context package.
package gic

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// GenReqID generates a unique request ID for correlation.
func GenReqID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// LogCtxAwareError logs an error with context information.
func LogCtxAwareError(ctx context.Context, operation string, err error) {
	if err == nil {
		return
	}

	fmt.Printf("INF: [%s] %s: %v\n", getReqID(ctx), operation, err)
}

// getReqID extracts the request ID from context or generates a new one.
func getReqID(ctx context.Context) string {
	reqID := ctx.Value("requestID")
	if reqID == nil {
		return GenReqID()
	}
	id, ok := reqID.(string)
	if ok {
		return id
	}
	// Fallback – generate a new ID if the stored value is not a string
	return GenReqID()
}
