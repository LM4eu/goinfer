package types

type GoInferConf struct {
	ModelsDir  string
	TasksDir   string
	Origins    []string
	ApiKey     string
	OpenAiConf OpenAiConf
}

type InferenceParams struct {
	Threads           int     `json:"threads,omitempty" yaml:"threads,omitempty"`
	Tokens            int     `json:"tokens,omitempty" yaml:"tokens,omitempty"`
	TopK              int     `json:"topK,omitempty" yaml:"topK,omitempty"`
	TopP              float32 `json:"topP,omitempty" yaml:"topP,omitempty"`
	Temperature       float32 `json:"temp,omitempty" yaml:"temp,omitempty"`
	FrequencyPenalty  float32 `json:"freqPenalty,omitempty" yaml:"freqPenalty,omitempty"`
	PresencePenalty   float32 `json:"presPenalty,omitempty" yaml:"presPenalty,omitempty"`
	RepeatPenalty     float32 `json:"repeatPenalty,omitempty" yaml:"repeatPenalty,omitempty"`
	TailFreeSamplingZ float32 `json:"tfs,omitempty" yaml:"tfs,omitempty"`
	StopPrompts       string  `json:"stop,omitempty" yaml:"stop,omitempty"`
}

type InferenceResult struct {
	Text               string  `json:"text"`
	ThinkingTime       float64 `json:"thinkingTime"`
	ThinkingTimeFormat string  `json:"thinkingTimeFormat"`
	EmitTime           float64 `json:"emitTime"`
	EmitTimeFormat     string  `json:"emitTimeFormat"`
	TotalTime          float64 `json:"totalTime"`
	TotalTimeFormat    string  `json:"totalTimeFormat"`
	TokensPerSecond    float64 `json:"tokensPerSecond"`
	TotalTokens        int     `json:"totalTokens"`
}

type Task struct {
	Name        string          `json:"name" yaml:"name"`
	Model       string          `json:"model" yaml:"model"`
	Template    string          `json:"template" yaml:"template"`
	ModelConf   ModelConf       `json:"modelConf,omitempty" yaml:"modelConf,omitempty"`
	InferParams InferenceParams `json:"inferParams,omitempty" yaml:"inferParams,omitempty"`
}

type ModelConf struct {
	Ctx int `json:"ctx" yaml:"ctx"`
}

type MsgType string

const (
	TokenMsgType  MsgType = "token"
	SystemMsgType MsgType = "system"
)

type StreamedMessage struct {
	Content string  `json:"content"`
	Num     int     `json:"num"`
	MsgType MsgType `json:"msg_type"`
}
