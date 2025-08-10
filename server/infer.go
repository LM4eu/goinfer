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

// parseInferQuery parses inference parameters from echo.Map.
func parseInferQuery(m echo.Map) (types.InferQuery, error) {
	v, ok := m["prompt"]
	if !ok {
		return types.InferQuery{}, errors.New("missing mandatory field: prompt")
	}
	prompt, ok := v.(string)
	if !ok {
		return types.InferQuery{}, errors.New("prompt must be a string")
	}

	modelConf := state.DefaultModelConf
	modelConfRaw, ok := m["model"]
	if ok {
		if modelMap, ok := modelConfRaw.(map[string]any); ok {
			for k, v := range modelMap {
				switch k {
				case "name":
					if name, ok := v.(string); ok {
						modelConf.Name = name
					}
				case "ctx":
					if ctx, ok := v.(float64); ok {
						modelConf.Ctx = int(ctx)
					}
				}
			}
		}
	}

	stream := state.DefaultInferenceParams.Stream
	v, ok = m["stream"]
	if ok {
		if s, ok := v.(bool); ok {
			stream = s
		}
	}

	tokens := state.DefaultInferenceParams.MaxTokens
	v, ok = m["max_tokens"]
	if ok {
		if t, ok := v.(float64); ok {
			tokens = int(t)
		}
	}

	topK := state.DefaultInferenceParams.TopK
	v, ok = m["top_k"]
	if ok {
		if k, ok := v.(float64); ok {
			topK = int(k)
		}
	}

	topP := state.DefaultInferenceParams.TopP
	v, ok = m["top_p"]
	if ok {
		if p, ok := v.(float64); ok {
			topP = float32(p)
		}
	}

	minP := state.DefaultInferenceParams.MinP
	v, ok = m["min_p"]
	if ok {
		if p, ok := v.(float64); ok {
			minP = float32(p)
		}
	}

	temp := state.DefaultInferenceParams.Temperature
	v, ok = m["temperature"]
	if ok {
		if t, ok := v.(float64); ok {
			temp = float32(t)
		}
	}

	freqPenalty := state.DefaultInferenceParams.FrequencyPenalty
	v, ok = m["frequency_penalty"]
	if ok {
		if fp, ok := v.(float64); ok {
			freqPenalty = float32(fp)
		}
	}

	presPenalty := state.DefaultInferenceParams.PresencePenalty
	v, ok = m["presence_penalty"]
	if ok {
		if pp, ok := v.(float64); ok {
			presPenalty = float32(pp)
		}
	}

	repeatPenalty := state.DefaultInferenceParams.RepeatPenalty
	v, ok = m["repeat_penalty"]
	if ok {
		if rp, ok := v.(float64); ok {
			repeatPenalty = float32(rp)
		}
	}

	tfs := state.DefaultInferenceParams.TailFreeSamplingZ
	v, ok = m["tfs"]
	if ok {
		if t, ok := v.(float64); ok {
			tfs = float32(t)
		}
	}

	stop := state.DefaultInferenceParams.StopPrompts
	v, ok = m["stop"]
	if ok {
		if stopSlice, ok := v.([]any); ok {
			if len(stopSlice) > 0 {
				stop = make([]string, len(stopSlice))
				for i, val := range stopSlice {
					stop[i] = fmt.Sprint(val)
				}
			}
		}
	}

	return types.InferQuery{
		Prompt:    prompt,
		ModelConf: modelConf,
		InferParams: types.InferenceParams{
			Stream:            stream,
			MaxTokens:         tokens,
			TopK:              topK,
			TopP:              topP,
			MinP:              minP,
			Temperature:       temp,
			FrequencyPenalty:  freqPenalty,
			PresencePenalty:   presPenalty,
			RepeatPenalty:     repeatPenalty,
			TailFreeSamplingZ: tfs,
			StopPrompts:       stop,
		}}, nil
}

// InferHandler handles inference requests.
func InferHandler(c echo.Context) error {
	if state.IsInferring {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}

	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		if state.IsDebug {
			fmt.Println("Inference params decoding error", err)
		}
		return c.NoContent(http.StatusBadRequest)
	}

	query, err := parseInferQuery(m)
	if err != nil {
		if state.IsDebug {
			fmt.Println("Inference params parsing error", err)
		}
		return c.NoContent(http.StatusBadRequest)
	}

	// Do we need to load the model?
	loadModel := true
	if state.IsModelLoaded {
		if state.LoadedModel == query.ModelConf.Name {
			if state.ModelConf.Ctx == query.ModelConf.Ctx {
				loadModel = false
			}
		}
	}

	if loadModel {
		state.ModelConf = query.ModelConf
		statusCode, err := lm.LoadModel(query.ModelConf)
		if err != nil {
			if state.IsDebug {
				fmt.Println("Error loading model:", err)
			}
			return c.JSON(statusCode, echo.Map{"error": fmt.Sprintf("failed to load model: %v", err)})
		}

		if state.IsDebug {
			fmt.Println("Loaded model with params:")
			jsonData, err := json.MarshalIndent(state.ModelConf, "", "  ")
			if err != nil {
				fmt.Println("Error:", err)
			}
			fmt.Println(string(jsonData))
		}
	}

	if query.InferParams.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}

	ch := make(chan types.StreamedMessage)
	errCh := make(chan types.StreamedMessage)

	defer close(ch)
	defer close(errCh)

	go lm.Infer(query, c, ch, errCh)

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
			if !query.InferParams.Stream {
				return c.JSON(http.StatusOK, res.Data)
			}
		}
		return nil
	case err, ok := <-errCh:
		if ok {
			if query.InferParams.Stream {
				enc := json.NewEncoder(c.Response())
				err := lm.StreamMsg(err, c, enc)
				if err != nil {
					if state.IsDebug {
						fmt.Println("Streaming error", err)
					}
					return c.JSON(http.StatusInternalServerError, echo.Map{"error": err})
				}
			} else {
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Content})
			}
		}
		return nil
	case <-c.Request().Context().Done():
		fmt.Println("\nRequest canceled")
		state.ContinueInferringController = false
		return c.NoContent(http.StatusNoContent)
	}
}

// AbortHandler aborts ongoing inference.
func AbortHandler(c echo.Context) error {
	if !state.IsInferring {
		fmt.Println("No inference running, nothing to abort")
		return c.NoContent(http.StatusAccepted)
	}
	if state.IsVerbose {
		fmt.Println("Aborting inference")
	}
	state.ContinueInferringController = false
	return c.NoContent(http.StatusNoContent)
}
