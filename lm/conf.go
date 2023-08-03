package lm

import (
	llama "github.com/go-skynet/go-llama.cpp"
	"github.com/synw/altiplano/goinfer/types"
)

var DefaultInferenceParams = types.InferenceParams{
	Threads:           4,
	Tokens:            512,
	TopK:              40,
	TopP:              0.95,
	Temperature:       0.2,
	FrequencyPenalty:  0.0,
	PresencePenalty:   0.0,
	TailFreeSamplingZ: 1.0,
	StopPrompts:       "</end>",
}

var DefaultModelParams = llama.ModelOptions{
	ContextSize: 1024,
	Embeddings:  false,
	NGPULayers:  0,
}
