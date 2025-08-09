package types

import "fmt"

// Default values for InferenceParams.
const (
	DefaultThreads           = 4
	DefaultNPredict          = 512
	DefaultTopK              = 40
	DefaultTopP              = 0.95
	DefaultTemperature       = 0.2
	DefaultFrequencyPenalty  = 0.0
	DefaultPresencePenalty   = 0.0
	DefaultRepeatPenalty     = 1.0
	DefaultTailFreeSamplingZ = 1.0
	DefaultStopPrompt        = "</s>"
)

// GoInferConf holds the configuration for GoInfer.
type GoInferConf struct {
	ModelsDir   string
	WebServer   WebServerConf
	OpenAiConf  OpenAiConf
	LlamaConfig *LlamaConfig
}

// WebServerConf holds the configuration for GoInfer web server.
type WebServerConf struct {
	Port string
	EnableApiOpenAi bool `json:"enableApiOpenAi"`
	Origins         []string
	ApiKey          string
}

// LlamaConfig holds configuration for the Llama server proxy.
type LlamaConfig struct {
	BinaryPath string
	ModelPath  string
	Host       string
	Port       int
	Args       []string
}

// InferenceParams holds parameters for inference.
type InferenceParams struct {
	Stream            bool     `json:"stream,omitempty"            yaml:"stream,omitempty"`
	Threads           int      `json:"threads,omitempty"           yaml:"threads,omitempty"`
	NPredict          int      `json:"n_predict,omitempty"         yaml:"n_predict,omitempty"`
	TopK              int      `json:"top_k,omitempty"             yaml:"top_k,omitempty"`
	TopP              float32  `json:"top_p,omitempty"             yaml:"top_p,omitempty"`
	Temperature       float32  `json:"temperature,omitempty"       yaml:"temperature,omitempty"`
	FrequencyPenalty  float32  `json:"frequency_penalty,omitempty" yaml:"frequency_penalty,omitempty"`
	PresencePenalty   float32  `json:"presence_penalty,omitempty"  yaml:"presence_penalty,omitempty"`
	RepeatPenalty     float32  `json:"repeat_penalty,omitempty"    yaml:"repeat_penalty,omitempty"`
	TailFreeSamplingZ float32  `json:"tfs_z,omitempty"             yaml:"tfs_z,omitempty"`
	StopPrompts       []string `json:"stop,omitempty"              yaml:"stop,omitempty"`
}

// NewInferenceParams creates a new InferenceParams with default values.
func NewInferenceParams() InferenceParams {
	return InferenceParams{
		Stream:            false,
		Threads:           DefaultThreads,
		NPredict:          DefaultNPredict,
		TopK:              DefaultTopK,
		TopP:              DefaultTopP,
		Temperature:       DefaultTemperature,
		FrequencyPenalty:  DefaultFrequencyPenalty,
		PresencePenalty:   DefaultPresencePenalty,
		RepeatPenalty:     DefaultRepeatPenalty,
		TailFreeSamplingZ: DefaultTailFreeSamplingZ,
		StopPrompts:       []string{DefaultStopPrompt},
	}
}

// Validate validates the InferenceParams and returns an error if invalid.
func (p InferenceParams) Validate() error {
	// Threads must be positive if set
	if p.Threads <= 0 {
		return fmt.Errorf("threads must be positive, got %d", p.Threads)
	}
	// TopK must be non-negative if set
	if p.TopK < 0 {
		return fmt.Errorf("top_k must be non-negative, got %d", p.TopK)
	}
	// TopP must be between 0.0 and 1.0 if set
	if p.TopP < 0.0 || p.TopP > 1.0 {
		return fmt.Errorf("top_p must be between 0.0 and 1.0, got %f", p.TopP)
	}
	// Temperature must be non-negative if set
	if p.Temperature < 0.0 {
		return fmt.Errorf("temperature must be non-negative, got %f", p.Temperature)
	}
	// RepeatPenalty must be non-negative if set
	if p.RepeatPenalty < 0.0 {
		return fmt.Errorf("repeat_penalty must be non-negative, got %f", p.RepeatPenalty)
	}
	// TailFreeSamplingZ must be non-negative if set
	if p.TailFreeSamplingZ < 0.0 {
		return fmt.Errorf("tail_free_sampling_z must be non-negative, got %f", p.TailFreeSamplingZ)
	}
	return nil
}

// Clone creates a deep copy of InferenceParams.
func (p InferenceParams) Clone() InferenceParams {
	// Create a copy of the slice to avoid sharing references
	var stopPrompts []string
	if p.StopPrompts != nil {
		stopPrompts = make([]string, len(p.StopPrompts))
		copy(stopPrompts, p.StopPrompts)
	}

	return InferenceParams{
		Stream:            p.Stream,
		Threads:           p.Threads,
		NPredict:          p.NPredict,
		TopK:              p.TopK,
		TopP:              p.TopP,
		Temperature:       p.Temperature,
		FrequencyPenalty:  p.FrequencyPenalty,
		PresencePenalty:   p.PresencePenalty,
		RepeatPenalty:     p.RepeatPenalty,
		TailFreeSamplingZ: p.TailFreeSamplingZ,
		StopPrompts:       stopPrompts,
	}
}

// InferenceStats holds statistics about inference.
type InferenceStats struct {
	ThinkingTime       float64 `json:"thinkingTime"`
	ThinkingTimeFormat string  `json:"thinkingTimeFormat"`
	EmitTime           float64 `json:"emitTime"`
	EmitTimeFormat     string  `json:"emitTimeFormat"`
	TotalTime          float64 `json:"totalTime"`
	TotalTimeFormat    string  `json:"totalTimeFormat"`
	TokensPerSecond    float64 `json:"tokensPerSecond"`
	TotalTokens        int     `json:"totalTokens"`
}

// InferenceResult holds the result of inference.
type InferenceResult struct {
	Text  string         `json:"text"`
	Stats InferenceStats `json:"stats"`
}

// Task represents a task to be executed.
type Task struct {
	Name        string          `json:"name"        yaml:"name"`
	Template    string          `json:"template"    yaml:"template"`
	ModelConf   ModelConf       `json:"modelConf"   yaml:"modelConf"`
	InferParams InferenceParams `json:"inferParams" yaml:"inferParams"`
}

// ModelConf holds configuration for a model.
type ModelConf struct {
	Name      string `json:"name"                 yaml:"name"`
	Ctx       int    `json:"ctx,omitempty"        yaml:"ctx,omitempty"`
	GPULayers int    `json:"gpu_layers,omitempty" yaml:"gpu_layers,omitempty"`
}

// TemplateInfo holds information about a template.
type TemplateInfo struct {
	Name string `json:"name" yaml:"name"`
	Ctx  int    `json:"ctx"  yaml:"ctx"`
}

// MsgType represents the type of a message.
type MsgType string

const (
	TokenMsgType  MsgType = "token"
	SystemMsgType MsgType = "system"
	ErrorMsgType  MsgType = "error"
)

// StreamedMessage represents a streamed message.
type StreamedMessage struct {
	Content string         `json:"content"`
	Num     int            `json:"num"` // number of tokens
	MsgType MsgType        `json:"msg_type"`
	Data    map[string]any `json:"data,omitempty"`
}

// ApiType represents the type of API.
type ApiType string

const (
	Llama  ApiType = "llama"
	OpenAi ApiType = "openai"
)
