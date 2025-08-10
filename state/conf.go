package state

import (
	"github.com/synw/goinfer/types"
)

var DefaultInferenceParams = types.InferenceParams{
	Stream:            false,
	MaxTokens:         512,
	TopK:              40,
	TopP:              0.95,
	MinP:              0.05,
	Temperature:       0.2,
	FrequencyPenalty:  0.0,
	PresencePenalty:   0.0,
	RepeatPenalty:     1.0,
	TailFreeSamplingZ: 1.0,
	StopPrompts:       []string{"</s>"},
}

var DefaultModelConf = types.ModelConf{
	Name: "",
	Ctx:  2048,
}
