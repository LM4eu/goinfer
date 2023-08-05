package server

import (
	"errors"
	"fmt"
	"net/http"

	llama "github.com/go-skynet/go-llama.cpp"
	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/files"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
)

func parseModelParams(m echo.Map) (string, llama.ModelOptions, error) {
	var model string
	v, ok := m["model"]
	if !ok {
		return "", llama.ModelOptions{}, errors.New("provide a model name")
	}
	model = v.(string)
	ctx := lm.DefaultModelParams.ContextSize
	v, ok = m["ctx"]
	if ok {
		ctx = int(v.(float64))
	}
	embeddings := lm.DefaultModelParams.Embeddings
	v, ok = m["embeddings"]
	if ok {
		embeddings = v.(bool)
	}
	gpuLayers := lm.DefaultModelParams.NGPULayers
	v, ok = m["gpuLayers"]
	if ok {
		gpuLayers = v.(int)
	}
	params := llama.ModelOptions{
		ContextSize: ctx,
		Embeddings:  embeddings,
		NGPULayers:  gpuLayers,
	}
	return model, params, nil
}

func LoadModelHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	model, params, err := parseModelParams(m)
	if err != nil {
		fmt.Println(("error in params:" + err.Error()))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "model params",
		})
	}
	lm.LoadModel(model, params)
	return c.NoContent(http.StatusNoContent)
}

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
	return c.JSON(http.StatusOK, echo.Map{
		"models":        models,
		"isModelLoaded": state.IsModelLoaded,
		"loadedModel":   state.LoadedModel,
		"ctx":           state.ModelConf.Ctx,
	})
}
