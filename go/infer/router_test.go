// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"slices"
	"testing"

	"github.com/LM4eu/goinfer/conf"
)

func TestNewEcho(t *testing.T) {
	t.Parallel()
	// Minimal configuration required by NewEcho.
	cfg := &conf.Cfg{
		Server: conf.ServerCfg{
			Origins: "http://localhost",
			APIKeys: map[string]string{
				"model":   "testkey",
				"goinfer": "testkey",
				"openai":  "testkey",
			},
		},
	}

	tests := []struct {
		name          string
		wantRoutes    []string
		enableModels  bool
		enableGoinfer bool
		enableOpenAPI bool
	}{
		{
			name:          "all disabled",
			enableModels:  false,
			enableGoinfer: false,
			enableOpenAPI: false,
			wantRoutes:    []string{},
		},
		{
			name:          "models only",
			enableModels:  true,
			enableGoinfer: false,
			enableOpenAPI: false,
			wantRoutes:    []string{"/models"},
		},
		{
			name:          "full stack",
			enableModels:  true,
			enableGoinfer: true,
			enableOpenAPI: true,
			wantRoutes: []string{
				"/models",
				"/infer",
				"/infer/abort",
				"/v1/chat/completions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inf := &Infer{Cfg: cfg}
	e := inf.NewEcho(":0")
			if e == nil {
				t.Fatalf("NewEcho returned nil")
			}

			// No explicit middleware check – Echo always registers the logger middleware internally.

			// Collect all registered route paths.
			var routes []string
			for _, r := range e.Routes() {
				routes = append(routes, r.Path)
			}

			// Ensure each expected route appears.
			for _, want := range tt.wantRoutes {
				found := slices.Contains(routes, want)
				if !found {
					t.Fatalf("expected route %s to be registered", want)
				}
			}
		})
	}
}
