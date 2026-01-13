// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lynxai-team/goinfer/conf"
	"github.com/lynxai-team/goinfer/proxy"
	"github.com/stretchr/testify/require"
)

// newEchoCtx creates an echo instance and a test context wired to the provided
// request and response recorder.
func newEchoCtx(req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestConcurrencyGuard(t *testing.T) {
	t.Parallel()

	// dummy Infer instance
	inf := Infer{Cfg: &conf.Cfg{}, ProxyMan: &proxy.ProxyManager{}}
	// minimalist request
	body := `{"prompt":"test"}`

	// Run some concurrent calls to inferHandler.
	results := make([]int, 20)

	var grp sync.WaitGroup
	for i := range results {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/completion", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		grp.Add(1)
		go func(ii int) {
			c, rec := newEchoCtx(req)
			_ = inf.completionHandler(c) // ignore returned error; handler writes status
			results[ii] = rec.Code
			grp.Done()
		}(i)
	}
	grp.Wait()

	// At least one request must receive http.StatusOK (200) due to the guard.
	found200 := slices.Contains(results, http.StatusOK)
	if !found200 {
		t.Fatalf("expected at least one request to receive StatusOK (200), got %v", results)
	}
}

// Test configureAPIKeyAuth when no API key is configured – the middleware should be a no-op.
func TestConfigureAPIKeyAuth_NoKey(t *testing.T) {
	t.Parallel()
	cfg := &conf.Cfg{APIKey: ""} // no API key
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
	cfg := &conf.Cfg{APIKey: apiKey}
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
	inf := &Infer{Cfg: &conf.Cfg{ModelsDir: t.TempDir()}}
	req := httptest.NewRequest(http.MethodGet, "/models", http.NoBody)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err := inf.modelsHandler(c)
	require.NoError(t, err)

	// The response should be JSON with count 0 and an empty models map.
	require.Equal(t, http.StatusNoContent, rec.Code)
	body := rec.Body.Bytes()
	require.Contains(t, string(body), `"count":0`)
	require.Contains(t, string(body), `"models"`)
}
