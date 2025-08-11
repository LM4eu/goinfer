package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/files"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// parseModelParams parses model parameters from echo.Map.
func parseModelParams(m echo.Map) (types.ModelParams, error) {
	modelConf := types.DefaultModelConf

	name, ok := m["model"]
	if !ok {
		return types.ModelParams{}, errors.New("missing mandatory field: name")
	}

	// Type assertion with error checking
	modelConf.Name, ok = name.(string)
	if !ok {
		return types.ModelParams{}, errors.New("model name must be a string")
	}

	v, ok := m["ctx"]
	if ok {
		if ctxVal, ok := v.(float64); ok {
			modelConf.Ctx = int(ctxVal)
		}
	}

	return modelConf, nil
}

// StartLlamaHandler handles loading a model.
func StartLlamaHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return fmt.Errorf("failed to bind model parameters: %w", err)
	}

	modelConf, err := parseModelParams(m)
	if err != nil {
		fmt.Println("error in params:" + err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "model params"})
	}

	err = state.RestartLlamaServer(modelConf)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "error starting llama-server " + err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// StopLlamaHandler unloads the currently loaded model.
func StopLlamaHandler(c echo.Context) error {
	err := state.StopLlamaServer()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// ModelsStateHandler returns the state of models.
func ModelsStateHandler(c echo.Context) error {
	count, exe, args, running, uptime := state.GetServerStatus()

	llamaInfo := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]any{
			"start_count": count,
			"exe":         exe,
			"args":        args,
			"running":     running,
			"uptime":      fmt.Sprintf("duration: %s", uptime.Round(time.Second)),
			"conf":        state.Llama.Conf,
		},
	}

	if state.IsVerbose {
		fmt.Println("Reading files in:", state.ModelsDir)
	}

	modelsInfo := map[string]any{"jsonrpc": "2.0", "id": 2}

	var statusCode int
	models, err := files.ReadModels(state.ModelsDir)
	if err == nil {
		statusCode = http.StatusOK
		modelsInfo["result"] = models
		if state.IsVerbose {
			fmt.Println("Found models:", models)
		}
	} else {
		statusCode = http.StatusInternalServerError
		modelsInfo["error"] = "cannot fetch model files: " + err.Error()
		fmt.Println("Error while reading models:", err)
	}

	return c.JSON(statusCode, []any{llamaInfo, modelsInfo})
}
