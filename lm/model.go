package lm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
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
func LoadModel(modelConf types.ModelConf) (int, error) {
	if modelConf.Name == "" {
		return 400, fmt.Errorf("model name cannot be empty: %w", ErrInvalidInput)
	}

	mpath := filepath.Join(state.ModelsDir, modelConf.Name)
	// check if the model file exists
	_, err := os.Stat(mpath)
	if err != nil {
		if os.IsNotExist(err) {
			return 404, fmt.Errorf("the model file %s does not exist: %w", mpath, ErrModelNotFound)
		}
		return 500, fmt.Errorf("error checking model file %s: %w", mpath, err)
	}
	// check if the model is already loaded
	if state.LoadedModel == modelConf.Name {
		return 202, fmt.Errorf("the model is already loaded: %w", ErrInvalidInput)
	}
	if state.IsModelLoaded {
		UnloadModel()
	}

	lm, err := llama.New(
		mpath,
		llama.SetContext(modelConf.Ctx),
		llama.EnableEmbeddings,
		llama.SetGPULayers(99), // TODO modelConf.NGPULayers
	)
	if err != nil {
		return 500, fmt.Errorf("cannot load model %s: %w", modelConf.Name, ErrModelLoadFailed)
	}

	if state.IsVerbose || state.IsDebug {
		fmt.Println("Loaded model", mpath)
		if state.IsDebug {
			jsonData, err := json.MarshalIndent(modelConf, "", "  ")
			if err != nil {
				return 500, fmt.Errorf("error marshalling model params: %w", err)
			}
			fmt.Println(string(jsonData))
		}
	}

	state.Lm = lm
	state.ModelConf = modelConf
	state.IsModelLoaded = true
	state.LoadedModel = modelConf.Name

	return 200, nil
}

// Standard application errors.
var (
	ErrModelNotFound      = errors.New("model not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrModelLoadFailed    = errors.New("failed to load model")
	ErrTemplateParseError = errors.New("template parsing error")
)
