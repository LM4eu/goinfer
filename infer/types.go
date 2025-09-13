// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

type (

	// Model holds configuration for a model.
	Model struct {
		Name string `json:"name,omitempty" yaml:"name,omitempty"`
		Ctx  int    `json:"ctx,omitempty"  yaml:"ctx,omitempty"`
	}

	// Sampling contains sampling-related configuration.
	Sampling struct {
		TopK              int     `json:"top_k,omitempty"             yaml:"top_k,omitempty"`
		TopP              float32 `json:"top_p,omitempty"             yaml:"top_p,omitempty"`
		MinP              float32 `json:"min_p,omitempty"             yaml:"min_p,omitempty"`
		Temperature       float32 `json:"temperature,omitempty"       yaml:"temperature,omitempty"`
		FrequencyPenalty  float32 `json:"frequency_penalty,omitempty" yaml:"frequency_penalty,omitempty"`
		PresencePenalty   float32 `json:"presence_penalty,omitempty"  yaml:"presence_penalty,omitempty"`
		RepeatPenalty     float32 `json:"repeat_penalty,omitempty"    yaml:"repeat_penalty,omitempty"`
		TailFreeSamplingZ float32 `json:"tfs,omitempty"               yaml:"tfs,omitempty"`
	}

	// Generation contains generation-related configuration.
	Generation struct {
		StopPrompts []string `json:"stop,omitempty"       yaml:"stop,omitempty"`
		MaxTokens   int      `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	}

	// Media contains media-related configuration.
	Media struct {
		Images []string `json:"images,omitempty" yaml:"images,omitempty"`
		Audios []string `json:"audios,omitempty" yaml:"audios,omitempty"`
	}

	// InferParams holds parameters for inference.
	InferParams struct {
		Media      Media      `json:"media"      yaml:"media"`
		Generation Generation `json:"generation" yaml:"generation"`
		Sampling   Sampling   `json:"sampling"   yaml:"sampling"`
		Stream     bool       `json:"stream"     yaml:"stream"`
	}

	// InferQuery represents a task to be executed.
	InferQuery struct {
		Prompt string      `json:"prompt,omitempty" yaml:"prompt,omitempty"`
		Model  Model       `json:"model"            yaml:"model"`
		Params InferParams `json:"params"           yaml:"params"`
	}

	// StreamedMsg represents a streamed message from the inference engine.
	StreamedMsg struct {
		Data    map[string]any `json:"data,omitempty"`
		Content string         `json:"content,omitempty"`
		MsgType MsgType        `json:"msg_type,omitempty"`
		Num     int            `json:"num,omitempty"`
	}

	// MsgType represents the type of a message.
	MsgType string
)

const (
	TokenMsgType  MsgType = "token"
	SystemMsgType MsgType = "system"
	ErrorMsgType  MsgType = "error"
)

var (
	DefaultModel = Model{
		Name: "default",
		Ctx:  2048,
	}

	DefaultInferParams = InferParams{
		Stream: false,
		Sampling: Sampling{
			TopK:              40,
			TopP:              0.95,
			MinP:              0.05,
			Temperature:       0.2,
			FrequencyPenalty:  0.0,
			PresencePenalty:   0.0,
			RepeatPenalty:     1.0,
			TailFreeSamplingZ: 1.0,
		},
		Generation: Generation{
			MaxTokens:   512,
			StopPrompts: []string{"</s>"},
		},
		Media: Media{},
	}
)
