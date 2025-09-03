// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package server

import (
	"embed"
	"net/http"
	"strings"

	"github.com/LM4eu/goinfer/conf"
	"github.com/LM4eu/goinfer/errors"
	"github.com/LM4eu/goinfer/models"
	"github.com/LM4eu/goinfer/proxy"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

var (
	// The proxyManager is the shared ProxyManager instance for all server handlers.
	proxyManager = proxy.NewProxyManager()

	//go:embed all:dist
	embeddedFiles embed.FS
)

func NewEcho(cfg *conf.GoInferCfg, addr, services string) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Add middleware logger
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method} ${status} ${uri}  ${latency_human} ${remote_ip} ${error}\n",
	}))

	if l, ok := e.Logger.(*log.Logger); ok {
		l.SetHeader("[${time_rfc3339}] ${level}")
	}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     strings.Split(cfg.Server.Origins, ","),
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowMethods:     []string{http.MethodGet, http.MethodOptions, http.MethodPost},
		AllowCredentials: true,
	}))

	// Add unified error handling middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				return errors.ErrorHandler(err, c)
			}
			return nil
		}
	})

	configured := false

	// ------- Admin web frontend -------
	if strings.Contains(services, "admin") {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "dist",
			Index:      "index.html",
			Browse:     false,
			HTML5:      true,
			Filesystem: http.FS(embeddedFiles),
		}))

		configured = true
	}

	// ------------ Models ------------
	if strings.Contains(services, "model") {
		grp := e.Group("/model")
		setupAPIKeyAuth(grp, cfg, "model")
		grp.GET("/state", models.Dir(cfg.ModelsDir).Handler)

		configured = true
	}

	// ----- Inference (llama.cpp) -----
	if strings.Contains(services, "goinfer") {
		grp := e.Group("/completion")
		setupAPIKeyAuth(grp, cfg, "goinfer")
		grp.POST("", inferHandler)
		grp.GET("/abort", abortHandler)

		configured = true
	}

	// ----- Inference OpenAI API -----
	if strings.Contains(services, "openai") {
		grp := e.Group("/v1")
		grp.POST("/chat/completions", handleChatCompletions)
		setupAPIKeyAuth(grp, cfg, "openai")

		configured = true
	}

	if configured {
		return e
	}

	return nil
}

// setupAPIKeyAuth sets up API key authentication for a grp.
func setupAPIKeyAuth(grp *echo.Group, cfg *conf.GoInferCfg, service string) {
	apiKey := conf.GetAPIKey(cfg.Server.APIKeys, service)
	if apiKey != "" {
		grp.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
			return key == apiKey, nil
		}))
	}
}
