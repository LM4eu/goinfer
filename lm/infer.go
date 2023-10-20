package lm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	llama "github.com/go-skynet/go-llama.cpp"
	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func StreamMsg(msg types.StreamedMessage, c echo.Context, enc *json.Encoder) error {
	c.Response().Write([]byte("data: "))
	if err := enc.Encode(msg); err != nil {
		return err
	}
	c.Response().Write([]byte("\n"))
	c.Response().Flush()
	return nil
}

func Infer(
	prompt string,
	template string,
	params types.InferenceParams,
	c echo.Context,
	ch chan<- types.StreamedMessage,
	errCh chan<- types.StreamedMessage,
) {
	if !state.IsModelLoaded {
		errCh <- types.StreamedMessage{
			Num:     1,
			Content: "no model loaded",
			MsgType: types.ErrorMsgType,
		}
		return
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
	if state.IsDebug {
		fmt.Println("Inference params:")
		fmt.Printf("%+v\n\n", params)

	}
	startThinking := time.Now()
	startEmitting := time.Now()
	var thinkingElapsed time.Duration
	ntokens := 0
	enc := json.NewEncoder(c.Response())
	res, err := state.Lm.Predict(finalPrompt, llama.SetTokenCallback(func(token string) bool {
		if ntokens == 0 {
			startEmitting = time.Now()
			thinkingElapsed = time.Since(startThinking)
			if state.IsVerbose {
				fmt.Println("Thinking time:", thinkingElapsed)
				fmt.Println("Emitting ..")
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
			if params.Stream && state.ContinueInferingController {
				StreamMsg(smsg, c, enc)
				// sleep to let the time to stream this message, as a second
				// message with the token has to be streamed in this loop as well
				time.Sleep(2 * time.Millisecond)
			}
		}
		/*if state.IsVerbose && !params.Stream {
			fmt.Print(token)
		}*/
		for _, stopToken := range params.StopPrompts {
			s, _ := strconv.Unquote(stopToken)
			if token == s {
				return false
			}
		}
		if params.Stream {
			tmsg := types.StreamedMessage{
				Content: token,
				Num:     ntokens,
				MsgType: types.TokenMsgType,
			}
			if state.ContinueInferingController {
				StreamMsg(tmsg, c, enc)
			}
		}
		ntokens++
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
	)

	state.IsInfering = false
	if err != nil {
		errCh <- types.StreamedMessage{
			Num:     ntokens + 1,
			Content: "inference error",
			MsgType: types.ErrorMsgType,
		}
		return
	}
	if state.ContinueInferingController {
		// the inference was not aborted
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
		stats := types.InferenceStats{
			ThinkingTime:       thinkingElapsed.Seconds(),
			ThinkingTimeFormat: thinkingElapsed.String(),
			EmitTime:           emittingElapsed.Seconds(),
			EmitTimeFormat:     emittingElapsed.String(),
			TotalTime:          totalTime.Seconds(),
			TotalTimeFormat:    totalTime.String(),
			TokensPerSecond:    tps,
			TotalTokens:        ntokens,
		}
		result := types.InferenceResult{
			Text:  res,
			Stats: stats,
		}
		// result
		b, _ := json.Marshal(&result)
		var _res map[string]interface{}
		_ = json.Unmarshal(b, &_res)
		endmsg := types.StreamedMessage{
			Num:     ntokens + 1,
			Content: "result",
			MsgType: types.SystemMsgType,
			Data:    _res,
		}
		if params.Stream {
			StreamMsg(endmsg, c, enc)
		}
		ch <- endmsg
	}
	state.ContinueInferingController = true
}
