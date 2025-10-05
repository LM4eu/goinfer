// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

type (
	// OpenaiChatCompletions is the expected JSON request on /v1/chat/completions endpoint.
	// The implementation is inspired by the `ChatCompletionNewParams` struct:
	// https://github.com/openai/openai-go/blob/main/chatcompletion.go#L2902
	// And the llama-server documentation:
	// https://github.com/ggml-org/llama.cpp/tree/master/tools/server#post-v1chatcompletions-openai-compatible-chat-completions-api
	// See also the llama-server code, especially the `oaicompat_completion_params_parse()` function:
	// https://github.com/ggml-org/llama.cpp/blob/master/tools/server/utils.hpp#L478
	// and the `params_from_json_cmpl()` function:
	// https://github.com/ggml-org/llama.cpp/blob/master/tools/server/server.cpp#L294
	//
	// This struct has some restrictions: (last check in Oct. 2025)
	// - `stop` could also be a string, but the llama-server converts the Stop string into an array.
	// - `n` is not proposed because, only `"n": 1` is supported.
	OpenaiChatCompletions struct {
		ModelField

		Messages         []MessageParam    `json:"messages,omitzero,omitempty"`        // Messages.Content+Role are the single mandatory fields
		ToolChoice       any               `json:"tool_choice,omitzero,omitempty"`     // can be: string AllowedToolChoice NamedToolChoice NamedToolChoiceCustom
		FunctionCall     any               `json:"function_call,omitzero,omitempty"`   // can be: string FunctionCallOption
		ResponseFormat   any               `json:"response_format,omitzero,omitempty"` // can be string ResponseFormatJSONSchemaParam ResponseFormatJSONObjectParam
		LogitBias        map[string]int64  `json:"logit_bias,omitzero,omitempty"`
		WebSearchOptions WebSearchOptions  `json:"web_search_options,omitzero"`
		Prediction       PredictionContent `json:"prediction,omitzero"`
		Audio            Audio             `json:"audio,omitzero"`
		Verbosity        string            `json:"verbosity,omitzero,omitempty"`
		ServiceTier      string            `json:"service_tier,omitzero,omitempty"`
		ReasoningEffort  string            `json:"reasoning_effort,omitzero,omitempty"` // minimal, low, medium, high
		PromptCacheKey   string            `json:"prompt_cache_key,omitzero,omitempty"`
		SafetyIdentifier string            `json:"safety_identifier,omitzero,omitempty"`
		User             string            `json:"user,omitzero,omitempty"`
		Stop             []string          `json:"stop,omitzero,omitempty"` // can be string or []string
		Modalities       []string          `json:"modalities,omitzero,omitempty"`
		Functions        []Function        `json:"functions,omitzero,omitempty"`
		Tools            []any             `json:"tools,omitzero,omitempty"` // array of FunctionTool and CustomTool
		Seed             int64             `json:"seed,omitzero,omitempty"`
		TopP             float64           `json:"top_p,omitzero,omitempty"`
		TopLogprobs      int64             `json:"top_logprobs,omitzero,omitempty"`
		Temperature      float64           `json:"temperature,omitzero,omitempty"`
		PresencePenalty  float64           `json:"presence_penalty,omitzero,omitempty"`
		// N             int64            `json:"n,omitzero,omitempty"` // llama-server only supports `"n":1`
		MaxTokens           int64         `json:"max_tokens,omitzero,omitempty"`
		MaxCompletionTokens int64         `json:"max_completion_tokens,omitzero,omitempty"`
		FrequencyPenalty    float64       `json:"frequency_penalty,omitzero,omitempty"`
		StreamOptions       StreamOptions `json:"stream_options,omitzero"`
		ParallelToolCalls   bool          `json:"parallel_tool_calls,omitzero,omitempty"`
		Store               bool          `json:"store,omitzero,omitempty"`
		Logprobs            bool          `json:"logprobs,omitzero,omitempty"`
	}
	AllowedToolChoice struct {
		Type         string       `json:"type,omitzero,omitempty"`
		AllowedTools AllowedTools `json:"allowed_tools,omitzero"`
	}
	Audio struct {
		Format string `json:"format,omitzero,omitempty"`
		Voice  string `json:"voice,omitzero,omitempty"`
	}
	ContentPartUnion struct {
		Type       string                          `json:"type,omitzero,omitempty"`
		Text       string                          `json:"text,omitzero,omitempty"`
		ImageURL   ContentPartImageImageURL        `json:"image_url,omitzero"`
		InputAudio ContentPartInputAudioInputAudio `json:"input_audio,omitzero"`
		File       ContentPartFileFile             `json:"file,omitzero"`
	}
	ContentPartFileFile struct {
		FileData string `json:"file_data,omitzero,omitempty"`
		FileID   string `json:"file_id,omitzero,omitempty"`
		Filename string `json:"filename,omitzero,omitempty"`
	}
	ContentPartImageImageURL struct {
		URL    string `format:"uri" json:"url,omitzero,omitempty"`
		Detail string `             json:"detail,omitzero,omitempty"`
	}
	ContentPartInputAudioInputAudio struct {
		Data   string `json:"data,omitzero,omitempty"`
		Format string `json:"format,omitzero,omitempty"`
	}
	ContentPartText struct {
		Text string `json:"text,omitzero,omitempty"`
		Type string `json:"type,omitzero,omitempty"`
	}
	CustomTool struct {
		Custom CustomToolCustom `json:"custom,omitzero,omitempty"`
		Type   string           `json:"type,omitzero,omitempty"`
	}
	CustomToolCustom struct {
		Format      CustomToolCustomFormat `json:"format,omitzero,omitempty"`
		Name        string                 `json:"name,omitzero,omitempty"`
		Description string                 `json:"description,omitzero,omitempty"`
	}
	CustomToolCustomFormat struct {
		Type    string                               `json:"type,omitzero,omitempty"`
		Grammar CustomToolCustomFormatGrammarGrammar `json:"grammar,omitzero"`
	}
	CustomToolCustomFormatGrammarGrammar struct {
		Definition string `json:"definition,omitzero,omitempty"`
		Syntax     string `json:"syntax,omitzero,omitempty"`
	}
	FunctionCallOption struct {
		Name string `json:"name,omitzero,omitempty"`
	}
	FunctionTool struct {
		Type     string             `json:"type,omitzero,omitempty"`
		Function FunctionDefinition `json:"function,omitzero"`
	}
	FunctionDefinition struct {
		Parameters  map[string]any `json:"parameters,omitzero,omitempty"`
		Name        string         `json:"name"`
		Description string         `json:"description,omitzero"`
		Strict      bool           `json:"strict,omitzero"`
	}
	MessageParam struct {
		Name       string `json:"name,omitzero,omitempty"`
		Role       string `json:"role,omitzero,omitempty"`
		Content    any    `json:"content,omitzero,omitempty"` // required: string []ContentPartText []contentPartUnion
		ToolCallID string `json:"tool_call_id,omitzero,omitempty"`
	}
	NamedToolChoice struct {
		Function NamedToolChoiceFunction `json:"function,omitzero"`
		Type     string                  `json:"type,omitzero,omitempty"`
	}
	NamedToolChoiceFunction struct {
		Name string `json:"name,omitzero,omitempty"`
	}
	NamedToolChoiceCustom struct {
		Custom NamedToolChoiceCustomCustom `json:"custom,omitzero"`
		Type   string                      `json:"type,omitzero,omitempty"`
	}
	NamedToolChoiceCustomCustom struct {
		Name string `json:"name,omitzero,omitempty"`
	}
	PredictionContent struct {
		Content any    `json:"content,omitzero,omitempty"`
		Type    string `json:"type,omitzero,omitempty"`
	}
	StreamOptions struct {
		IncludeObfuscation bool `json:"include_obfuscation,omitzero,omitempty"`
		IncludeUsage       bool `json:"include_usage,omitzero,omitempty"`
	}
	AllowedTools struct {
		Mode  string           `json:"mode,omitzero,omitempty"`
		Tools []map[string]any `json:"tools,omitzero,omitempty"`
	}
	Function struct {
		Parameters  map[string]any `json:"parameters,omitzero,omitempty"`
		Name        string         `json:"name,omitzero,omitempty"`
		Description string         `json:"description,omitzero,omitempty"`
	}
	ResponseFormatJSONObjectParam struct {
		Type string `json:"type"` // value is always `json_object`
	}
	ResponseFormatJSONSchemaParam struct {
		Type       string                             `json:"type"`
		JSONSchema ResponseFormatJSONSchemaJSONSchema `json:"json_schema,omitzero"`
	}
	ResponseFormatJSONSchemaJSONSchema struct {
		Schema      any    `json:"schema,omitzero,omitempty"`
		Name        string `json:"name"`
		Description string `json:"description,omitzero,omitempty"`
		Strict      bool   `json:"strict,omitzero,omitempty"`
	}
	WebSearchOptions struct {
		UserLocation      WebSearchOptionsUserLocation `json:"user_location,omitzero"`
		SearchContextSize string                       `json:"search_context_size,omitzero,omitempty"`
	}
	WebSearchOptionsUserLocation struct {
		Approximate WebSearchOptionsUserLocationApproximate `json:"approximate,omitzero"`
		Type        string                                  `json:"type,omitzero,omitempty"`
	}
	WebSearchOptionsUserLocationApproximate struct {
		City     string `json:"city,omitzero,omitempty"`
		Country  string `json:"country,omitzero,omitempty"`
		Region   string `json:"region,omitzero,omitempty"`
		Timezone string `json:"timezone,omitzero,omitempty"`
	}
)
