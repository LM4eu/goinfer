package state

import (
	"github.com/synw/goinfer/types"
)

// models state.
var (
	ModelsDir     = ""
	IsModelLoaded = false
	LoadedModel   = ""
	ModelConf     = types.DefaultModelConf
)

// Inference state.
var (
	ContinueInferringController = true
	IsInferring                 = false
)

// app state.
var (
	IsVerbose = true
	IsDebug   = false
)
