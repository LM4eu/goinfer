// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"slices"
	"testing"

	"github.com/lynxai-team/goinfer/conf"
)

func TestNewEcho(t *testing.T) {
	t.Parallel()
	// Minimal configuration required by NewEcho.
	cfg := &conf.Cfg{
		Origins: "http://localhost",
		APIKey:  "test/key",
	}

	inf := &Infer{Cfg: cfg}
	e := inf.NewEcho()
	if e == nil {
		t.Fatalf("NewEcho returned nil")
	}

	// No explicit middleware check – Echo always registers the logger middleware internally.

	wantRoutes := []string{
		"/models",
		"/completion",
		"/completions",
		"/v1/chat/completions",
		"/abort",
	}

	// Collect all registered route paths.
	routes := make([]string, 0, len(wantRoutes))
	for _, r := range e.Routes() {
		routes = append(routes, r.Path)
	}

	// Ensure each expected route appears.
	for _, want := range wantRoutes {
		found := slices.Contains(routes, want)
		if !found {
			t.Fatalf("expected route %s to be registered", want)
		}
	}
}
