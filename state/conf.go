package state

import (
	llama "github.com/go-skynet/go-llama.cpp"
	"github.com/synw/goinfer/types"
)

var DefaultInferenceParams = types.InferenceParams{
	Stream:            false,
	Threads:           4,
	NPredict:          512,
	TopK:              40,
	TopP:              0.95,
	Temperature:       0.2,
	FrequencyPenalty:  0.0,
	PresencePenalty:   0.0,
	RepeatPenalty:     1.0,
	TailFreeSamplingZ: 1.0,
	StopPrompts:       []string{"</s>"},
}

var DefaultModelOptions = llama.ModelOptions{
	ContextSize:   2048,
	Seed:          0,
	F16Memory:     false,
	MLock:         false,
	Embeddings:    false,
	MMap:          true,
	LowVRAM:       false,
	NBatch:        512,
	FreqRopeBase:  10000,
	FreqRopeScale: 1.0,
}

var DefaultModelConf = types.ModelConf{
	Name: "",
	Ctx:  2048,
}
