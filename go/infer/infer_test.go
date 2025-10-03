// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// --- Helper -------------------------------------------------------------------

// newEchoCtx creates an echo instance and a test context wired to the provided
// request and response recorder.
func newEchoCtx(req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
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

	strCtxSize := strconv.Itoa(ctxSize)
	strTemperature := fmt.Sprint(temperature)
	strMaxTokens := strconv.Itoa(maxTokens)

	body := `
	{	"prompt":      "hello",
		"model":       "dummy-model",
		"ctx":         ` + strCtxSize + `,
		"stream":      true,
		"temperature": ` + strTemperature + `,
		"max_tokens":  ` + strMaxTokens + `,
		"stop": ["STOP1", "STOP2"]
	}	}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/infer", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	echoCtx, _ := newEchoCtx(req)

	query, err := parseInferQuery(echoCtx)
	if err != nil {
		t.Fatalf("expected no error, body=%v err=%v", body, err)
	}
	if query == nil {
		t.Fatal("Unexpected nil query")
	}
	if query.Prompt != "hello" {
		t.Errorf("Prompt mismatch: want %q, got %q", "hello", query.Prompt)
	}
	if query.Model != "dummy-model" {
		t.Errorf("Model.Name mismatch: want %q, got %q", "dummy-model", query.Model)
	}
	if query.Ctx != ctxSize {
		t.Errorf("Model.Ctx mismatch: want %d, got %d", ctxSize, query.Ctx)
	}
	if !query.Stream {
		t.Errorf("Params.Stream should be true")
	}
	if query.Temperature != temperature {
		t.Errorf("Params.Sampling.Temperature mismatch: want %v, got %v", temperature, query.Temperature)
	}
	if query.MaxTokens != maxTokens {
		t.Errorf("Params.Generation.MaxTokens mismatch: want %d, got %d", maxTokens, query.MaxTokens)
	}
	if len(query.StopPrompts) != 2 ||
		query.StopPrompts[0] != "STOP1" ||
		query.StopPrompts[1] != "STOP2" {
		t.Errorf("StopPrompts not parsed correctly: %#v", query.StopPrompts)
	}
}

// --- 2. parseInferQuery – missing required `prompt` -------------------------

func TestParseInferQuery_MissingPrompt(t *testing.T) {
	t.Parallel()
	body := `{"model": "dummy"}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/infer", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	echoCtx, _ := newEchoCtx(req)
	_, err := parseInferQuery(echoCtx)
	if err == nil {
		t.Fatalf("expected error for missing prompt, got nil")
	}
	if !errors.Is(err, gie.ErrInvalidPrompt) {
		t.Fatalf("expected ErrInvalidPrompt, got %v", err)
	}
}

func TestParseInferQuery_OnlyPrompt(t *testing.T) {
	t.Parallel()
	body := `{"prompt": "hello LM4"}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/infer", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	echoCtx, _ := newEchoCtx(req)
	query, err := parseInferQuery(echoCtx)
	if err != nil {
		t.Errorf("Error when prompt is provided err=%v", err)
	}
	if query == nil {
		t.Fatal("Unexpected nil query")
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
	var inf Infer
	// Simulate an ongoing inference by setting the flag under lock.
	inf.mu.Lock()
	inf.IsInferring = true
	inf.mu.Unlock()

	// Minimal request body – any valid JSON works; the handler will early‑exit
	// because IsInferring is already true.
	body := `{"prompt":"test"}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/infer", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// Run two concurrent calls to inferHandler.
	var grp sync.WaitGroup
	results := make([]int, 2)

	for i := range 2 {
		grp.Add(1)
		go func(ii int) {
			defer grp.Done()
			c, rec := newEchoCtx(req)
			_ = inf.inferHandler(c) // ignore returned error; handler writes status
			results[ii] = rec.Code
		}(i)
	}
	grp.Wait()

	// At least one request must receive http.StatusOK (200) due to the guard.
	found200 := slices.Contains(results, http.StatusOK)
	if !found200 {
		t.Fatalf("expected at least one request to receive StatusOK (200), got %v", results)
	}
}

// Test configureAPIKeyAuth when no API key is configured – the middleware should be a no‑op.
func TestConfigureAPIKeyAuth_NoKey(t *testing.T) {
	t.Parallel()
	cfg := &conf.Cfg{
		Server: conf.ServerCfg{
			APIKey: "", // no API key
		},
	}
	e := echo.New()
	grp := e.Group("/test")
	// Apply the auth configuration.
	inf := &Infer{Cfg: cfg}
	inf.configureAPIKeyAuth(grp)

	// Attach a simple handler that returns 200.
	grp.GET("", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Request without any Authorization header should succeed.
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
}

// Test configureAPIKeyAuth with a valid API key – only the correct key should be accepted.
func TestConfigureAPIKeyAuth_WithKey(t *testing.T) {
	t.Parallel()
	const apiKey = "secret-key"
	cfg := &conf.Cfg{
		Server: conf.ServerCfg{
			APIKey: apiKey,
		},
	}

	e := echo.New()
	grp := e.Group("/secure")
	inf := &Infer{Cfg: cfg}
	inf.configureAPIKeyAuth(grp)

	grp.GET("", func(c echo.Context) error {
		return c.String(http.StatusOK, "authorized")
	})

	// Request with the correct key in the Authorization header should succeed.
	reqOK := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	reqOK.Header.Set(echo.HeaderAuthorization, "Bearer "+apiKey)
	recOK := httptest.NewRecorder()
	e.ServeHTTP(recOK, reqOK)
	require.Equal(t, http.StatusOK, recOK.Code)
	require.Equal(t, "authorized", recOK.Body.String())

	// Request with an incorrect key should be rejected (401 Unauthorized).
	reqBad := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	reqBad.Header.Set(echo.HeaderAuthorization, "wrong")
	recBad := httptest.NewRecorder()
	e.ServeHTTP(recBad, reqBad)
	require.Equal(t, http.StatusBadRequest, recBad.Code)
}

// Test modelsHandler returns an empty model list when the configuration has none.
func TestModelsHandler_Empty(t *testing.T) {
	t.Parallel()
	inf := &Infer{
		Cfg: &conf.Cfg{
			ModelsDir: t.TempDir(),
			Swap:      conf.Cfg{}.Swap, // zero value – no models configured
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/models", http.NoBody)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err := inf.modelsHandler(c)
	require.NoError(t, err)

	// The response should be JSON with count 0 and an empty models map.
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.Bytes()
	require.Contains(t, string(body), `"count":0`)
	require.Contains(t, string(body), `"models":{}`)
}

// Test the url helper builds the correct address.
func TestURLHelper(t *testing.T) {
	t.Parallel()
	require.Equal(t, "http://localhost:8080/test", url(":8080", "/test"))
	require.Equal(t, "http://127.0.0.1:9090/api", url("127.0.0.1:9090", "/api"))
}
