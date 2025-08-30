// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package state

var (
	// Global app state.
	Verbose = true
	Debug   = false

	// Inference state.
	ContinueInferringController = true
	IsInferring                 = false
)
