// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

// Completion is the expected JSON request on /completion endpoint (llama-server).
// The /completions endpoint (with final s) does the same (/completion is the legacy).
// https://github.com/ggml-org/llama.cpp/blob/master/tools/server#post-completion-given-a-prompt-it-returns-the-predicted-completion
// See also the C++ source code, especially the `params_from_json_cmpl()` function:
// https://github.com/ggml-org/llama.cpp/blob/master/tools/server/server.cpp#L294
//
// This struct as some restrictions: (last check in Oct. 2025)
// - `stop` could also be a string, but the C++ code convert that string into an array.
// - we use `max_tokens`: `n_predict` is native llama.cpp and `max_tokens` is the compatibility with OpenAI API.
type Completion struct {
	Prompt              any              `json:"prompt,omitempty"`
	JSONSchema          map[string]any   `json:"json_schema,omitempty"`
	Grammar             string           `json:"grammar,omitempty"`
	Stop                []string         `json:"stop,omitempty"`
	LoRA                []map[string]any `json:"lora,omitempty"`
	ResponseFields      []string         `json:"response_fields,omitempty"`
	Samplers            []string         `json:"samplers,omitempty"`
	LogitBias           [][2]any         `json:"logit_bias,omitempty"`
	DrySequenceBreakers []rune           `json:"dry_sequence_breakers,omitempty"`
	MinKeep             int              `json:"min_keep,omitempty"`
	NIndent             int              `json:"n_indent,omitempty"`
	NKeep               int              `json:"n_keep,omitempty"`
	IDSlot              int              `json:"id_slot,omitempty"`
	TMaxPredictMs       int              `json:"t_max_predict_ms,omitempty"`
	RepeatLastN         int              `json:"repeat_last_n,omitempty"`
	TopK                int              `json:"top_k,omitempty"`
	NProbs              int              `json:"n_probs,omitempty"`
	Seed                int64            `json:"seed,omitempty"`
	MaxTokens           int              `json:"max_tokens,omitempty"`
	DryAllowedLength    int              `json:"dry_allowed_length,omitempty"`
	DryPenaltyLastN     int              `json:"dry_penalty_last_n,omitempty"`
	DryBase             float32          `json:"dry_base,omitempty"`
	MinP                float32          `json:"min_p,omitempty"`
	XTCThreshold        float32          `json:"xtc_threshold,omitempty"`
	Temperature         float32          `json:"temperature,omitempty"`
	MirostatTau         float32          `json:"mirostat_tau,omitempty"`
	MirostatEta         float32          `json:"mirostat_eta,omitempty"`
	XTCProbability      float32          `json:"xtc_probability,omitempty"`
	DynatempExponent    float32          `json:"dynatemp_exponent,omitempty"`
	DryMultiplier       float32          `json:"dry_multiplier,omitempty"`
	DynatempRange       float32          `json:"dynatemp_range,omitempty"`
	TopP                float32          `json:"top_p,omitempty"`
	FrequencyPenalty    float32          `json:"frequency_penalty,omitempty"`
	PresencePenalty     float32          `json:"presence_penalty,omitempty"`
	RepeatPenalty       float32          `json:"repeat_penalty,omitempty"`
	TypicalP            float32          `json:"typical_p,omitempty"`
	Stream              bool             `json:"stream,omitempty"`
	ReturnTokens        bool             `json:"return_tokens,omitempty"`
	CachePrompt         bool             `json:"cache_prompt,omitempty"`
	TimingsPerToken     bool             `json:"timings_per_token,omitempty"`
	ReturnProgress      bool             `json:"return_progress,omitempty"`
	PostSamplingProbs   bool             `json:"post_sampling_probs,omitempty"`
	IgnoreEOS           bool             `json:"ignore_eos,omitempty"`
	Mirostat            bool             `json:"mirostat,omitempty"`
}
