package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func parseParams(m echo.Map) (string, string, string, types.InferenceParams, error) {
	params := lm.DefaultInferenceParams
	v, ok := m["model"]
	if !ok {
		return "", "", "", params, errors.New("provide a model")
	}
	model := v.(string)
	v, ok = m["messages"]
	if !ok {
		return "", "", "", params, errors.New("provide a messages array")
	}
	qmsgs := v.([]interface{})
	prompt := ""
	template := state.OpenAiConf.Template
	for _, m := range qmsgs {
		el := m.(map[string]interface{})
		role := el["role"].(string)
		content := el["content"].(string)
		if role == "system" {
			template = strings.Replace(template, "{system}", content, 1)
		} else if role == "user" {
			prompt = content
		}
	}
	v, ok = m["stream"]
	if ok {
		params.Stream = v.(bool)
	}
	v, ok = m["temperature"]
	if ok {
		params.Temperature = float32(v.(float64))
	}
	v, ok = m["top_p"]
	if ok {
		params.TopP = float32(v.(float64))
	}
	v, ok = m["top_k"]
	if ok {
		params.TopK = int(v.(float64))
	}
	v, ok = m["max_tokens"]
	if ok {
		params.NPredict = int(v.(float64))
	}
	v, ok = m["stop"]
	if ok {
		params.StopPrompts = v.([]string)
	}
	v, ok = m["presence_penalty"]
	if ok {
		params.PresencePenalty = float32(v.(float64))
	}
	v, ok = m["frequency_penalty"]
	if ok {
		params.FrequencyPenalty = float32(v.(float64))
	}
	v, ok = m["repeat_penalty"]
	if ok {
		params.RepeatPenalty = float32(v.(float64))
	}
	v, ok = m["tfs_z"]
	if ok {
		params.TailFreeSamplingZ = float32(v.(float64))
	}
	return model, prompt, template, params, nil
}

// Create an Openai api for /v1/chat/completion
func CreateCompletionHandler(c echo.Context) error {
	if state.IsInfering {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	//fmt.Println("COMPLETION", m)
	model, prompt, template, params, err := parseParams(m)
	if err != nil {
		panic(err)
	}
	if state.LoadedModel != model {
		lm.LoadModel(model, lm.DefaultModelParams)
	}
	if params.Stream {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
	}
	ctx := c.Request().Context()
	if ctx.Err() != nil { // If context has an error (e.g., canceled), stop processing
		fmt.Println("Context error")
		return c.NoContent(http.StatusNoContent)
	}

	res, err := lm.InferOpenAi(prompt, template, params, c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	/*select {
	case <-notifier:
		fmt.Println("Notifier end")
		return nil
	case <-c.Request().Context().Done(): // Check context.
		fmt.Println("Request done end")
		// If it reaches here, this means that context was canceled (a timeout was reached, etc.).
		return c.JSON(http.StatusOK, res)
	}*/
	return c.JSON(http.StatusOK, res)
}
