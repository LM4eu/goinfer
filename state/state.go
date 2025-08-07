package state

import (
	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/types"
)

// models state.
var (
	ModelsDir     = ""
	IsModelLoaded = false
	LoadedModel   = ""
	ModelOptions  = DefaultModelOptions
)

// inference state.
var (
	ContinueInferringController = true
	IsInferring                 = false
)

// app state.
var (
	IsVerbose = true
	IsDebug   = false
)

// tasks.
var TasksDir = "./tasks"

// OpenAi api.
var OpenAiConf types.OpenAiConf

// the language model instance.
var Lm llama.LLama
