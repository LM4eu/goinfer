// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// --- Helper -------------------------------------------------------------------

// newEcho creates an echo instance and a test context wired to the provided
// request and response recorder.
func newEcho(req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	return c, rec
}

// --- 1. parseInferQuery – valid payload ---------------------------------------

func TestParseInferQuery_ValidPayload(t *testing.T) {
	t.Parallel()

	const ctxSize = 2048
	const temperature = 0.7
	const maxTokens = 128

	payload := map[string]any{
		"prompt":      "hello",
		"model":       "dummy-model",
		"ctx":         ctxSize,
		"stream":      true,
		"temperature": temperature,
		"max_tokens":  maxTokens,
		"stop": []any{
			"STOP1",
			"STOP2",
		},
	}
	query, err := parseInferQuery(t.Context(), payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if query.Prompt != "hello" {
		t.Errorf("Prompt mismatch: want %q, got %q", "hello", query.Prompt)
	}
	if query.Model.Name != "dummy-model" {
		t.Errorf("Model.Name mismatch: want %q, got %q", "dummy-model", query.Model.Name)
	}
	if query.Model.Ctx != ctxSize {
		t.Errorf("Model.Ctx mismatch: want %d, got %d", ctxSize, query.Model.Ctx)
	}
	if !query.Params.Stream {
		t.Errorf("Params.Stream should be true")
	}
	if query.Params.Sampling.Temperature != temperature {
		t.Errorf("Params.Sampling.Temperature mismatch: want %v, got %v", temperature, query.Params.Sampling.Temperature)
	}
	if query.Params.Generation.MaxTokens != maxTokens {
		t.Errorf("Params.Generation.MaxTokens mismatch: want %d, got %d", maxTokens, query.Params.Generation.MaxTokens)
	}
	if len(query.Params.Generation.StopPrompts) != 2 ||
		query.Params.Generation.StopPrompts[0] != "STOP1" ||
		query.Params.Generation.StopPrompts[1] != "STOP2" {

		t.Errorf("StopPrompts not parsed correctly: %#v", query.Params.Generation.StopPrompts)
	}
}

// --- 2. parseInferQuery – missing required `prompt` -------------------------

func TestParseInferQuery_MissingPrompt(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"model": "dummy",
	}
	_, err := parseInferQuery(t.Context(), payload)
	if err == nil {
		t.Fatalf("expected error for missing prompt, got nil")
	}
	if !errors.Is(err, gie.ErrInvalidPrompt) {
		t.Fatalf("expected ErrInvalidPrompt, got %v", err)
	}
}

// --- 3. getInt – type mismatch ------------------------------------------------

func TestGetInt_TypeMismatch(t *testing.T) {
	t.Parallel()
	m := echo.Map{
		"someInt": "not-an-int",
	}
	v := getInt(t.Context(), m, "someInt")
	if v != 0 {
		t.Fatalf("expected type‑mismatch error, got nil")
	}
}

// --- 4. getFloat – type mismatch ----------------------------------------------

func TestGetFloat_TypeMismatch(t *testing.T) {
	t.Parallel()
	m := echo.Map{
		"someFloat": "not-a-float",
	}
	v := getFloat(t.Context(), m, "someFloat")
	if v != 0.0 {
		t.Fatalf("expected type‑mismatch error, got nil")
	}
}

// --- 5. Concurrency guard – only one inference at a time ---------------------

func TestConcurrencyGuard(t *testing.T) {
	t.Parallel()
	// Build a dummy Infer instance – we can keep Cfg nil because the handler
	// never dereferences it before the mutex check.
	inf := &Infer{
		Cfg: nil,
	}
	// Simulate an ongoing inference by setting the flag under lock.
	inf.mu.Lock()
	inf.IsInferring = true
	inf.mu.Unlock()

	// Minimal request body – any valid JSON works; the handler will early‑exit
	// because IsInferring is already true.
	body := `{"prompt":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/infer", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// Run two concurrent calls to inferHandler.
	var grp sync.WaitGroup
	results := make([]int, 2)

	for i := range 2 {
		grp.Add(1)
		go func(idx int) {
			defer grp.Done()
			c, rec := newEcho(req)
			_ = inf.inferHandler(c) // ignore returned error; handler writes status
			results[idx] = rec.Code
		}(i)
	}
	grp.Wait()

	// At least one request must receive http.StatusAccepted (202) due to the guard.
	found202 := slices.Contains(results, http.StatusAccepted)
	if !found202 {
		t.Fatalf("expected at least one request to receive StatusAccepted (202), got %v", results)
	}
}
