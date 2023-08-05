package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func parseParams(m echo.Map) (string, string, types.InferenceParams, error) {
	v, ok := m["model"]
	if !ok {
		return "", "", types.InferenceParams{}, errors.New("provide a model")
	}
	model := v.(string)
	/* v, ok = m["messages"]
	if !ok {
		return "", "", types.InferenceParams{}, errors.New("provide a messages array")
	}*/
	//qmsgs := v.([]types.OpenAiMessage)
	params := types.InferenceParams{}
	return model, "", params, nil
}

// Create an Openai api for /v1/completion
func CreateCompletionHandler(c echo.Context) error {
	if state.IsInfering {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}
