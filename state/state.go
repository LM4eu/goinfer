package state

import (
	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/types"
)

// models state
var ModelsDir = ""
var IsModelLoaded = false
var LoadedModel = ""
var ModelOptions = DefaultModelOptions

// inference state
var ContinueInferringController = true
var IsInferring = false

// app state
var IsVerbose = true
var IsDebug = false

// tasks
var TasksDir = "./tasks"

// OpenAi api
var OpenAiConf types.OpenAiConf

// the language model instance
var Lm llama.LLama
