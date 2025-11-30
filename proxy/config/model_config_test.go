// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ModelConfigSanitizedCommand(t *testing.T) {
	cfg := &ModelConfig{
		Cmd: `python model1.py \
    --arg1 value1 \
    --arg2 value2`,
	}

	args, err := cfg.SanitizedCommand()
	assert.NoError(t, err)
	assert.Equal(t, []string{"python", "model1.py", "--arg1", "value1", "--arg2", "value2"}, args)
}

func TestConfig_ModelFilters(t *testing.T) {
	content := `
macros:
  default_strip: "temperature, top_p"
models:
  model1:
    cmd: path/to/cmd --port ${PORT}
    filters:
      # macros inserted and list is cleaned of duplicates and empty strings
      stripParams: "model, top_k, top_k, temperature, ${default_strip}, , ,"
  # check for strip_params (legacy field name) compatibility
  legacy:
    cmd: path/to/cmd --port ${PORT}
    filters:
      strip_params: "model, top_k, top_k, temperature, ${default_strip}, , ,"
`
	cfg, err := LoadConfigFromReader(strings.NewReader(content))
	assert.NoError(t, err)
	if cfg == nil {
		return
	}
	for modelId, modelConfig := range cfg.Models {
		t.Run("Testing macros in filters for model "+modelId, func(t *testing.T) {
			assert.Equal(t, "model, top_k, temperature, top_p, ,", modelConfig.Filters.StripParams)
			sanitized, err := modelConfig.Filters.SanitizedStripParams()
			if assert.NoError(t, err) {
				// model has been removed
				// empty strings have been removed
				// duplicates have been removed
				assert.Equal(t, []string{"temperature", "top_k", "top_p"}, sanitized)
			}
		})
	}
}

func TestConfig_ModelSendLoadingState(t *testing.T) {
	content := `
sendLoadingState: true
models:
  model1:
    cmd: path/to/cmd --port ${PORT}
    sendLoadingState: false
  model2:
    cmd: path/to/cmd --port ${PORT}
`
	cfg, err := LoadConfigFromReader(strings.NewReader(content))
	assert.NoError(t, err)
	assert.True(t, cfg.SendLoadingState)
	if assert.NotNil(t, cfg.Models["model1"].SendLoadingState) {
		assert.False(t, *cfg.Models["model1"].SendLoadingState)
	}
	if assert.NotNil(t, cfg.Models["model2"].SendLoadingState) {
		assert.True(t, *cfg.Models["model2"].SendLoadingState)
	}
}
