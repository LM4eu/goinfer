package lm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-skynet/go-llama.cpp"
	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func streamOpenAiMsg(msg types.OpenAiChatCompletionDeltaResponse, c echo.Context, enc *json.Encoder) error {
	if err := enc.Encode(msg); err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}

func InferOpenAi(prompt string, template string, params types.InferenceParams, c echo.Context) (types.OpenAiChatCompletion, error) {
	state.IsInfering = true
	state.ContinueInferingController = true
	finalPrompt := strings.Replace(template, "{prompt}", prompt, 1)
	if state.IsVerbose {
		fmt.Println("---------- prompt ----------")
		fmt.Println(finalPrompt)
		fmt.Println("----------------------------")
		fmt.Println("Thinking ..")
	}
	enc := json.NewEncoder(c.Response())
	ntokens := 0
	res, err := state.Lm.Predict(finalPrompt, llama.Debug, llama.SetTokenCallback(func(token string) bool {
		if state.IsVerbose {
			fmt.Print(token)
		}
		if params.Stream {
			tmsg := types.OpenAiChatCompletionDeltaResponse{
				ID:      strconv.Itoa(ntokens),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   state.LoadedModel,
				Choices: []types.DeltaChoice{
					{
						Index:        ntokens,
						FinishReason: "",
						Delta: types.Delta{
							Role:    "assistant",
							Content: token,
						},
					},
				},
			}
			streamOpenAiMsg(tmsg, c, enc)
		}
		ntokens++
		return state.ContinueInferingController
	}),
		llama.SetTokens(params.NPredict),
		llama.SetThreads(params.Threads),
		llama.SetTopK(params.TopK),
		llama.SetTopP(params.TopP),
		llama.SetTemperature(params.Temperature),
		llama.SetStopWords(strings.Join(params.StopPrompts, ",")),
		llama.SetFrequencyPenalty(params.FrequencyPenalty),
		llama.SetPresencePenalty(params.PresencePenalty),
		llama.SetPenalty(params.RepeatPenalty),
	)
	state.IsInfering = false
	id := strconv.Itoa(ntokens)
	endres := types.OpenAiChatCompletion{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   state.LoadedModel,
		Choices: []types.OpenAiChoice{
			{
				Index: 0,
				Message: types.OpenAiMessage{
					Role:    "assistant",
					Content: res,
				},
				FinishReason: "stop",
			},
		},
		Usage: types.OpenAiUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      ntokens,
		},
	}
	if err != nil {
		return endres, err
	}
	return endres, nil
}
