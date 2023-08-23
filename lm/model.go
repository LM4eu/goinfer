package lm

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	llama "github.com/go-skynet/go-llama.cpp"
	"github.com/synw/goinfer/state"
)

func LoadModel(model string, params llama.ModelOptions) error {
	name := model
	if !(strings.HasSuffix(name, ".bin")) {
		name = name + ".bin"
	}
	mpath := filepath.Join(state.ModelsDir, name)
	if state.IsVerbose {
		fmt.Println("Loading model", mpath)
	}
	if state.IsModelLoaded {
		state.Lm.Free()
	}
	lm, err := llama.New(
		mpath,
		llama.SetContext(params.ContextSize),
		llama.EnableEmbeddings,
		llama.SetGPULayers(params.NGPULayers),
	)
	if err != nil {
		return errors.New("can not load model " + model)
	}
	state.Lm = lm
	state.ModelConf.Ctx = params.ContextSize
	state.IsModelLoaded = true
	state.LoadedModel = model
	return nil
}
