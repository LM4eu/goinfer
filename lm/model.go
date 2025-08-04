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

func UnloadModel() {
	if state.IsModelLoaded {
		state.Lm.Free()
	}
	state.IsModelLoaded = false
	state.LoadedModel = ""
}

func LoadModel(model string, params llama.ModelOptions) (int, error) {
	mpath := filepath.Join(state.ModelsDir, model)
	// check if the model file exists
	_, err := os.Stat(mpath)
	if err != nil {
		if os.IsNotExist(err) {
			return 404, errors.New("the model file " + mpath + " does not exist")
		}
	}
	// check if the model is already loaded
	if state.LoadedModel == model {
		return 202, errors.New("the model is already loaded")
	}
	if state.IsModelLoaded {
		UnloadModel()
	}
	//fmt.Println("MODEL PARAMS:")
	//fmt.Printf("%+v\n", params)
	lm, err := llama.New(
		mpath,
		llama.SetContext(params.ContextSize),
		llama.EnableEmbeddings,
		llama.SetGPULayers(params.NGPULayers),
	)
	if err != nil {
		return 500, errors.New("can not load model " + model)
	}
	if state.IsVerbose || state.IsDebug {
		fmt.Println("Loaded model", mpath)
		if state.IsDebug {
			jsonData, err := json.MarshalIndent(params, "", "  ")
			if err != nil {
				//fmt.Println("Error decoding json:", err)
				return 500, err
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
