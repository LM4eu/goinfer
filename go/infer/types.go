// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

type (
	// InferQuery represents an inference task request.
	InferQuery struct {
		Prompt            string   `json:"prompt,omitempty" yaml:"prompt,omitempty"`
		Name              string   `json:"name,omitempty" yaml:"name,omitempty"`
		Ctx               int      `json:"ctx,omitempty"  yaml:"ctx,omitempty"`
		Images            []string `json:"images,omitempty" yaml:"images,omitempty"`
		Audios            []string `json:"audios,omitempty" yaml:"audios,omitempty"`
		StopPrompts       []string `json:"stop,omitempty"       yaml:"stop,omitempty"`
		MaxTokens         int      `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
		TopK              int      `json:"top_k,omitempty"             yaml:"top_k,omitempty"`
		TopP              float32  `json:"top_p,omitempty"             yaml:"top_p,omitempty"`
		MinP              float32  `json:"min_p,omitempty"             yaml:"min_p,omitempty"`
		Temperature       float32  `json:"temperature,omitempty"       yaml:"temperature,omitempty"`
		FrequencyPenalty  float32  `json:"frequency_penalty,omitempty" yaml:"frequency_penalty,omitempty"`
		PresencePenalty   float32  `json:"presence_penalty,omitempty"  yaml:"presence_penalty,omitempty"`
		RepeatPenalty     float32  `json:"repeat_penalty,omitempty"    yaml:"repeat_penalty,omitempty"`
		TailFreeSamplingZ float32  `json:"tfs,omitempty"               yaml:"tfs,omitempty"`
		Stream            bool     `json:"stream"     yaml:"stream"`
		Timeout           int      `json:"timeout"          yaml:"timeout"` // in seconds
	}

	// StreamedMsg represents a streamed message from the inference engine.
	StreamedMsg struct {
		Data    map[string]any `json:"data,omitempty"`
		Content string         `json:"content,omitempty"`
		Error   error          `json:"error,omitempty"`
		MsgType MsgType        `json:"msg_type,omitempty"`
		Num     int            `json:"num,omitempty"`
	}

	// MsgType represents the type of a message in the inference protocol.
	MsgType string
)

const (
	// TokenMsgType represents a token message type.
	TokenMsgType MsgType = "token"
	// SystemMsgType represents a system message type.
	SystemMsgType MsgType = "system"
	// ErrorMsgType represents an error message type.
	ErrorMsgType MsgType = "error"
)

var (
	DefaultQuery = InferQuery{
		Name:              "default",
		Ctx:               2048,
		Stream:            false,
		TopK:              40,
		TopP:              0.95,
		MinP:              0.05,
		Temperature:       0.2,
		FrequencyPenalty:  0.0,
		PresencePenalty:   0.0,
		RepeatPenalty:     1.0,
		TailFreeSamplingZ: 1.0,
		MaxTokens:         512,
		StopPrompts:       []string{"</s>"},
	}
)
