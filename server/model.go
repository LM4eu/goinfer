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
func parseModelParams(m echo.Map) (types.ModelConf, error) {
	modelConf := types.DefaultModelConf

	name, ok := m["name"]
	if !ok {
		return types.ModelConf{}, errors.New("missing mandatory field: name")
	}

	// Type assertion with error checking
	modelConf.Name, ok = name.(string)
	if !ok {
		return types.ModelConf{}, errors.New("model name must be a string")
	}

	v, ok := m["ctx"]
	if ok {
		if ctxVal, ok := v.(float64); ok {
			modelConf.Ctx = int(ctxVal)
		}
	}

	return modelConf, nil
}

// LoadModelHandler handles loading a model.
func LoadModelHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return fmt.Errorf("failed to bind model parameters: %w", err)
	}

	modelConf, err := parseModelParams(m)
	if err != nil {
		fmt.Println("error in params:" + err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "model params"})
	}

	errcode, err := state.StartLlamaWithModel(modelConf)
	if err != nil {
		switch errcode {
		case 500:
			if state.IsDebug {
				panic(fmt.Errorf("debug - Error loading model: %w", err))
			}
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "error loading model"})
		case 404:
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		case 202:
			return c.JSON(http.StatusAccepted, echo.Map{"error": err.Error()})
		case 400:
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// UnloadModelHandler unloads the currently loaded model.
func UnloadModelHandler(c echo.Context) error {
	state.UnloadModel()
	return c.NoContent(http.StatusNoContent)
}

// ModelsStateHandler returns the state of models.
func ModelsStateHandler(c echo.Context) error {
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

	templates, err := files.ReadTemplates()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "reading templates",
		})
	}

	if state.IsVerbose {
		fmt.Println("Found templates:", templates)
	}

	for _, model := range models {
		_, hasTemplate := templates[model]
		if !hasTemplate {
			templates[model] = types.TemplateInfo{
				Name: "unknown",
				Ctx:  0,
			}
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"models":        templates,
		"isModelLoaded": state.IsModelLoaded,
		"loadedModel":   state.LoadedModel,
		"ctx":           state.ModelConf.Ctx,
	})
}
