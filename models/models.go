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

func (dir Dir) Str() string {
	return string(dir)
}

// StateHandler returns the state of models.
func (dir Dir) StateHandler(c echo.Context) error {
	models, err := dir.Search()
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

func (dir Dir) Search() ([]string, error) {
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
