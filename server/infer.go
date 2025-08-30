package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// inferHandler handles infer requests.
func inferHandler(c echo.Context) error {
	// Initialize context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Check if infer is already running
	if state.IsInferring {
		fmt.Println("Infer already running")
		return c.NoContent(http.StatusAccepted)
	}

	// Bind request parameters
	reqMap := echo.Map{}
	if err := c.Bind(&reqMap); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// Parse infer parameters directly
	query, err := parseInferQuery(reqMap)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid parameter values",
			"code":  "INVALID_PARAMS",
		})
	}

	// Setup streaming response if needed
	if query.Params.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}

	// Execute infer directly (no retry)
	result, err := execute(c, ctx, query)
	if err != nil {
		return err
	}

	// Handle the infer result
	if state.Verbose {
		fmt.Println("-------- result ----------")
		for key, value := range result.Data {
			fmt.Printf("%s: %v\n", key, value)
		}
		fmt.Println("--------------------------")
	}

	if !query.Params.Stream {
		return c.JSON(http.StatusOK, result.Data)
	}
	return nil
}

// parseInferQuery parses infer parameters from echo.Map directly.
func parseInferQuery(m echo.Map) (types.InferQuery, error) {
	req := types.InferQuery{
		Prompt: "",
		Model:  types.DefaultModel,
		Params: types.DefaultInferParams,
	}

	// Check required prompt parameter
	if _, ok := m["prompt"]; !ok {
		return req, errors.New("prompt is required")
	}

	// Parse simple parameters directly
	if val, ok := m["prompt"].(string); ok {
		req.Prompt = val
	} else {
		return req, errors.New("prompt must be a string")
	}

	if val, ok := m["model"].(string); ok {
		req.Model.Name = val
	}

	if val, ok := m["ctx"].(int); ok {
		req.Model.Ctx = val
	}

	if val, ok := m["stream"].(bool); ok {
		req.Params.Stream = val
	}

	if val, ok := m["temperature"].(float64); ok {
		req.Params.Sampling.Temperature = float32(val)
	}

	if val, ok := m["min_p"].(float64); ok {
		req.Params.Sampling.MinP = float32(val)
	}

	if val, ok := m["top_p"].(float64); ok {
		req.Params.Sampling.TopP = float32(val)
	}

	if val, ok := m["presence_penalty"].(float64); ok {
		req.Params.Sampling.PresencePenalty = float32(val)
	}

	if val, ok := m["frequency_penalty"].(float64); ok {
		req.Params.Sampling.FrequencyPenalty = float32(val)
	}

	if val, ok := m["repeat_penalty"].(float64); ok {
		req.Params.Sampling.RepeatPenalty = float32(val)
	}

	if val, ok := m["tfs"].(float64); ok {
		req.Params.Sampling.TailFreeSamplingZ = float32(val)
	}

	if val, ok := m["top_k"].(int); ok {
		req.Params.Sampling.TopK = val
	}

	if val, ok := m["max_tokens"].(int); ok {
		req.Params.Generation.MaxTokens = val
	}

	// Parse stop prompts array
	if v, ok := m["stop"]; ok {
		if slice, ok := v.([]any); ok {
			if len(slice) > 10 {
				return req, errors.New("stop array too large (max 10)")
			}
			if len(slice) > 0 {
				req.Params.Generation.StopPrompts = make([]string, len(slice))
				for i, val := range slice {
					if strVal, ok := val.(string); ok {
						req.Params.Generation.StopPrompts[i] = strVal
					} else {
						return req, fmt.Errorf("stop[%d] must be a string", i)
					}
				}
			}
		} else {
			return req, errors.New("stop must be an array")
		}
	}

	// Parse media byte arrays
	if v, ok := m["images"]; ok {
		if slice, ok := v.([]any); ok && len(slice) > 0 {
			req.Params.Media.Images = make([]byte, len(slice))
			for i, val := range slice {
				if byteVal, ok := val.(byte); ok {
					req.Params.Media.Images[i] = byteVal
				} else {
					return req, fmt.Errorf("images[%d] must be a byte", i)
				}
			}
		}
	}

	if v, ok := m["audios"]; ok {
		if slice, ok := v.([]any); ok && len(slice) > 0 {
			req.Params.Media.Audios = make([]byte, len(slice))
			for i, val := range slice {
				if byteVal, ok := val.(byte); ok {
					req.Params.Media.Audios[i] = byteVal
				} else {
					return req, fmt.Errorf("audios[%d] must be a byte", i)
				}
			}
		}
	}

	return req, nil
}

// execute executes inference directly.
func execute(c echo.Context, ctx context.Context, query types.InferQuery) (*types.StreamedMsg, error) {
	// Execute infer in goroutine with timeout
	inferCtx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	resultChan := make(chan types.StreamedMsg)
	errorChan := make(chan types.StreamedMsg)
	defer close(resultChan)
	defer close(errorChan)

	go lm.Infer(query, c, resultChan, errorChan)

	// Process response directly
	select {
	case response, ok := <-resultChan:
		if ok {
			return &response, nil
		}
		return nil, errors.New("infer channel closed unexpectedly")

	case message, ok := <-errorChan:
		if ok {
			if message.MsgType == types.ErrorMsgType {
				return nil, fmt.Errorf("infer error: %s", message.Content)
			}
			return nil, fmt.Errorf("infer error: %v", message)
		}
		return nil, errors.New("error channel closed unexpectedly")

	case <-inferCtx.Done():
		if state.Debug {
			fmt.Printf("Infer timeout\n")
		}
		return nil, errors.New("infer timeout")

	case <-c.Request().Context().Done():
		// Client canceled request
		state.ContinueInferringController = false
		return nil, errors.New("req canceled by client")
	}
}

// abortHandler aborts ongoing inference.
func abortHandler(c echo.Context) error {
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
