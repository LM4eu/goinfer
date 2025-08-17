package server

import (
	"errors"
	"fmt"
	"net/http"

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

type ModelsDir string

func (dir ModelsDir) Str() string {
	return string(dir)
}

// ModelsStateHandler returns the state of models.
func (dir ModelsDir) ModelsStateHandler(c echo.Context) error {
	if state.Verbose {
		fmt.Println("Reading files in:", dir)
	}

	modelsInfo := map[string]any{"jsonrpc": "2.0", "id": 1}

	var statusCode int
	models, err := files.ReadModels(dir.Str())
	if err == nil {
		statusCode = http.StatusOK
		modelsInfo["result"] = models
		if state.Verbose {
			fmt.Println("Found models:", models)
		}
	} else {
		statusCode = http.StatusInternalServerError
		modelsInfo["error"] = "cannot fetch model files: " + err.Error()
		fmt.Println("Error while reading models:", err)
	}

	return c.JSON(statusCode, modelsInfo)
}
