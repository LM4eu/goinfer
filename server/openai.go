package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/files"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// openAiModel is used in the HTTP response body
type openAiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func parseParams(m echo.Map) (types.InferQuery, error) {
	query := types.InferQuery{
		Prompt:      "",
		ModelParams: types.DefaultModelConf,
		InferParams: types.DefaultInferParams,
	}

	v, ok := m["prompt"]
	if !ok {
		return query, errors.New("missing mandatory field: prompt")
	}
	query.Prompt = v.(string)

	v, ok = m["model"]
	if !ok {
		return query, errors.New("missing mandatory field: model")
	}
	query.ModelParams.Name = v.(string)

	v, ok = m["stream"]
	if ok {
		query.InferParams.Stream = v.(bool)
	}

	v, ok = m["temperature"]
	if ok {
		query.InferParams.Temperature = float32(v.(float64))
	}

	v, ok = m["min_p"]
	if ok {
		query.InferParams.MinP = float32(v.(float64))
	}

	v, ok = m["top_p"]
	if ok {
		query.InferParams.TopP = float32(v.(float64))
	}

	v, ok = m["top_k"]
	if ok {
		query.InferParams.TopK = int(v.(float64))
	}

	v, ok = m["max_tokens"]
	if ok {
		query.InferParams.MaxTokens = int(v.(float64))
	}

	v, ok = m["stop"]
	if ok {
		st := v.([]any)
		stf := []string{}
		for _, s := range st {
			stf = append(stf, s.(string))
		}
		query.InferParams.StopPrompts = stf
	}

	v, ok = m["presence_penalty"]
	if ok {
		query.InferParams.PresencePenalty = float32(v.(float64))
	}

	v, ok = m["frequency_penalty"]
	if ok {
		query.InferParams.FrequencyPenalty = float32(v.(float64))
	}

	v, ok = m["repeat_penalty"]
	if ok {
		query.InferParams.RepeatPenalty = float32(v.(float64))
	}

	v, ok = m["tfs"]
	if ok {
		query.InferParams.TailFreeSamplingZ = float32(v.(float64))
	}

	return query, nil
}

// Create an OpenAI api for /v1/chat/completions.
func CreateCompletionHandler(c echo.Context) error {
	if state.IsInferring {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}

	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	query, err := parseParams(m)
	if err != nil {
		panic(err)
	}

	// Do we need to start/restart llama-server?
	if state.IsStartNeeded(query.ModelParams) {
		err := state.RestartLlamaServer(query.ModelParams)
		if err != nil {
			if state.IsDebug {
				fmt.Println("Error loading model:", err)
			}
			return c.JSON(
				http.StatusInternalServerError,
				echo.Map{"error": "failed to load model" + err.Error()},
			)
		}
	}

	if query.InferParams.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}

	ctx := c.Request().Context()
	if ctx.Err() != nil { // If context has an error (e.g., canceled), stop processing
		fmt.Println("Context error")
		return c.NoContent(http.StatusNoContent)
	}

	ch := make(chan lm.OpenAiChatCompletion)
	errCh := make(chan error)

	defer close(ch)
	defer close(errCh)

	go lm.InferOpenAi(query, c, ch, errCh)

	select {
	case res, ok := <-ch:
		if ok {
			return c.JSON(http.StatusOK, res)
		}
		return nil
	case err, ok := <-errCh:
		if ok {
			fmt.Println("ERR", err)
			// panic(err)
		}
		return nil
	case <-c.Request().Context().Done():
		fmt.Println("\nRequest canceled")
		state.ContinueInferringController = false
		return c.NoContent(http.StatusNoContent)
	}
}

func OpenAiListModels(c echo.Context) error {
	if state.IsVerbose {
		fmt.Println("Reading files in:", state.ModelsDir)
	}

	models, err := files.ReadModels(state.ModelsDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "reading models",
		})
	}

	if state.IsVerbose {
		fmt.Println("Found models:", models)
	}

	endmodels := []openAiModel{}
	for _, m := range models {
		endmodels = append(endmodels,
			openAiModel{
				ID:      m,
				Object:  "model",
				Created: time.Now().Unix(),
				OwnedBy: "",
			},
		)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"object": "list",
		"data":   endmodels,
	})
}
