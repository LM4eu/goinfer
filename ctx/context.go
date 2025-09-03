// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package ctx

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// GenerateRequestID generates a unique request ID for correlation.
func GenerateRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// getRequestID extracts the request ID from context or generates a new one.
func getRequestID(ctx context.Context) string {
	requestID := ctx.Value("requestID")
	if requestID == nil {
		return GenerateRequestID()
	}
	rid, ok := requestID.(string)
	if ok {
		return rid
	}
	// Fallback – generate a new ID if the stored value is not a string
	return GenerateRequestID()
}

// wrapErr wraps an error with context information.
func wrapErr(ctx context.Context, operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("[%s] %s: %w", getRequestID(ctx), operation, err)
}

// LogContextAwareError logs an error with context information.
func LogContextAwareError(ctx context.Context, operation string, err error) {
	if err == nil {
		return
	}

	fmt.Printf("INFO: [%s] %s: %v\n", getRequestID(ctx), operation, err)
}

// CheckContextCancelled checks if context is canceled and returns an appropriate error.
func CheckContextCancelled(ctx context.Context, operation string) error {
	err := ctx.Err()
	if err != nil {
		return wrapErr(ctx, operation, fmt.Errorf("context canceled: %w", err))
	}
	return nil
}
