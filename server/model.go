package server

import (
	"errors"

	"github.com/labstack/echo/v4"
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
