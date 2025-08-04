package llama

//--------------------------------------------------------------------------------
// from github.com/go-skynet/go-LLama.cpp

type LLama string

func New(model string, opts ...ModelOption) (LLama, error) {
	var l LLama
	return l, nil
}

type ModelOption func(p *ModelOptions)

type ModelOptions struct {
	ContextSize   int
	Seed          int
	NBatch        int
	F16Memory     bool
	MLock         bool
	MMap          bool
	LowVRAM       bool
	Embeddings    bool
	NUMA          bool
	NGPULayers    int
	MainGPU       string
	TensorSplit   string
	FreqRopeBase  float32
	FreqRopeScale float32
	MulMatQ       *bool
	LoraBase      string
	LoraAdapter   string
	Perplexity    bool
}

// SetContext sets the context size.
func SetContext(c int) ModelOption {
	return func(p *ModelOptions) {
		p.ContextSize = c
	}
}

// SetGPULayers sets the number of GPU layers to use to offload computation
func SetGPULayers(n int) ModelOption {
	return func(p *ModelOptions) {
		p.NGPULayers = n
	}
}

var EnableEmbeddings ModelOption = func(p *ModelOptions) {
	p.Embeddings = true
}

func (l *LLama) Predict(text string, opts ...PredictOption) (string, error) {
	return string(*l), nil
}

func (l *LLama) Free() {}

// SetTokenCallback sets the prompts that will stop predictions.
func SetTokenCallback(fn func(string) bool) PredictOption {
	return func(p *PredictOptions) {
		p.TokenCallback = fn
	}
}

func SetTokens(tokens int) PredictOption {
	return func(p *PredictOptions) {
		p.Tokens = tokens
	}
}

// SetTopK sets the value for top-K sampling.
func SetTopK(topk int) PredictOption {
	return func(p *PredictOptions) {
		p.TopK = topk
	}
}

// SetTopP sets the value for nucleus sampling.
func SetTopP(topp float32) PredictOption {
	return func(p *PredictOptions) {
		p.TopP = topp
	}
}

// SetTemperature sets the temperature value for text generation.
func SetTemperature(temp float32) PredictOption {
	return func(p *PredictOptions) {
		p.Temperature = temp
	}
}

// SetPathPromptCache sets the session file to store the prompt cache.
func SetPathPromptCache(f string) PredictOption {
	return func(p *PredictOptions) {
		p.PathPromptCache = f
	}
}

// SetPenalty sets the repetition penalty for text generation.
func SetPenalty(penalty float32) PredictOption {
	return func(p *PredictOptions) {
		p.Penalty = penalty
	}
}

func SetRopeFreqBase(ropeFreqBase float32) PredictOption {
	return func(p *PredictOptions) {
		p.RopeFreqBase = ropeFreqBase
	}
}

// SetRepeat sets the number of times to repeat text generation.
func SetRepeat(repeat int) PredictOption {
	return func(p *PredictOptions) {
		p.Repeat = repeat
	}
}

// SetBatch sets the batch size.
func SetBatch(size int) PredictOption {
	return func(p *PredictOptions) {
		p.Batch = size
	}
}

// SetThreads sets the number of threads to use for text generation.
func SetThreads(threads int) PredictOption {
	return func(p *PredictOptions) {
		p.Threads = threads
	}
}

// SetStopWords sets the prompts that will stop predictions.
func SetStopWords(stop ...string) PredictOption {
	return func(p *PredictOptions) {
		p.StopPrompts = stop
	}
}

// SetFrequencyPenalty sets the frequency penalty parameter, freq_penalty.
func SetFrequencyPenalty(fp float32) PredictOption {
	return func(p *PredictOptions) {
		p.FrequencyPenalty = fp
	}
}

// SetPresencePenalty sets the presence penalty parameter, presence_penalty.
func SetPresencePenalty(pp float32) PredictOption {
	return func(p *PredictOptions) {
		p.PresencePenalty = pp
	}
}

type PredictOption func(p *PredictOptions)

type PredictOptions struct {
	Seed, Threads, Tokens, TopK, Repeat, Batch, NKeep int
	TopP, Temperature, Penalty                        float32
	NDraft                                            int
	F16KV                                             bool
	DebugMode                                         bool
	StopPrompts                                       []string
	IgnoreEOS                                         bool

	TailFreeSamplingZ float32
	TypicalP          float32
	FrequencyPenalty  float32
	PresencePenalty   float32
	Mirostat          int
	MirostatETA       float32
	MirostatTAU       float32
	PenalizeNL        bool
	LogitBias         string
	TokenCallback     func(string) bool

	PathPromptCache             string
	MLock, MMap, PromptCacheAll bool
	PromptCacheRO               bool
	Grammar                     string
	MainGPU                     string
	TensorSplit                 string

	// Rope parameters
	RopeFreqBase  float32
	RopeFreqScale float32

	// Negative prompt parameters
	NegativePromptScale float32
	NegativePrompt      string
}
