package lm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// OpenAI response structures
type OpenAiChoice struct {
	Index        int           `json:"index"`
	Message      OpenAiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAiChatCompletion struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAiChoice `json:"choices"`
	Usage   OpenAiUsage    `json:"usage"`
}

type OpenAiDelta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAiDeltaChoice struct {
	Delta        OpenAiDelta `json:"delta"`
	Index        int         `json:"index"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

type OpenAiChatCompletionDeltaResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenAiDeltaChoice `json:"choices"`
}

// Main Inference Functions

// InferOpenAi performs OpenAI model inference.
func InferOpenAi(query types.InferQuery, c echo.Context, ch chan<- OpenAiChatCompletion, errCh chan<- error) {

	/*
		if !state.Llama.IsRunning() {
			errCh <- createErrorMessageOpenAi(1, "no model loaded", nil, ErrCodeModelNotLoaded)
			return
		}

		// LogVerboseInfo("OpenAI", &stats, query.Prompt)

		if state.IsDebug {
			fmt.Println("Inference query:")
			fmt.Printf("%+v\n\n", query)
		}

		ntokens := 0
		enc := json.NewEncoder(c.Response())

		startThinking := time.Now()
		var thinkingElapsed time.Duration
		var startEmitting time.Time

		state.IsInferring = true
		state.ContinueInferringController = true

		res, err := state.Llama.Predict(
			query,
			func(token string) bool {
				err := streamDeltaMsgOpenAi(ntokens, token, enc, c, query, startThinking, &thinkingElapsed, &startEmitting)
				if err != nil {
					errCh <- createErrorMessageOpenAi(ntokens+1, "streamDeltaMsgOpenAi error", err, ErrStreamDeltaMsgOpenAi)
				}
				return state.ContinueInferringController
			})

		state.IsInferring = false

		if err != nil {
			state.ContinueInferringController = false
			errCh <- createErrorMessageOpenAi(ntokens+1, "inference error", err, ErrCodeInferenceFailed)
		}

		if !state.ContinueInferringController {
			return
		}

		if query.InferParams.Stream {
			err := sendOpenAiStreamTermination(c)
			if err != nil {
				state.ContinueInferringController = false
				errCh <- createErrorMessageOpenAi(ntokens+1, "cannot send stream termination", err, ErrCodeStreamFailed)
				LogError("OpenAI", "cannot send stream termination", err)
			}
		}

		if state.ContinueInferringController {
			ch <- createOpenAiResult(query, ntokens, res)
		}
	*/
}

// Streaming Functions


// streamDeltaMsgOpenAi streams a delta message to the client.
func streamDeltaMsgOpenAi(ntokens int, token string, enc *json.Encoder, c echo.Context, query types.InferQuery, startThinking time.Time, thinkingElapsed *time.Duration, startEmitting *time.Time) error {
	if ntokens == 0 {
		*startEmitting = time.Now()
		*thinkingElapsed = time.Since(startThinking)

		err := sendStartEmittingMessageOpenAi(enc, c, query, ntokens, *thinkingElapsed)
		if err != nil {
			LogError("OpenAI", "cannot emit start message", err)
			state.ContinueInferringController = false
			return err
		}
	}

	if !state.ContinueInferringController {
		return nil
	}

	LogToken(token)

	if !query.InferParams.Stream {
		return nil
	}

	tmsg := createOpenAiDeltaMessage(query, ntokens, token)

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

// createOpenAiDeltaMessage creates a delta message for streaming.
func createOpenAiDeltaMessage(query types.InferQuery, ntokens int, token string) OpenAiChatCompletionDeltaResponse {
	return OpenAiChatCompletionDeltaResponse{
		ID:      strconv.Itoa(ntokens),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   query.ModelParams.Name,
		Choices: []OpenAiDeltaChoice{
			{
				Index:        ntokens,
				FinishReason: "",
				Delta: OpenAiDelta{
					Role:    "assistant",
					Content: token,
				},
			},
		},
	}
}

// sendStartEmittingMessageOpenAi sends the start_emitting message to the client.
func sendStartEmittingMessageOpenAi(enc *json.Encoder, c echo.Context, query types.InferQuery, ntokens int, thinkingElapsed time.Duration) error {
	if !query.InferParams.Stream || !state.ContinueInferringController {
		return nil
	}

	// Create a system message similar to the one in infer.go but adapted for OpenAI format
	smsg := OpenAiChatCompletionDeltaResponse{
		ID:      strconv.Itoa(ntokens),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   query.ModelParams.Name,
		Choices: []OpenAiDeltaChoice{
			{
				Index:        ntokens,
				FinishReason: "",
				Delta: OpenAiDelta{
					Role:    "system",
					Content: "start_emitting",
				},
			},
		},
	}

	_, err := c.Response().Write([]byte("data: "))
	if err != nil {
		return fmt.Errorf("failed to write stream begin: %w", err)
	}

	err = enc.Encode(smsg)
	if err != nil {
		return fmt.Errorf("failed to encode stream message: %w", err)
	}

	_, err = c.Response().Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write stream message: %w", err)
	}

	c.Response().Flush()
	time.Sleep(2 * time.Millisecond) // Give some time to stream this message
	return err
}

// Utility Functions

// createErrorMessageOpenAi creates an InferenceError for OpenAI inference.
func createErrorMessageOpenAi(ntokens int, content string, context any, errorCode string) *InferError {
	return &InferError{
		Code:    errorCode,
		Message: content,
		Context: context,
	}
}



// Result Creation Functions

// createOpenAiResult creates the final OpenAI result.
func createOpenAiResult(query types.InferQuery, ntokens int, res string) OpenAiChatCompletion {
	id := strconv.Itoa(ntokens)
	return OpenAiChatCompletion{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   query.ModelParams.Name,
		Choices: []OpenAiChoice{
			{
				Index: 0,
				Message: OpenAiMessage{
					Role:    "assistant",
					Content: res,
				},
				FinishReason: "stop",
			},
		},
		Usage: OpenAiUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      ntokens,
		},
	}
}
