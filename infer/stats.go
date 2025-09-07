// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"errors"
	"fmt"
	"time"
)

// CalcInfStats calculates infer statistics from raw data. TODO: unused.
func CalcInfStats(tokenCount int, thinkingElapsed time.Duration, startEmitting time.Time) (InferStat, float64, error) {
	// Simple validation
	if tokenCount < 0 {
		return InferStat{}, 0.0, fmt.Errorf("invalid token count: %d", tokenCount)
	}

	if startEmitting.IsZero() && tokenCount > 0 {
		return InferStat{}, 0.0, errors.New("startEmitting time is required for token calculation")
	}

	emittingElapsed := time.Since(startEmitting)
	tokensPerSecond := 0.0

	if emittingElapsed.Seconds() > 0 {
		tokensPerSecond = float64(int((float64(tokenCount)/emittingElapsed.Seconds())*100)) / 100 // Round to 2 decimal places
	}

	totalTime := thinkingElapsed + emittingElapsed

	stats := InferStat{
		ThinkingTime:       thinkingElapsed.Seconds(),
		ThinkingTimeFormat: thinkingElapsed.String(),
		EmitTime:           emittingElapsed.Seconds(),
		EmitTimeFormat:     emittingElapsed.String(),
		TotalTime:          totalTime.Seconds(),
		TotalTimeFormat:    totalTime.String(),
		TokensPerSecond:    tokensPerSecond,
		TotalTokens:        tokenCount,
	}

	return stats, tokensPerSecond, nil
}
