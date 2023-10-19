package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func ParseInferParams(m echo.Map) (string, string, types.ModelConf, types.InferenceParams, error) {
	// fmt.Println("Params", m)
	v, ok := m["prompt"]
	if !ok {
		return "", "", types.ModelConf{}, types.InferenceParams{}, errors.New("provide a prompt")
	}
	prompt := v.(string)
	template := "{prompt}"
	v, ok = m["template"]
	if ok {
		template = v.(string)
	}
	modelConf := state.DefaultModelConf
	modelConfRaw, ok := m["model"]
	if ok {
		mp := modelConfRaw.(map[string]interface{})
		for k, v := range mp {
			if k == "name" {
				modelConf.Name = v.(string)
			} else if k == "ctx" {
				modelConf.Ctx = int(v.(float64))
			} else if k == "gpu_layers" {
				modelConf.GPULayers = int(v.(float64))
			}
		}
	}
	stream := state.DefaultInferenceParams.Stream
	v, ok = m["stream"]
	if ok {
		stream = v.(bool)
	}
	threads := state.DefaultInferenceParams.Threads
	v, ok = m["threads"]
	if ok {
		threads = int(v.(float64))
	}
	tokens := state.DefaultInferenceParams.NPredict
	v, ok = m["n_predict"]
	if ok {
		tokens = int(v.(float64))
	}
	topK := state.DefaultInferenceParams.TopK
	v, ok = m["top_k"]
	if ok {
		topK = int(v.(float64))
	}
	topP := state.DefaultInferenceParams.TopP
	v, ok = m["top_p"]
	if ok {
		topP = float32(v.(float64))
	}
	temp := state.DefaultInferenceParams.Temperature
	v, ok = m["temperature"]
	if ok {
		temp = float32(v.(float64))
	}
	freqPenalty := state.DefaultInferenceParams.FrequencyPenalty
	v, ok = m["frequency_penalty"]
	if ok {
		freqPenalty = float32(v.(float64))
	}
	presPenalty := state.DefaultInferenceParams.PresencePenalty
	v, ok = m["presence_penalty"]
	if ok {
		presPenalty = float32(v.(float64))
	}
	repeatPenalty := state.DefaultInferenceParams.RepeatPenalty
	v, ok = m["repeat_penalty"]
	if ok {
		repeatPenalty = float32(v.(float64))
	}
	tfs := state.DefaultInferenceParams.TailFreeSamplingZ
	v, ok = m["tfs_z"]
	if ok {
		tfs = float32(v.(float64))
	}
	stop := state.DefaultInferenceParams.StopPrompts
	v, ok = m["stop"]
	if ok {
		s := v.([]interface{})
		if len(s) > 0 {
			stop = make([]string, len(s))
			for i, val := range s {
				stop[i] = fmt.Sprint(val)
			}
		}
	}
	params := types.InferenceParams{
		Stream:            stream,
		Threads:           threads,
		NPredict:          tokens,
		TopK:              topK,
		TopP:              topP,
		Temperature:       temp,
		FrequencyPenalty:  freqPenalty,
		PresencePenalty:   presPenalty,
		RepeatPenalty:     repeatPenalty,
		TailFreeSamplingZ: tfs,
		StopPrompts:       stop,
	}
	return prompt, template, modelConf, params, nil
}

func setModelOptions(modelConf types.ModelConf) error {
	opts := state.DefaultModelOptions
	opts.ContextSize = modelConf.Ctx
	state.ModelOptions = opts
	return nil
}

func InferHandler(c echo.Context) error {
	if state.IsInfering {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		if state.IsDebug {
			fmt.Println("Inference params decoding error", err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}
	prompt, template, modelConf, params, err := ParseInferParams(m)
	if err != nil {
		if state.IsDebug {
			fmt.Println("Inference params parsing error", err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}
	if modelConf.Name != "" {
		setModelOptions(modelConf)
		lm.LoadModel(modelConf.Name, state.ModelOptions)
		if state.IsDebug {
			fmt.Println("Loaded model with params:")
			jsonData, err := json.MarshalIndent(state.ModelOptions, "", "  ")
			if err != nil {
				fmt.Println("Error:", err)
			}
			fmt.Println(string(jsonData))
		}
	}
	//fmt.Println("Params", params)
	if params.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}

	ch := make(chan types.StreamedMessage)
	errCh := make(chan types.StreamedMessage)
	defer close(ch)
	defer close(errCh)

	go lm.Infer(prompt, template, params, c, ch, errCh)

	select {
	case res, ok := <-ch:
		if ok {
			if state.IsVerbose {
				fmt.Println("-------- result ----------")
				for key, value := range res.Data {
					fmt.Printf("%s: %v\n", key, value)
				}
				fmt.Println("--------------------------")
			}
			if !params.Stream {
				return c.JSON(http.StatusOK, res.Data)
			}
		}
		return nil
	case err, ok := <-errCh:
		if ok {
			if params.Stream {
				enc := json.NewEncoder(c.Response())
				err := lm.StreamMsg(err, c, enc)
				if err != nil {
					if state.IsDebug {
						fmt.Println("Streaming error", err)
					}
					return c.NoContent(http.StatusInternalServerError)
				}
			} else {
				return c.JSON(http.StatusInternalServerError, err)
			}
		}
		return nil
	case <-c.Request().Context().Done():
		fmt.Println("\nRequest canceled")
		state.ContinueInferingController = false
		return c.NoContent(http.StatusNoContent)
	}
}

func AbortHandler(c echo.Context) error {
	if !state.IsInfering {
		fmt.Println("No inference running, nothing to abort")
		return c.NoContent(http.StatusAccepted)
	}
	if state.IsVerbose {
		fmt.Println("Aborting inference")
	}
	state.ContinueInferingController = false
	return c.NoContent(http.StatusNoContent)
}
