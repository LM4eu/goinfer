// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"embed"
	"fmt"
	"net/http"
	"strings"

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
}

//go:embed all:dist
var embeddedFiles embed.FS

func (inf *Infer) NewEcho(cfg *conf.GoInferCfg, addr string,
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
				return gie.ErrorHandler(err, c)
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
		fmt.Printf("INF: Listen GET %s/ (web UI)\n", addr)
	}

	// ------------ Models ------------
	if enableModelsEndpoint {
		grp := e.Group("/models")
		setupAPIKeyAuth(grp, cfg, "model")
		grp.GET("", inf.modelsHandler)
		fmt.Printf("INF: Listen GET %s/models (model files)\n", addr)
	}

	// ----- Inference (llama.cpp) -----
	if enableGoinferEndpoint {
		grp := e.Group("/goinfer")
		setupAPIKeyAuth(grp, cfg, "goinfer")
		grp.POST("", inf.inferHandler)
		grp.GET("/abort", inf.abortHandler)
		fmt.Printf("INF: Listen POST %s/goinfer (inference)\n", addr)
		fmt.Printf("INF: Listen GET  %s/goinfer/abort\n", addr)
	}

	// ----- Inference OpenAI API -----
	if enableOpenAPIEndpoint {
		grp := e.Group("/v1")
		grp.POST("/chat/completions", inf.handleChatCompletions)
		setupAPIKeyAuth(grp, cfg, "openai")
		fmt.Printf("INF: Listen POST %s/v1/chat/completions (inference)\n", addr)
	}

	return e
}

// setupAPIKeyAuth sets up API key authentication for a grp.
func setupAPIKeyAuth(grp *echo.Group, cfg *conf.GoInferCfg, service string) {
	// Select the API key with preference order
	key, exists := cfg.Server.APIKeys[service]
	if !exists {
		key, exists = cfg.Server.APIKeys["user"]
		if !exists {
			key, exists = cfg.Server.APIKeys["admin"]
			if !exists {
				fmt.Printf("WRN: No API key for %q, neither for user, nor admin => disable API key security\n", service)
				return
			}
		}
	}

	if key == "" {
		fmt.Printf("WRN: Empty API key => disable API key for %q\n", service)
		return
	}

	grp.Use(middleware.KeyAuth(func(received_key string, c echo.Context) (bool, error) {
		if received_key == key {
			return true, nil
		}

		fmt.Printf("WRN: Received API key is NOT the configured one for %q\n", service)
		return false, nil
	}))
}

// modelsHandler returns the state of models.
func (inf *Infer) modelsHandler(c echo.Context) error {
	models, err := inf.Cfg.Search()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("failed to search models: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"models": models,
		"count":  len(models),
	})
}
