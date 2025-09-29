// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LM4eu/goinfer/conf"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// Test that Infer.Infer returns an error via errChan when the model name is empty.
func TestInfer_Method_ModelNotLoaded(t *testing.T) {
	t.Parallel()
	inf := &Infer{}
	// Minimal Echo context – request body is irrelevant for this test.
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	query := InferQuery{
		Model: Model{Name: ""}, // empty model name triggers validation error
	}

	resChan := make(chan StreamedMsg, 1)
	errChan := make(chan StreamedMsg, 1)

	inf.Infer(context.Background(), &query, c, resChan, errChan)

	select {
	case msg := <-errChan:
		require.Equal(t, ErrorMsgType, msg.MsgType, "expected error message type")
		require.Contains(t, msg.Error.Error(), "model not loaded", "error message should mention model not loaded")
	default:
		t.Fatalf("expected an error message on errChan")
	}
}

// Test configureAPIKeyAuth when no API key is configured – the middleware should be a no‑op.
func TestConfigureAPIKeyAuth_NoKey(t *testing.T) {
	t.Parallel()
	cfg := &conf.Cfg{
		Server: conf.ServerCfg{
			APIKeys: map[string]string{}, // empty map – no keys
		},
	}
	e := echo.New()
	grp := e.Group("/test")
	// Apply the auth configuration.
	configureAPIKeyAuth(grp, cfg, "model")

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
			APIKeys: map[string]string{
				"model": apiKey,
			},
		},
	}
	e := echo.New()
	grp := e.Group("/secure")
	configureAPIKeyAuth(grp, cfg, "model")

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
