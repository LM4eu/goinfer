package lm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	llama "github.com/go-skynet/go-llama.cpp"
	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
	"github.com/synw/goinfer/ws"
)

func sse(token string, i int, c echo.Context, enc *json.Encoder) error {
	msg := types.StreamedMessage{
		Content: token,
		Num:     i,
		MsgType: types.TokenMsgType,
	}
	if err := enc.Encode(msg); err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}

func onToken(token string, i int, c echo.Context, enc *json.Encoder) {
	if state.IsVerbose {
		fmt.Print(token)
	}
	sse(token, i, c, enc)
	if state.UseWs {
		ws.SendToken(token, i)
	}
}

func Infer(prompt string, template string, params types.InferenceParams, c echo.Context) (types.InferenceResult, error) {
	if !state.IsModelLoaded {
		return types.InferenceResult{}, errors.New("load a model before infering")
	}
	state.IsInfering = true
	state.ContinueInferingController = true
	finalPrompt := strings.Replace(template, "{prompt}", prompt, 1)
	if state.IsVerbose {
		//fmt.Println("Inference params:")
		//fmt.Println(params)
		fmt.Println("---------- prompt ----------")
		fmt.Println(finalPrompt)
		fmt.Println("----------------------------")
		fmt.Println("Thinking ..")
	}
	startThinking := time.Now()
	startEmitting := time.Now()
	var thinkingElapsed time.Duration
	ntokens := 0
	enc := json.NewEncoder(c.Response())
	res, err := state.Lm.Predict(finalPrompt, llama.Debug, llama.SetTokenCallback(func(token string) bool {
		if ntokens == 0 {
			startEmitting = time.Now()
			thinkingElapsed = time.Since(startThinking)
			if state.IsVerbose {
				fmt.Println("Thinking time:", thinkingElapsed)
				fmt.Println("Emitting")
			}
		}
		onToken(token, ntokens, c, enc)
		ntokens++
		return state.ContinueInferingController
	}),
		llama.SetTokens(params.Tokens),
		llama.SetThreads(params.Threads),
		llama.SetTopK(params.TopK),
		llama.SetTopP(params.TopP),
		llama.SetTemperature(params.Temperature),
		llama.SetStopWords(params.StopPrompts),
		llama.SetFrequencyPenalty(params.FrequencyPenalty),
		llama.SetPresencePenalty(params.PresencePenalty),
		llama.SetPenalty(params.RepeatPenalty),
	)
	state.IsInfering = false
	if err != nil {
		return types.InferenceResult{}, err
	}
	emittingElapsed := time.Since(startEmitting)
	if state.IsVerbose {
		fmt.Println("Emitting time:", emittingElapsed)
	}
	tpsRaw := float64(ntokens) / emittingElapsed.Seconds()
	s := fmt.Sprintf("%.2f", tpsRaw)
	tps := 0.0
	if res, err := strconv.ParseFloat(s, 64); err == nil {
		tps = res
	}
	totalTime := thinkingElapsed + emittingElapsed
	if state.IsVerbose {
		fmt.Println("Total time:", totalTime)
		fmt.Println("Tokens per seconds", tps)
		fmt.Println("Tokens emitted", ntokens)
	}
	result := types.InferenceResult{
		Text:               res,
		ThinkingTime:       thinkingElapsed.Seconds(),
		ThinkingTimeFormat: thinkingElapsed.String(),
		EmitTime:           emittingElapsed.Seconds(),
		EmitTimeFormat:     emittingElapsed.String(),
		TotalTime:          totalTime.Seconds(),
		TotalTimeFormat:    totalTime.String(),
		TokensPerSecond:    tps,
		TotalTokens:        ntokens,
	}
	state.IsInfering = false
	state.ContinueInferingController = true
	return result, nil
}
