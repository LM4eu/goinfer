package lm

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	llama "github.com/go-skynet/go-llama.cpp"
	"github.com/synw/goinfer/state"
)

func UnloadModel() {
	if state.IsModelLoaded {
		state.Lm.Free()
	}
	state.IsModelLoaded = false
	state.LoadedModel = ""
}

func LoadModel(model string, params llama.ModelOptions) error {
	name := model
	mpath := filepath.Join(state.ModelsDir, name)
	UnloadModel()
	lm, err := llama.New(
		mpath,
		llama.SetContext(params.ContextSize),
		llama.EnableEmbeddings,
		llama.SetGPULayers(params.NGPULayers),
	)
	if err != nil {
		return errors.New("can not load model " + model)
	}
	if state.IsVerbose || state.IsDebug {
		fmt.Println("Loaded model", mpath)
		if state.IsDebug {
			jsonData, err := json.MarshalIndent(params, "", "  ")
			if err != nil {
				fmt.Println("Error:", err)
				return err
			}
			fmt.Println(string(jsonData))
		}
	}
	state.Lm = lm
	state.ModelOptions = params
	state.IsModelLoaded = true
	state.LoadedModel = model
	return nil
}
