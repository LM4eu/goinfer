package server

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
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
	models, err := dir.SearchModels()
	if err != nil {
		fmt.Println("Error while reading models:", err)
		e := map[string]string{"error": "cannot fetch model files: " + err.Error()}
		return c.JSON(http.StatusInternalServerError, e)
	}

	if state.Verbose {
		fmt.Println("Found models:", models)
	}

	return c.JSON(http.StatusOK, models)
}

func (dir ModelsDir) SearchModels() ([]string, error) {
	var modelFiles []string
	// dir = one or multiple directories separated by ':'
	directories := strings.Split(dir.Str(), ":")
	for _, d := range directories {
		err := appendModels(modelFiles, strings.TrimSpace(d))
		if err != nil {
			return nil, err
		}
	}
	return modelFiles, nil
}

func appendModels(files []string, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if state.Verbose {
				fmt.Println("Searching model files in:", path)
			}
			return nil // => step into this directory
		}
		if strings.HasSuffix(path, ".gguf") {
			files = append(files, path)
		}
		return nil
	})
}
