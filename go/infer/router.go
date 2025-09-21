// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"embed"
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
	Cfg                         *conf.Cfg
	IsInferring                 bool
	ContinueInferringController bool
	mu                          sync.Mutex
}

//go:embed all:dist
var embeddedFiles embed.FS

// NewEcho creates a new Echo server configured with Goinfer routes and middleware.
func (inf *Infer) NewEcho(cfg *conf.Cfg, addr string,
	enableWebUI, enableModelsEndpoint, enableGoinferEndpoint, enableOpenAPIEndpoint bool,
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
	if enableWebUI {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "dist",
			Index:      "index.html",
			Browse:     false,
			HTML5:      true,
			Filesystem: http.FS(embeddedFiles),
		}))
		slog.Info("Listen", "GET", url(addr, "/"), "service", "webui")
	}

	// ------------ Models ------------
	if enableModelsEndpoint {
		grp := e.Group("/models")
		configureAPIKeyAuth(grp, cfg, "model")
		grp.GET("", inf.modelsHandler)
		slog.Info("Listen", "GET", url(addr, "/models"))
	}

	// ----- Inference (llama.cpp) -----
	if enableGoinferEndpoint {
		grp := e.Group("/goinfer")
		configureAPIKeyAuth(grp, cfg, "goinfer")
		grp.POST("", inf.inferHandler)
		grp.GET("/abort", inf.abortHandler)
		slog.Info("Listen", "POST", url(addr, "/goinfer"))
		slog.Info("Listen", "GET", url(addr, "/goinfer/abort"))
	}

	// ----- Inference OpenAI API -----
	if enableOpenAPIEndpoint {
		grp := e.Group("/v1")
		configureAPIKeyAuth(grp, cfg, "openai")
		grp.POST("/chat/completions", inf.handleChatCompletions)
		slog.Info("Listen", "POST", url(addr, "/v1/chat/completions"), "service", "openai")
	}

	return e
}

func url(addr, endpoint string) string {
	url := "http://"
	if addr != "" && addr[0] == ':' {
		url += "localhost"
	}
	return url + addr + endpoint
}

// configureAPIKeyAuth sets up API‑key authentication for a grp.
func configureAPIKeyAuth(grp *echo.Group, cfg *conf.Cfg, service string) {
	// Select the API key with preference order
	key, exists := cfg.Server.APIKeys[service]
	if !exists {
		key, exists = cfg.Server.APIKeys["user"]
		if !exists {
			key, exists = cfg.Server.APIKeys["admin"]
			if !exists {
				slog.Warn("No API key => disable API key security", "service", service)
				return
			}
		}
	}

	if key == "" {
		slog.Warn("Empty API key => disable API key for service", "service", service)
		return
	}

	grp.Use(middleware.KeyAuth(func(received_key string, c echo.Context) (bool, error) {
		if received_key == key {
			return true, nil
		}

		slog.Warn("Received API key is NOT the configured one for service", "service", service)
		return false, nil
	}))
}

// modelsHandler returns the state of models.
func (inf *Infer) modelsHandler(c echo.Context) error {
	models, err := inf.Cfg.ListModels()

	response := map[string]any{
		"count": len(models),
	}

	if err != nil {
		response["error"] = err.Error()
	}

	response["models"] = models

	return c.JSON(http.StatusOK, response)
}
