package types

type OpenAiChatCompletion struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAiChoice `json:"choices"`
	Usage   OpenAiUsage    `json:"usage"`
}

type Delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeltaChoice struct {
	Delta        Delta  `json:"delta"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason,omitempty"`
}

type OpenAiChatCompletionDeltaResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []DeltaChoice `json:"choices"`
}

type OpenAiChoice struct {
	Index        int           `json:"index"`
	Message      OpenAiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAiConf struct {
	Threads  int    `json:"threads"`
	Template string `json:"template"`
}

type OpenAiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}
