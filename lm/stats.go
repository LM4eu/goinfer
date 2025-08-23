package lm

import (
	"errors"
	"strconv"
	"time"
)

// InferenceStats holds unified statistics about inference.
type InferenceStats struct {
	ThinkingTime       float64 `json:"thinkingTime"`
	ThinkingTimeFormat string  `json:"thinkingTimeFormat"`
	EmitTime           float64 `json:"emitTime"`
	EmitTimeFormat     string  `json:"emitTimeFormat"`
	TotalTime          float64 `json:"totalTime"`
	TotalTimeFormat    string  `json:"totalTimeFormat"`
	TokensPerSecond    float64 `json:"tokensPerSecond"`
	TotalTokens        int     `json:"totalTokens"`
}

// CalculateInferenceStats calculates inference statistics from raw data
func CalculateInferenceStats(ntokens int, thinkingElapsed time.Duration, startEmitting time.Time) (InferenceStats, float64) {
	var errs []error
	
	if ntokens < 0 {
		errs = append(errs, errors.New("token count cannot be negative"))
	}
	
	if startEmitting.IsZero() && ntokens > 0 {
		errs = append(errs, errors.New("startEmitting time is required when tokens > 0"))
	}
	
	if len(errs) > 0 {
		return InferenceStats{}, 0.0
	}

	emittingElapsed := time.Since(startEmitting)
	var tps float64
	
	if emittingElapsed.Seconds() > 0 {
		tpsRaw := float64(ntokens) / emittingElapsed.Seconds()
		var err error
		tps, err = strconv.ParseFloat(strconv.FormatFloat(tpsRaw, 'f', 2, 64), 64)
		if err != nil {
			tps = 0.0
		}
	} else {
		tps = 0.0
	}

	totalTime := thinkingElapsed + emittingElapsed

	return InferenceStats{
		ThinkingTime:       thinkingElapsed.Seconds(),
		ThinkingTimeFormat: thinkingElapsed.String(),
		EmitTime:           emittingElapsed.Seconds(),
		EmitTimeFormat:     emittingElapsed.String(),
		TotalTime:          totalTime.Seconds(),
		TotalTimeFormat:    totalTime.String(),
		TokensPerSecond:    tps,
		TotalTokens:        ntokens,
	}, tps
}