package types

// ModelConf holds configuration for a model.
type ModelConf struct {
	Name string `json:"name"           yaml:"name"`
	Ctx  int    `json:"ctx,omitempty"  yaml:"ctx,omitempty"`
}

var DefaultModelConf = ModelConf{
	Name: "",
	Ctx:  2048,
}

// InferParams holds parameters for inference.
type InferParams struct {
	Stream            bool     `json:"stream,omitempty"            yaml:"stream,omitempty"`
	MaxTokens         int      `json:"max_tokens,omitempty"        yaml:"max_tokens,omitempty"`
	TopK              int      `json:"top_k,omitempty"             yaml:"top_k,omitempty"`
	TopP              float32  `json:"top_p,omitempty"             yaml:"top_p,omitempty"`
	MinP              float32  `json:"min_p,omitempty"             yaml:"min_p,omitempty"`
	Temperature       float32  `json:"temperature,omitempty"       yaml:"temperature,omitempty"`
	FrequencyPenalty  float32  `json:"frequency_penalty,omitempty" yaml:"frequency_penalty,omitempty"`
	PresencePenalty   float32  `json:"presence_penalty,omitempty"  yaml:"presence_penalty,omitempty"`
	RepeatPenalty     float32  `json:"repeat_penalty,omitempty"    yaml:"repeat_penalty,omitempty"`
	TailFreeSamplingZ float32  `json:"tfs,omitempty"               yaml:"tfs,omitempty"`
	StopPrompts       []string `json:"stop,omitempty"              yaml:"stop,omitempty"`
	Images            []byte   `json:"images,omitempty"            yaml:"images,omitempty"`
	Audios            []byte   `json:"audios,omitempty"            yaml:"audios,omitempty"`
}

var DefaultInferParams = InferParams{
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
	Images:            nil,
}

// InferQuery represents a task to be executed.
type InferQuery struct {
	Prompt      string      `json:"prompt"      yaml:"prompt"`
	ModelConf   ModelConf   `json:"modelConf"   yaml:"modelConf"`
	InferParams InferParams `json:"inferParams" yaml:"inferParams"`
}

// TemplateInfo holds information about a template.
type TemplateInfo struct {
	Name string `json:"name" yaml:"name"`
	Ctx  int    `json:"ctx"  yaml:"ctx"`
}

// StreamedMessage represents a streamed message.
type StreamedMessage struct {
	Content string         `json:"content"`
	Num     int            `json:"num"` // number of tokens
	MsgType MsgType        `json:"msg_type"`
	Data    map[string]any `json:"data,omitempty"`
}

// MsgType represents the type of a message.
type MsgType string

const (
	TokenMsgType  MsgType = "token"
	SystemMsgType MsgType = "system"
	ErrorMsgType  MsgType = "error"
)
