// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"strings"
	"testing"
)

func TestCfg_GenLlamaINI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		info *ModelInfo
		want string
	}{
		{"model-with-flags", &ModelInfo{Flags: "  -c   0   --n-gpu-layers 99  --no-jinja 	--context-switch  "}, `
version = 1

[model-with-flags]
model = /path/model-with-flags.gguf
c = 0
n-gpu-layers = 99
no-jinja = true
context-switch = true

[model-with-flags` + PLUS_A + `]
model = /path/model-with-flags.gguf
jinja = true
chat-template-file = template.jinja
c = 0
n-gpu-layers = 99
no-jinja = true
context-switch = true
`},
		{"model-no-flags", &ModelInfo{Flags: ""}, `
version = 1

[model-no-flags]
model = /path/model-no-flags.gguf

[model-no-flags` + PLUS_A + `]
model = /path/model-no-flags.gguf
jinja = true
chat-template-file = template.jinja
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := DefaultCfg()
			tt.info.Path = "/path/" + tt.name + ".gguf"
			cfg.Info = map[string]*ModelInfo{tt.name: tt.info}
			got := string(cfg.GenLlamaINI())
			if strings.TrimSpace(got) != strings.TrimSpace(tt.want) {
				t.Errorf("GenLlamaINI() = %v, want %v", got, tt.want)
			}
		})
	}
}
