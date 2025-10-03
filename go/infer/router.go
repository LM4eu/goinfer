// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/LM4eu/goinfer/conf"
	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/llama-swap/proxy"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

// Infer manages proxying requests to the backend LLM engine.
type Infer struct {
	Cfg           *conf.Cfg
	ProxyMan      *proxy.ProxyManager
	isInferring   bool
	stopInferring bool
	mu            sync.Mutex
}

// NewEcho creates a new Echo server configured with Goinfer routes and middleware.
func (inf *Infer) NewEcho(addr string) *echo.Echo {
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
	if inf.Cfg.Server.Origins != "" {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     strings.Split(inf.Cfg.Server.Origins, ","),
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
				return gie.HandleErrorMiddleware(err, c)
			}
			return nil
		}
	})

	grp := e.Group("/")
	inf.configureAPIKeyAuth(grp)

	// ---- /models -------------------
	grp.GET("models", inf.modelsHandler)

	// ----- /completion --------------
	grp.POST("completion", inf.completionHandler) // legacy
	grp.POST("completions", inf.completionHandler)
	grp.POST("v1/chat/completions", inf.chatCompletionsHandler) // OpenAI API

	// ---- /abort --------------
	grp.GET("abort", inf.abortHandler) // abort all running inferences

	slog.Info("Listen", "GET", url(addr, "/models"))
	slog.Info("Listen", "POST", url(addr, "/completion (legacy)"))
	slog.Info("Listen", "POST", url(addr, "/completions"))
	slog.Info("Listen", "POST", url(addr, "/v1/chat/completions (OpenAI API)"))
	slog.Info("Listen", "GET", url(addr, "/abort (abort all running inferences)"))

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
func (inf *Infer) configureAPIKeyAuth(grp *echo.Group) {
	if inf.Cfg.Server.APIKey == "" {
		slog.Warn("Empty API key => disable API key security")
		return
	}

	grp.Use(middleware.KeyAuth(func(received_key string, _ echo.Context) (bool, error) {
		if received_key == inf.Cfg.Server.APIKey {
			return true, nil
		}
		slog.Warn("Mismatched API key", "len(received)", len(received_key), "len(expected)", len(inf.Cfg.Server.APIKey))
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
