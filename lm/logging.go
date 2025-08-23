package lm

import (
	"fmt"

	"github.com/synw/goinfer/state"
)

// LogVerboseInfo logs verbose information using the unified statistics calculation
func LogVerboseInfo(prefix string, stats *InferenceStats, finalPrompt string) {
	if state.Verbose {
		fmt.Println("----------", prefix, "prompt ----------")
		fmt.Println(finalPrompt)
		fmt.Println("----------------------------")
		fmt.Println("Thinking ..")
		fmt.Println("Thinking time:", stats.ThinkingTimeFormat)
		fmt.Println("Emitting ..")
		fmt.Println("Emitting time:", stats.EmitTimeFormat)
		fmt.Println("Total time:", stats.TotalTimeFormat)
		fmt.Println("Tokens per seconds", stats.TokensPerSecond)
		fmt.Println("Tokens emitted", stats.TotalTokens)
	}
}

// LogInfo logs basic information with the specified prefix
func LogInfo(prefix, message string) {
	if state.Verbose {
		fmt.Println(prefix, ":", message)
	}
}

// LogError logs error information with the specified prefix
func LogError(prefix, message string, err error) {
	if state.Verbose {
		if err != nil {
			fmt.Println(prefix, "ERROR:", message, "-", err)
		} else {
			fmt.Println(prefix, "ERROR:", message)
		}
	}
}

// LogToken logs token information during streaming
func LogToken(token string) {
	if state.Verbose {
		fmt.Print(token)
	}
}