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

// parseInferQuery parses inference parameters from echo.Map using simple type assertions.
func parseInferQuery(m echo.Map) (types.InferQuery, error) {
	var errs []error
	
	query := types.InferQuery{
		Prompt:      "",
		ModelParams: types.DefaultModelConf,
		InferParams: types.DefaultInferParams,
	}

	// Parse prompt (required)
	if v, ok := m["prompt"]; ok {
		if prompt, ok := v.(string); ok {
			query.Prompt = prompt
		} else {
			errs = append(errs, errors.New("field prompt must be a string"))
		}
	} else {
		errs = append(errs, errors.New("missing mandatory field: prompt"))
	}

	// Parse model parameters
	if v, ok := m["model"]; ok {
		if name, ok := v.(string); ok {
			query.ModelParams.Name = name
		}
	}

	if v, ok := m["ctx"]; ok {
		if ctx, ok := v.(int); ok {
			query.ModelParams.Ctx = ctx
		}
	}

	// Parse inference parameters
	if v, ok := m["stream"]; ok {
		if stream, ok := v.(bool); ok {
			query.InferParams.Stream = stream
		}
	}

	if v, ok := m["temperature"]; ok {
		if temp, ok := v.(float64); ok {
			query.InferParams.Temperature = float32(temp)
		}
	}

	if v, ok := m["min_p"]; ok {
		if minP, ok := v.(float64); ok {
			query.InferParams.MinP = float32(minP)
		}
	}

	if v, ok := m["top_p"]; ok {
		if topP, ok := v.(float64); ok {
			query.InferParams.TopP = float32(topP)
		}
	}

	if v, ok := m["top_k"]; ok {
		if topK, ok := v.(int); ok {
			query.InferParams.TopK = topK
		}
	}

	if v, ok := m["max_tokens"]; ok {
		if maxTokens, ok := v.(int); ok {
			query.InferParams.MaxTokens = maxTokens
		}
	}

	if v, ok := m["presence_penalty"]; ok {
		if penalty, ok := v.(float64); ok {
			query.InferParams.PresencePenalty = float32(penalty)
		}
	}

	if v, ok := m["frequency_penalty"]; ok {
		if penalty, ok := v.(float64); ok {
			query.InferParams.FrequencyPenalty = float32(penalty)
		}
	}

	if v, ok := m["repeat_penalty"]; ok {
		if penalty, ok := v.(float64); ok {
			query.InferParams.RepeatPenalty = float32(penalty)
		}
	}

	if v, ok := m["tfs"]; ok {
		if tfs, ok := v.(float64); ok {
			query.InferParams.TailFreeSamplingZ = float32(tfs)
		}
	}

	// Parse stop prompts (special case for slice)
	if v, ok := m["stop"]; ok {
		if stopSlice, ok := v.([]any); ok {
			if len(stopSlice) > 0 {
				query.InferParams.StopPrompts = make([]string, len(stopSlice))
				for i, val := range stopSlice {
					query.InferParams.StopPrompts[i] = fmt.Sprint(val)
				}
			}
		} else {
			errs = append(errs, errors.New("field stop must be an array"))
		}
	}

	// Parse images (special case for byte array)
	if v, ok := m["images"]; ok {
		if slice, ok := v.([]any); ok {
			if len(slice) > 0 {
				query.InferParams.Images = make([]byte, len(slice))
				for i, val := range slice {
					if byteVal, ok := val.(byte); ok {
						query.InferParams.Images[i] = byteVal
					} else {
						errs = append(errs, fmt.Errorf("invalid byte value in images array at index %d", i))
					}
				}
			}
		} else {
			errs = append(errs, errors.New("field images must be an array"))
		}
	}

	// Parse audios (special case for byte array)
	if v, ok := m["audios"]; ok {
		if slice, ok := v.([]any); ok {
			if len(slice) > 0 {
				query.InferParams.Audios = make([]byte, len(slice))
				for i, val := range slice {
					if byteVal, ok := val.(byte); ok {
						query.InferParams.Audios[i] = byteVal
					} else {
						errs = append(errs, fmt.Errorf("invalid byte value in audios array at index %d", i))
					}
				}
			}
		} else {
			errs = append(errs, errors.New("field audios must be an array"))
		}
	}

	// If there are any errors, return them all joined
	if len(errs) > 0 {
		return query, errors.Join(errs...)
	}

	return query, nil
}

// InferHandler handles inference requests.
func InferHandler(c echo.Context) error {
	if state.IsInferring {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}

	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		if state.Debug {
			fmt.Println("Inference params decoding error", err)
		}
		return c.NoContent(http.StatusBadRequest)
	}

	query, err := parseInferQuery(m)
	if err != nil {
		if state.Debug {
			fmt.Println("Inference params parsing error", err)
		}
		return c.NoContent(http.StatusBadRequest)
	}

	// // Do we need to start/restart llama-server?
	// if state.IsStartNeeded(query.ModelParams) {
	// 	err := state.RestartLlamaServer(query.ModelParams)
	// 	if err != nil {
	// 		if state.IsDebug {
	// 			fmt.Println("Error loading model:", err)
	// 		}
	// 		return c.JSON(
	// 			http.StatusInternalServerError,
	// 			echo.Map{"error": "failed to load model" + err.Error()},
	// 		)
	// 	}
	// }

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
			if state.Verbose {
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
				err := lm.StreamMsg(&err, c, enc)
				if err != nil {
					if state.Debug {
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

// AbortLlamaHandler aborts ongoing inference.
func AbortLlamaHandler(c echo.Context) error {
	if !state.IsInferring {
		fmt.Println("No inference running, nothing to abort")
		return c.NoContent(http.StatusAccepted)
	}
	if state.Verbose {
		fmt.Println("Aborting inference")
	}
	state.ContinueInferringController = false
	return c.NoContent(http.StatusNoContent)
}
