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
	query := types.InferQuery{
		Prompt:      "",
		ModelParams: types.DefaultModelConf,
		InferParams: types.DefaultInferParams,
	}

	v, ok := m["prompt"]
	if !ok {
		return query, errors.New("missing mandatory field: prompt")
	}
	query.Prompt, ok = v.(string)
	if !ok {
		return query, errors.New("field prompt must be a string")
	}

	// Simplify by flattening the "model" sub-struct
	//
	//	modelConfRaw, ok := m["model"]
	//	if ok {
	//		if modelMap, ok := modelConfRaw.(map[string]any); ok {
	//			for k, v := range modelMap {
	//				switch k {
	//				case "name":
	//					if name, ok := v.(string); ok {
	//						query.ModelConf.Name = name
	//					}
	//				case "ctx":
	//					if ctx, ok := v.(float64); ok {
	//						 = int(ctx)
	//					}
	//				}
	//			}
	//		}
	//	}

	v, ok = m["model"]
	if ok {
		query.ModelParams.Name = v.(string)
	}

	v, ok = m["ctx"]
	if ok {
		query.ModelParams.Ctx = v.(int)
	}

	v, ok = m["stream"]
	if ok {
		if s, ok := v.(bool); ok {
			query.InferParams.Stream = s
		}
	}

	v, ok = m["temperature"]
	if ok {
		if t, ok := v.(float64); ok {
			query.InferParams.Temperature = float32(t)
		}
	}

	v, ok = m["min_p"]
	if ok {
		if p, ok := v.(float64); ok {
			query.InferParams.MinP = float32(p)
		}
	}

	v, ok = m["top_p"]
	if ok {
		if p, ok := v.(float64); ok {
			query.InferParams.TopP = float32(p)
		}
	}

	v, ok = m["top_k"]
	if ok {
		if k, ok := v.(float64); ok {
			query.InferParams.TopK = int(k)
		}
	}

	v, ok = m["max_tokens"]
	if ok {
		if t, ok := v.(float64); ok {
			query.InferParams.MaxTokens = int(t)
		}
	}

	v, ok = m["stop"]
	if ok {
		if stopSlice, ok := v.([]any); ok {
			if len(stopSlice) > 0 {
				query.InferParams.StopPrompts = make([]string, len(stopSlice))
				for i, val := range stopSlice {
					query.InferParams.StopPrompts[i] = fmt.Sprint(val)
				}
			}
		}
	}

	v, ok = m["presence_penalty"]
	if ok {
		if pp, ok := v.(float64); ok {
			query.InferParams.PresencePenalty = float32(pp)
		}
	}

	v, ok = m["frequency_penalty"]
	if ok {
		if fp, ok := v.(float64); ok {
			query.InferParams.FrequencyPenalty = float32(fp)
		}
	}

	v, ok = m["repeat_penalty"]
	if ok {
		if rp, ok := v.(float64); ok {
			query.InferParams.RepeatPenalty = float32(rp)
		}
	}

	v, ok = m["tfs"]
	if ok {
		if t, ok := v.(float64); ok {
			query.InferParams.TailFreeSamplingZ = float32(t)
		}
	}

	v, ok = m["images"]
	if ok {
		if slice, ok := v.([]any); ok {
			if len(slice) > 0 {
				query.InferParams.Images = make([]byte, len(slice))
				for i, val := range slice {
					query.InferParams.Images[i] = val.(byte)
				}
			}
		}
	}

	v, ok = m["audios"]
	if ok {
		if slice, ok := v.([]any); ok {
			if len(slice) > 0 {
				query.InferParams.Audios = make([]byte, len(slice))
				for i, val := range slice {
					query.InferParams.Audios[i] = val.(byte)
				}
			}
		}
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

// AbortLlamaHandler aborts ongoing inference.
func AbortLlamaHandler(c echo.Context) error {
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
