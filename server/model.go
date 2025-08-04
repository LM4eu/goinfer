package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/files"
	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// parseModelParams parses model parameters from echo.Map
func parseModelParams(m echo.Map) (string, llama.ModelOptions, error) {
	var model string
	v, ok := m["name"]
	if !ok {
		return "", llama.ModelOptions{}, errors.New("provide a model name")
	}
	
	// Type assertion with error checking
	modelName, ok := v.(string)
	if !ok {
		return "", llama.ModelOptions{}, errors.New("model name must be a string")
	}
	model = modelName
	
	ctx := state.DefaultModelOptions.ContextSize
	v, ok = m["ctx"]
	if ok {
		if ctxVal, ok := v.(float64); ok {
			ctx = int(ctxVal)
		}
	}
	
	embeddings := state.DefaultModelOptions.Embeddings
	v, ok = m["embeddings"]
	if ok {
		if e, ok := v.(bool); ok {
			embeddings = e
		}
	}
	
	gpuLayers := state.DefaultModelOptions.NGPULayers
	v, ok = m["gpu_layers"]
	if ok {
		if gpuLayersVal, ok := v.(float64); ok {
			gpuLayers = int(gpuLayersVal)
		}
	}
	
	params := llama.ModelOptions{
		ContextSize: ctx,
		Embeddings:  embeddings,
		NGPULayers:  gpuLayers,
	}
	
	return model, params, nil
}

// LoadModelHandler handles loading a model
func LoadModelHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return fmt.Errorf("failed to bind model parameters: %w", err)
	}
	
	model, params, err := parseModelParams(m)
	if err != nil {
		fmt.Println("error in params:" + err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "model params",
		})
	}
	
	errcode, err := lm.LoadModel(model, params)
	if err != nil {
		if errcode == 500 {
			if state.IsDebug {
				panic(fmt.Errorf("Debug - Error loading model: %w\n", err))
			}
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "error loading model",
			})
		} else if errcode == 404 {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": err.Error(),
			})
		} else if errcode == 202 {
			return c.JSON(http.StatusAccepted, echo.Map{
				"error": err.Error(),
			})
		} else if errcode == 400 {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": err.Error(),
			})
		}
	}
	
	return c.NoContent(http.StatusNoContent)
}

// UnloadModelHandler unloads the currently loaded model
func UnloadModelHandler(c echo.Context) error {
	lm.UnloadModel()
	return c.NoContent(http.StatusNoContent)
}

// ModelsStateHandler returns the state of models
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
		"ctx":           state.ModelOptions.ContextSize,
	})
}
