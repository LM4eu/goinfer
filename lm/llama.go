package lm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// Type Definitions

// InferResult holds the result of inference.
type InferResult struct {
	Text  string     `json:"text"`
	Stats InferStats `json:"stats"`
}

// InferStats holds statistics about inference (alias for unified InferenceStats)
type InferStats = InferenceStats

// InferError represents a structured error for language model inference.
type InferError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Context any    `json:"context,omitempty"`
}

// Error implements the error interface.
func (e *InferError) Error() string {
	return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Context)
}

// Constants

// Error codes for inference operations.
const (
	ErrStreamDeltaMsgOpenAi = "STREAM_DELTA_OPENAI"
	ErrCodeModelNotLoaded   = "MODEL_NOT_LOADED"
	ErrCodeInferenceFailed  = "INFERENCE_FAILED"
	ErrCodeStreamFailed     = "STREAM_FAILED"
	ErrCodeInvalidParams    = "INVALID_PARAMS"
)

// Main Inference Functions

// Infer performs language model inference.
func Infer(query types.InferQuery, c echo.Context, ch chan<- types.StreamedMessage, errCh chan<- types.StreamedMessage) {
	// if !state.Llama.IsRunning() {
	// 	errCh <- createErrorMessage(1, "no model loaded")
	// 	return
	// }

	// Initialize statistics
	var stats InferenceStats
	LogVerboseInfo("Llama", &stats, query.Prompt)

	if state.Debug {
		fmt.Println("Inference params:")
		fmt.Printf("%+v\n\n", query.InferParams)
	}

	ntokens := 0
	// enc := json.NewEncoder(c.Response())

	// startThinking := time.Now()
	// var thinkingElapsed time.Duration
	// var startEmitting time.Time
	//
	// state.IsInferring = true
	// state.ContinueInferringController = true
	//
	//res, err := state.Llama.Predict(
	//	query,
	//	func(token string) bool {
	//		err := streamDeltaMsg(ntokens, token, enc, c, query.InferParams, startThinking, &thinkingElapsed, &startEmitting)
	//		if err != nil {
	//			errCh <- createErrorMessage(ntokens+1, "streamDeltaMsg error")
	//		}
	//		return state.ContinueInferringController
	//	})
	//
	// state.IsInferring = false
	//
	// if err != nil {
	// 	state.ContinueInferringController = false
	// 	errCh <- createErrorMessage(ntokens+1, "inference error")
	// }

	if !state.ContinueInferringController {
		return
	}

	if query.InferParams.Stream {
		err := SendStreamTermination(c)
		if err != nil {
			state.ContinueInferringController = false
			errCh <- createErrorMessage(ntokens+1, "cannot send stream termination")
			LogError("Llama", "cannot send stream termination", err)
		}
	}

	// stats := statsCollector.GetStats()
	// endmsg, err := createResult(res, stats, enc, c, query.InferParams)
	// if err != nil {
	// 	state.ContinueInferringController = false
	// 	errCh <- createErrorMessage(ntokens+1, "cannot create result msg")
	// }

	// if state.ContinueInferringController {
	// 	ch <- endmsg
	// }
}

// Streaming Functions


// streamDeltaMsg handles token processing during prediction.
func streamDeltaMsg(ntokens int, token string, enc *json.Encoder, c echo.Context, params types.InferParams, startThinking time.Time, startEmitting *time.Time, thinkingElapsed *time.Duration) error {
	if ntokens == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		err := sendStartEmittingMessage(enc, c, params, ntokens, *thinkingElapsed)
		if err != nil {
			LogError("Llama", "cannot emit start message", err)
			state.ContinueInferringController = false
			return err
		}
	}

	if !state.ContinueInferringController {
		return nil
	}

	LogToken(token)

	if !params.Stream {
		return nil
	}

	tmsg := types.StreamedMessage{
		Content: token,
		Num:     ntokens,
		MsgType: types.TokenMsgType,
	}

	_, err := c.Response().Write([]byte("data: "))
	if err != nil {
		return fmt.Errorf("failed to write stream begin: %w", err)
	}

	err = enc.Encode(tmsg)
	if err != nil {
		return fmt.Errorf("failed to encode stream message: %w", err)
	}

	_, err = c.Response().Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write stream message: %w", err)
	}

	c.Response().Flush()
	return nil
}

// sendStartEmittingMessage sends the start_emitting message to the client.
func sendStartEmittingMessage(enc *json.Encoder, c echo.Context, params types.InferParams, ntokens int, thinkingElapsed time.Duration) error {
	if !params.Stream || !state.ContinueInferringController {
		return nil
	}

	smsg := StreamSystemMessage("start_emitting", ntokens, map[string]any{
		"thinking_time":        thinkingElapsed,
		"thinking_time_format": thinkingElapsed.String(),
	})

	err := StreamMsg(smsg, c, enc)
	if err != nil {
		LogError("Llama", "cannot stream start_emitting message", err)
	}
	time.Sleep(2 * time.Millisecond) // Give some time to stream this message
	return err
}

// Utility Functions

// createErrorMessage sends an error message through the error channel.
func createErrorMessage(ntokens int, content string) types.StreamedMessage {
	return types.StreamedMessage{
		Num:     ntokens,
		Content: content,
		MsgType: types.ErrorMsgType,
	}
}


// Statistics Functions

// calculateStats calculates inference statistics (now uses the unified function).
func calculateStats(ntokens int, thinkingElapsed time.Duration, startEmitting time.Time) (InferStats, float64) {
	stats, tps := CalculateInferenceStats(ntokens, thinkingElapsed, startEmitting)
	return stats, tps
}

// Result Creation Functions

// createResult creates the final result message to the client.
func createResult(res string, stats InferStats, enc *json.Encoder, c echo.Context, params types.InferParams) (types.StreamedMessage, error) {
	result := InferResult{
		Text:  res,
		Stats: stats,
	}

	endmsg := types.StreamedMessage{}

	b, err := json.Marshal(&result)
	if err != nil {
		return endmsg, fmt.Errorf("error marshalling result: %w", err)
	}

	var _res map[string]any
	err = json.Unmarshal(b, &_res)
	if err != nil {
		return endmsg, fmt.Errorf("error unmarshalling result: %w", err)
	}

	endmsg = types.StreamedMessage{
		Num:     stats.TotalTokens + 1,
		Content: "result",
		MsgType: types.SystemMsgType,
		Data:    _res,
	}

	if params.Stream {
		err := StreamMsg(&endmsg, c, enc)
		if err != nil {
			return endmsg, fmt.Errorf("error streaming result message: %w", err)
		}
	}

	return endmsg, nil
}
