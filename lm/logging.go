// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package lm

import (
	"context"
	"fmt"
	"time"

	"github.com/LM4eu/goinfer/state"
)

// logMsg formats and logs a message with common context.
func logMsg(ctx context.Context, format string, args ...any) {
	if !state.Verbose {
		return
	}

	reqID := "req"
	if id := ctx.Value("requestID"); id != nil {
		if str, ok := id.(string); ok {
			reqID = str
		}
	}

	fmt.Printf("INFO: [%s] | c: %s | r: %s | %s\n",
		time.Now().Format(time.RFC3339), fmt.Sprintf("c-%d", time.Now().UnixNano()), reqID, fmt.Sprintf(format, args...))
}


// logError logs error information.
func logError(ctx context.Context, prefix, message string, err error) {
	if err != nil {
		logMsg(ctx, "%s | ERROR: %s - %v", prefix, message, err)
	} else {
		logMsg(ctx, "%s | ERROR: %s", prefix, message)
	}
}

// logToken logs token information.
func logToken(ctx context.Context, token string) {
	logMsg(ctx, "token: %s", token)
}

// // logOpStart logs the start of an operation.
// func logOpStart(ctx context.Context, operation string, details ...any) {
// 	if len(details) > 0 {
// 		logMsg(ctx, "%s | START | details: %v", operation, details)
// 	} else {
// 		logMsg(ctx, "%s | START", operation)
// 	}
// }
//
// // logOpEnd logs the end of an operation.
// func logOpEnd(ctx context.Context, operation string, duration time.Duration, success bool, details ...any) {
// 	status := "SUCCESS"
// 	if !success {
// 		status = "FAILED"
// 	}
//
// 	if len(details) > 0 {
// 		logMsg(ctx, "%s | END | status: %s | duration: %s | details: %v",
// 			operation, status, duration, details)
// 	} else {
// 		logMsg(ctx, "%s | END | status: %s | duration: %s",
// 			operation, status, duration)
// 	}
// }
//
// // logPerf logs performance metrics.
// func logPerf(ctx context.Context, operation string, metrics map[string]any) {
// 	logMsg(ctx, "%s | performance: %v", operation, metrics)
// }
