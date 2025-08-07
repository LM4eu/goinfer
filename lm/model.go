package lm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/state"
)

// UnloadModel unloads the currently loaded model.
func UnloadModel() {
	if state.IsModelLoaded {
		state.Lm.Free()
	}
	state.IsModelLoaded = false
	state.LoadedModel = ""
}

// Returns error code and error if any.
func LoadModel(model string, params llama.ModelOptions) (int, error) {
	if model == "" {
		return 400, fmt.Errorf("model name cannot be empty: %w", ErrInvalidInput)
	}

	mpath := filepath.Join(state.ModelsDir, model)
	// check if the model file exists
	_, err := os.Stat(mpath)
	if err != nil {
		if os.IsNotExist(err) {
			return 404, fmt.Errorf("the model file %s does not exist: %w", mpath, ErrModelNotFound)
		}
		return 500, fmt.Errorf("error checking model file %s: %w", mpath, err)
	}
	// check if the model is already loaded
	if state.LoadedModel == model {
		return 202, fmt.Errorf("the model is already loaded: %w", ErrInvalidInput)
	}
	if state.IsModelLoaded {
		UnloadModel()
	}

	lm, err := llama.New(
		mpath,
		llama.SetContext(params.ContextSize),
		llama.EnableEmbeddings,
		llama.SetGPULayers(params.NGPULayers),
	)
	if err != nil {
		return 500, fmt.Errorf("cannot load model %s: %w", model, ErrModelLoadFailed)
	}

	if state.IsVerbose || state.IsDebug {
		fmt.Println("Loaded model", mpath)
		if state.IsDebug {
			jsonData, err := json.MarshalIndent(params, "", "  ")
			if err != nil {
				return 500, fmt.Errorf("error marshalling model params: %w", err)
			}
			fmt.Println(string(jsonData))
		}
	}

	state.Lm = lm
	state.ModelOptions = params
	state.IsModelLoaded = true
	state.LoadedModel = model

	return 200, nil
}

// Standard application errors.
var (
	ErrModelNotFound      = errors.New("model not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrModelLoadFailed    = errors.New("failed to load model")
	ErrTemplateParseError = errors.New("template parsing error")
)
