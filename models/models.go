package models

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/state"
)

type Dir string

// Handler returns the state of models.
func (dir Dir) Handler(c echo.Context) error {
	models, err := dir.Search()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("failed to search models: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"models": models,
		"count":  len(models),
	})
}

func (dir Dir) Search() ([]string, error) {
	var modelFiles []string

	// dir = one or multiple directories separated by ':'
	directories := strings.Split(string(dir), ":")

	for _, d := range directories {
		err := appendModels(&modelFiles, strings.TrimSpace(d))
		if err != nil {
			return nil, fmt.Errorf("failed to search in '%s': %w", d, err)
		}
	}

	return modelFiles, nil
}

func appendModels(files *[]string, root string) error {
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
			*files = append(*files, path)
		}

		return nil
	})
}
