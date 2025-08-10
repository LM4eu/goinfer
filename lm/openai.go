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

type OpenAiChatCompletion struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAiChoice `json:"choices"`
	Usage   openAiUsage    `json:"usage"`
}

type delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deltaChoice struct {
	Delta        delta  `json:"delta"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason,omitempty"`
}

type openAiChatCompletionDeltaResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []deltaChoice `json:"choices"`
}

type openAiChoice struct {
	Index        int           `json:"index"`
	Message      openAiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Main Inference Functions

// InferOpenAi performs OpenAI model inference.
func InferOpenAi(
	prompt string,
	template string,
	params types.InferParams,
	c echo.Context,
	ch chan<- OpenAiChatCompletion,
	errCh chan<- error,
) {
	if !state.IsModelLoaded {
		errCh <- createErrorMessageOpenAi(1, "no model loaded", nil, ErrCodeModelNotLoaded)
		return
	}

	finalPrompt := strings.Replace(template, "{prompt}", prompt, 1)
	logOpenAiVerboseInfo(finalPrompt, 0, 0, 0) // Initial verbose logging with prompt

	if state.IsDebug {
		fmt.Println("Inference params:")
		fmt.Printf("%+v\n\n", params)
	}

	ntokens := 0
	enc := json.NewEncoder(c.Response())

	startThinking := time.Now()
	var thinkingElapsed time.Duration
	var startEmitting time.Time

	state.IsInferring = true
	state.ContinueInferringController = true

	res, err := state.Lm.Predict(finalPrompt, llama.SetTokenCallback(func(token string) bool {
		err := streamDeltaMsgOpenAi(ntokens, token, enc, c, params, startThinking, &thinkingElapsed, &startEmitting)
		if err != nil {
			errCh <- createErrorMessageOpenAi(ntokens+1, "streamDeltaMsgOpenAi error", err, ErrStreamDeltaMsgOpenAi)
		}
		return state.ContinueInferringController
	}),
		llama.SetTokens(params.MaxTokens),
		llama.SetTopK(params.TopK),
		llama.SetTopP(params.TopP),
		llama.SetTemperature(params.Temperature),
		llama.SetStopWords(params.StopPrompts...),
		llama.SetFrequencyPenalty(params.FrequencyPenalty),
		llama.SetPresencePenalty(params.PresencePenalty),
		llama.SetPenalty(params.RepeatPenalty),
		llama.SetRopeFreqBase(1e6),
	)

	state.IsInferring = false

	if err != nil {
		state.ContinueInferringController = false
		errCh <- createErrorMessageOpenAi(ntokens+1, "inference error", err, ErrCodeInferenceFailed)
	}

	if !state.ContinueInferringController {
		return
	}

	if params.Stream {
		err := sendOpenAiStreamTermination(c)
		if err != nil {
			state.ContinueInferringController = false
			errCh <- createErrorMessageOpenAi(ntokens+1, "cannot send stream termination", err, ErrCodeStreamFailed)
			fmt.Printf("Error sending stream termination: %v\n", err)
		}
	}

	if state.ContinueInferringController {
		ch <- createOpenAiResult(ntokens, res)
	}
}

// Streaming Functions

// streamOpenAiMsg streams a message to the client.
func streamOpenAiMsg(msg openAiChatCompletionDeltaResponse, c echo.Context, enc *json.Encoder) error {
	_, err := c.Response().Write([]byte("data: "))
	if err != nil {
		return fmt.Errorf("failed to write stream begin: %w", err)
	}

	err = enc.Encode(msg)
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

// sendOpenAiStreamTermination sends stream termination message.
func sendOpenAiStreamTermination(c echo.Context) error {
	_, err := c.Response().Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		return fmt.Errorf("failed to write stream termination: %w", err)
	}
	c.Response().Flush()
	return nil
}

// createOpenAiDeltaMessage creates a delta message for streaming.
func createOpenAiDeltaMessage(ntokens int, token string) openAiChatCompletionDeltaResponse {
	return openAiChatCompletionDeltaResponse{
		ID:      strconv.Itoa(ntokens),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   state.LoadedModel,
		Choices: []deltaChoice{
			{
				Index:        ntokens,
				FinishReason: "",
				Delta: delta{
					Role:    "assistant",
					Content: token,
				},
			},
		},
	}
}

// streamDeltaMsgOpenAi streams a delta message to the client.
func streamDeltaMsgOpenAi(ntokens int, token string, enc *json.Encoder, c echo.Context, params types.InferParams, startThinking time.Time, thinkingElapsed *time.Duration, startEmitting *time.Time) error {
	if ntokens == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		err := sendStartEmittingMessageOpenAi(enc, c, params, ntokens, *thinkingElapsed)
		if err != nil {
			fmt.Printf("Error emitting msg: %v\n", err)
			state.ContinueInferringController = false
			return err
		}
	}

	if !state.ContinueInferringController {
		return nil
	}

	if state.IsVerbose {
		fmt.Print(token)
	}

	if !params.Stream {
		return nil
	}

	tmsg := createOpenAiDeltaMessage(ntokens, token)

	err := streamOpenAiMsg(tmsg, c, enc)
	if err != nil {
		fmt.Printf("Error streaming delta message: %v\n", err)
		return err
	}

	return nil
}

// sendStartEmittingMessageOpenAi sends the start_emitting message to the client.
func sendStartEmittingMessageOpenAi(enc *json.Encoder, c echo.Context, params types.InferParams, ntokens int, thinkingElapsed time.Duration) error {
	if !params.Stream || !state.ContinueInferringController {
		return nil
	}

	// Create a system message similar to the one in infer.go but adapted for OpenAI format
	smsg := openAiChatCompletionDeltaResponse{
		ID:      strconv.Itoa(ntokens),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   state.LoadedModel,
		Choices: []deltaChoice{
			{
				Index:        ntokens,
				FinishReason: "",
				Delta: delta{
					Role:    "system",
					Content: "start_emitting",
				},
			},
		},
	}

	err := streamOpenAiMsg(smsg, c, enc)
	if err != nil {
		fmt.Printf("Error streaming start_emitting message: %v\n", err)
	}
	time.Sleep(2 * time.Millisecond) // Give some time to stream this message
	return err
}

// Utility Functions

// createErrorMessageOpenAi creates an InferenceError for OpenAI inference.
func createErrorMessageOpenAi(ntokens int, content string, context any, errorCode string) *InferError {
	return &InferError{
		Code:       errorCode,
		Message:    content,
		Context:    context,
		Timestamp:  time.Now(),
		TokenCount: ntokens,
	}
}

// logOpenAiVerboseInfo logs verbose information about the OpenAI inference process.
func logOpenAiVerboseInfo(finalPrompt string, thinkingElapsed time.Duration, emittingElapsed time.Duration, ntokens int) {
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

// Result Creation Functions

// createOpenAiResult creates the final OpenAI result.
func createOpenAiResult(ntokens int, res string) OpenAiChatCompletion {
	id := strconv.Itoa(ntokens)
	return OpenAiChatCompletion{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   state.LoadedModel,
		Choices: []openAiChoice{
			{
				Index: 0,
				Message: openAiMessage{
					Role:    "assistant",
					Content: res,
				},
				FinishReason: "stop",
			},
		},
		Usage: openAiUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      ntokens,
		},
	}
}
