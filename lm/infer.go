package lm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)


// Type Definitions

// InferenceError represents a structured error for language model inference
type InferenceError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Context    interface{} `json:"context,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
	TokenCount int         `json:"token_count,omitempty"`
	Stage      string      `json:"stage,omitempty"`
}

// Error implements the error interface
func (e *InferenceError) Error() string {
	return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Context)
}


// Constants

// Error codes for inference operations
const (
	ErrCodeModelNotLoaded     = "MODEL_NOT_LOADED"
	ErrCodeInferenceFailed    = "INFERENCE_FAILED"
	ErrCodeStreamFailed       = "STREAM_FAILED"
	ErrCodeInvalidParams      = "INVALID_PARAMS"
)


// Main Inference Functions

// Infer performs language model inference
func Infer(
	prompt string,
	template string,
	params types.InferenceParams,
	c echo.Context,
	ch chan<- types.StreamedMessage,
	errCh chan<- types.StreamedMessage,
) {
	if !state.IsModelLoaded {
		errCh <- createErrorMessage(1, "no model loaded")
	}

	finalPrompt := strings.Replace(template, "{prompt}", prompt, 1)
	logVerboseInfo(finalPrompt, 0, 0, 0) // Initial verbose logging with prompt

	if state.IsDebug {
		fmt.Println("Inference params:")
		fmt.Printf("%+v\n\n", params)
	}

	ntokens := 0
	enc := json.NewEncoder(c.Response())

	startThinking := time.Now()
	var thinkingElapsed time.Duration
	var startEmitting time.Time

	state.IsInfering = true
	state.ContinueInferingController = true

	res, err := state.Lm.Predict(finalPrompt, llama.SetTokenCallback(func(token string) bool {
		streamDeltaMsg(ntokens, token, enc, c, params, startThinking, &thinkingElapsed, &startEmitting)
		return state.ContinueInferingController
	}),
		llama.SetTokens(params.NPredict),
		llama.SetThreads(params.Threads),
		llama.SetTopK(params.TopK),
		llama.SetTopP(params.TopP),
		llama.SetTemperature(params.Temperature),
		llama.SetStopWords(params.StopPrompts...),
		llama.SetFrequencyPenalty(params.FrequencyPenalty),
		llama.SetPresencePenalty(params.PresencePenalty),
		llama.SetPenalty(params.RepeatPenalty),
		llama.SetRopeFreqBase(1e6),
	)

	state.IsInfering = false

	if err != nil {
		state.ContinueInferingController = false
		errCh <- createErrorMessage(ntokens+1, "inference error")
	}

	if !state.ContinueInferingController {
		return
	}

	if params.Stream {
		err := sendLlamaStreamTermination(c)
		if err != nil {
			state.ContinueInferingController = false
			errCh <- createErrorMessage(ntokens+1, "cannot send stream termination")
			fmt.Printf("Error sending stream termination: %v\n", err)
		}
	}

	stats, _ := calculateStats(ntokens, thinkingElapsed, startEmitting) // Ignore tps return value
	endmsg, err := createResult(res, stats, enc, c, params)
	if err != nil {
		state.ContinueInferingController = false
		errCh <- createErrorMessage(ntokens+1, "cannot create result msg")
	}

	if state.ContinueInferingController {
		ch <- endmsg
	}
}


// Streaming Functions

// StreamMsg streams a message to the client
func StreamMsg(msg types.StreamedMessage, c echo.Context, enc *json.Encoder) error {
	c.Response().Write([]byte("data: "))
	if err := enc.Encode(msg); err != nil {
		return fmt.Errorf("failed to encode stream message: %w", err)
	}
	c.Response().Write([]byte("\n"))
	c.Response().Flush()
	return nil
}

// sendLlamaStreamTermination sends stream termination message
func sendLlamaStreamTermination(c echo.Context) error {
	c.Response().Write([]byte("data: [DONE]\n\n"))
	c.Response().Flush()
	return nil
}


// streamDeltaMsg handles token processing during prediction
func streamDeltaMsg(ntokens int, token string, enc *json.Encoder, c echo.Context, params types.InferenceParams, startThinking time.Time, thinkingElapsed *time.Duration, startEmitting *time.Time) error {
	if ntokens == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)
		err := sendStartEmittingMessage(enc, c, params, ntokens, *thinkingElapsed)
		if err != nil {
			fmt.Printf("Error emitting msg: %v\n", err)
			state.ContinueInferingController = false
			return err
		}
	}

	if !state.ContinueInferingController {
		return nil
	}

	if state.IsVerbose {
		fmt.Print(token)
	}

	if !params.Stream {
		return nil
	}

	tmsg := types.StreamedMessage{
		Content: token,
		Num:     ntokens,
		MsgType: types.TokenMsgType,
	}
	err := StreamMsg(tmsg, c, enc)
	if err != nil {
		fmt.Printf("Error streaming delta message: %v\n", err)
		return err
	}

	return nil
}

// sendStartEmittingMessage sends the start_emitting message to the client
func sendStartEmittingMessage(enc *json.Encoder, c echo.Context, params types.InferenceParams, ntokens int, thinkingElapsed time.Duration) error {
	if !params.Stream || !state.ContinueInferingController {
		return nil
	}

	smsg := types.StreamedMessage{
		Num:     ntokens,
		Content: "start_emitting",
		MsgType: types.SystemMsgType,
		Data: map[string]interface{}{
			"thinking_time":        thinkingElapsed,
			"thinking_time_format": thinkingElapsed.String(),
		},
	}

	err := StreamMsg(smsg, c, enc)
	if err != nil {
		fmt.Printf("Error streaming start_emitting message: %v\n", err)
	}
	time.Sleep(2 * time.Millisecond) // Give some time to stream this message
	return err
}


// Utility Functions

// createErrorMessage sends an error message through the error channel
func createErrorMessage(ntokens int, content string) types.StreamedMessage {
	return types.StreamedMessage{
		Num:     ntokens,
		Content: content,
		MsgType: types.ErrorMsgType,
	}
}

// logVerboseInfo logs verbose information about the inference process
func logVerboseInfo(finalPrompt string, thinkingElapsed time.Duration, emittingElapsed time.Duration, ntokens int) {
	if state.IsVerbose {
		fmt.Println("---------- prompt ----------")
		fmt.Println(finalPrompt)
		fmt.Println("----------------------------")
		fmt.Println("Thinking ..")

		if thinkingElapsed > 0 {
			fmt.Println("Thinking time:", thinkingElapsed)
			fmt.Println("Emitting ..")
		}

		if emittingElapsed > 0 {
			fmt.Println("Emitting time:", emittingElapsed)
		}

		totalTime := thinkingElapsed + emittingElapsed
		fmt.Println("Total time:", totalTime)

		tpsRaw := float64(ntokens) / emittingElapsed.Seconds()
		tps, err := strconv.ParseFloat(fmt.Sprintf("%.2f", tpsRaw), 64)
		if err != nil {
			tps = 0.0
		}
		fmt.Println("Tokens per seconds", tps)
		fmt.Println("Tokens emitted", ntokens)
	}
}


// Statistics Functions

// calculateStats calculates inference statistics
func calculateStats(ntokens int, thinkingElapsed time.Duration, startEmitting time.Time) (types.InferenceStats, float64) {
	emittingElapsed := time.Since(startEmitting)
	tpsRaw := float64(ntokens) / emittingElapsed.Seconds()
	tps, err := strconv.ParseFloat(fmt.Sprintf("%.2f", tpsRaw), 64)
	if err != nil {
		tps = 0.0
	}

	totalTime := thinkingElapsed + emittingElapsed

	return types.InferenceStats{
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


// Result Creation Functions

// createResult creates the final result message to the client
func createResult(res string, stats types.InferenceStats, enc *json.Encoder, c echo.Context, params types.InferenceParams) (types.StreamedMessage, error) {
	result := types.InferenceResult{
		Text:  res,
		Stats: stats,
	}

	endmsg := types.StreamedMessage{}

	b, err := json.Marshal(&result)
	if err != nil {
		return endmsg, fmt.Errorf("error marshaling result: %w", err)
	}

	var _res map[string]interface{}
	err = json.Unmarshal(b, &_res)
	if err != nil {
		return endmsg, fmt.Errorf("error unmarshaling result: %w", err)
	}

	endmsg = types.StreamedMessage{
		Num:     stats.TotalTokens + 1,
		Content: "result",
		MsgType: types.SystemMsgType,
		Data:    _res,
	}

	if params.Stream {
		err := StreamMsg(endmsg, c, enc)
		if err != nil {
			return endmsg, fmt.Errorf("error streaming result message: %w", err)
		}
	}

	return endmsg, nil
}
