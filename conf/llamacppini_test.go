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
		name  string
		flags string
		want  string
	}{
		{"flags", "  -c   0   --n-gpu-layers 99  --no-jinja 	--context-switch  ", `version = 1

[flags]
model = /path/flags.gguf
c = 0
n-gpu-layers = 99
no-jinja = true
context-switch = true

[flags` + PLUS_A + `]
model = /path/flags.gguf
jinja = true
chat-template-file = template.jinja
c = 0
n-gpu-layers = 99
no-jinja = true
context-switch = true`},
		{"no-flags", "", `version = 1

[no-flags]
model = /path/no-flags.gguf

[no-flags` + PLUS_A + `]
model = /path/no-flags.gguf
jinja = true
chat-template-file = template.jinja`},
		{"quote", `--chat-template-kwargs '{"reasoning_effort": "high"}' -ot "blk\.1.\.ffn_.*=CPU"`, `version = 1

[quote]
model = /path/quote.gguf
chat-template-kwargs = {"reasoning_effort": "high"}
ot = blk\.1.\.ffn_.*=CPU

[quote` + PLUS_A + `]
model = /path/quote.gguf
jinja = true
chat-template-file = template.jinja
chat-template-kwargs = {"reasoning_effort": "high"}
ot = blk\.1.\.ffn_.*=CPU`},
		{"negative", `--treads -1`, `
version = 1

[negative]
model = /path/negative.gguf
treads = -1

[negative` + PLUS_A + `]
model = /path/negative.gguf
jinja = true
chat-template-file = template.jinja
treads = -1`}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := DefaultCfg()
			cfg.Info = map[string]*ModelInfo{tt.name: {Flags: tt.flags, Path: "/path/" + tt.name + ".gguf"}}
			got := string(cfg.GenLlamaINI())
			if strings.TrimSpace(got) != strings.TrimSpace(tt.want) {
				t.Errorf(`
------------------------------------------------
got: %v
------------------------------------------------
want: %v
------------------------------------------------`, got, tt.want)
			}
		})
	}
}
