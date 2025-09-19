// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/LM4eu/goinfer/conf"
	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

// Infer manages proxying requests to the backend LLM engine.
type Infer struct {
	Cfg                         *conf.GoInferCfg
	IsInferring                 bool
	ContinueInferringController bool
	mu                          sync.Mutex
}

//go:embed all:dist
var embeddedFiles embed.FS

// NewEcho creates a new Echo server configured with Goinfer routes and middleware.
func (inf *Infer) NewEcho(ctx context.Context, cfg *conf.GoInferCfg, addr string,
	enableAdminWebUI, enableModelsEndpoint, enableGoinferEndpoint, enableOpenAPIEndpoint bool,
) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Middleware logger
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method} ${status} ${uri}  ${latency_human} ${remote_ip} ${error}\n",
	}))

	if l, ok := e.Logger.(*log.Logger); ok {
		l.SetHeader("[${time_rfc3339}] ${level}")
	}

	// Middleware CORS
	if cfg.Server.Origins != "" {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     strings.Split(cfg.Server.Origins, ","),
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
			AllowMethods:     []string{http.MethodGet, http.MethodOptions, http.MethodPost},
			AllowCredentials: true,
		}))
	}

	// Middleware unified errors
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				return gie.HandleError(err, c)
			}
			return nil
		}
	})

	// ------- Admin web frontend -------
	if enableAdminWebUI {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "dist",
			Index:      "index.html",
			Browse:     false,
			HTML5:      true,
			Filesystem: http.FS(embeddedFiles),
		}))
		slog.InfoContext(ctx, "Listen GET (web UI)", "addr", addr)
	}

	// ------------ Models ------------
	if enableModelsEndpoint {
		grp := e.Group("/models")
		configureAPIKeyAuth(ctx, grp, cfg, "model")
		grp.GET("", inf.modelsHandler)
		slog.InfoContext(ctx, "Listen GET models endpoint", "addr", addr)
	}

	// ----- Inference (llama.cpp) -----
	if enableGoinferEndpoint {
		grp := e.Group("/goinfer")
		configureAPIKeyAuth(ctx, grp, cfg, "goinfer")
		grp.POST("", inf.inferHandler)
		grp.GET("/abort", inf.abortHandler)
		slog.InfoContext(ctx, "Listen POST goinfer endpoint", "addr", addr)
		slog.InfoContext(ctx, "Listen GET goinfer abort", "addr", addr)
	}

	// ----- Inference OpenAI API -----
	if enableOpenAPIEndpoint {
		grp := e.Group("/v1")
		grp.POST("/chat/completions", inf.handleChatCompletions)
		configureAPIKeyAuth(ctx, grp, cfg, "openai")
		slog.InfoContext(ctx, "Listen POST chat completions", "addr", addr)
	}

	return e
}

// configureAPIKeyAuth sets up API‑key authentication for a grp.
func configureAPIKeyAuth(ctx context.Context, grp *echo.Group, cfg *conf.GoInferCfg, service string) {
	// Select the API key with preference order
	key, exists := cfg.Server.APIKeys[service]
	if !exists {
		key, exists = cfg.Server.APIKeys["user"]
		if !exists {
			key, exists = cfg.Server.APIKeys["admin"]
			if !exists {
				slog.WarnContext(ctx, "No API key for service, disabling API key security", "service", service)
				return
			}
		}
	}

	if key == "" {
		slog.WarnContext(ctx, "Empty API key => disable API key for service", "service", service)
		return
	}

	grp.Use(middleware.KeyAuth(func(received_key string, c echo.Context) (bool, error) {
		if received_key == key {
			return true, nil
		}

		slog.WarnContext(ctx, "Received API key is NOT the configured one for service", "service", service)
		return false, nil
	}))
}

// modelsHandler returns the state of models.
func (inf *Infer) modelsHandler(c echo.Context) error {
	models, err := inf.Cfg.Search(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": errors.New("failed to search models: " + err.Error()),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"models": models,
		"count":  len(models),
	})
}
