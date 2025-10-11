// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
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

const faviconStr = `<svg fill="none" stroke="#37b" viewBox="0 0 164 200" xmlns="http://www.w3.org/2000/svg">
  <ellipse cx="82" cy="126" rx="35" ry="43" stroke-width="5"/>
  <ellipse cx="82" cy="122" rx="45" ry="55" stroke-width="6"/>
  <ellipse cx="82" cy="113" rx="59" ry="73" stroke-width="7"/>
  <ellipse cx="82" cy="100" rx="78" ry="96" stroke-width="8"/>
</svg>`

var favicon = []byte(faviconStr)

// NewEcho creates a new Echo server configured with Goinfer routes and middleware.
func (inf *Infer) NewEcho() *echo.Echo {
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
	if inf.Cfg.Origins != "" {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     strings.Split(inf.Cfg.Origins, ","),
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
	grp.GET("v1/models", inf.listModelsHandler)

	// ----- state -----------------
	grp.GET("health", func(_ echo.Context) error { return nil })
	grp.GET("logs/stream", inf.streamLogsHandler)
	grp.GET("props", inf.proxyToFirstRunningProcess)
	grp.GET("running", inf.listRunningProcessesHandler)
	grp.GET("unload", inf.unloadAllModelsHandler)

	// ----- /completion --------------
	grp.POST("completion", inf.completionHandler)  // llama.cpp API (legacy)
	grp.POST("completions", inf.completionHandler) // llama.cpp API
	grp.POST("v1/audio/speech", inf.proxyOAIHandler)
	grp.POST("v1/audio/transcriptions", inf.proxyOAIPostFormHandler)
	grp.POST("v1/chat/completions", inf.chatCompletionsHandler) // OpenAI API
	grp.POST("v1/completions", inf.proxyOAIHandler)

	// ----- /rerank ------------------
	grp.POST("rerank", inf.proxyOAIHandler)
	grp.POST("reranking", inf.proxyOAIHandler)
	grp.POST("v1/rerank", inf.proxyOAIHandler)
	grp.POST("v1/reranking", inf.proxyOAIHandler)

	// ----- /infill -----------------
	grp.POST("infill", inf.proxyOAIHandler)
	grp.POST("v1/embeddings", inf.proxyOAIHandler)

	// ---- /abort --------------
	grp.GET("abort", inf.abortHandler) // abort any running inference

	grp.GET("favicon.ico", func(c echo.Context) error { return c.Blob(http.StatusOK, "image/svg+xml", favicon) })
	grp.GET("favicon.svg", func(c echo.Context) error { return c.Blob(http.StatusOK, "image/svg+xml", favicon) })

	return e
}

// PrintRoutes creates a new Echo server configured with Goinfer routes and middleware.
func PrintRoutes(e *echo.Echo, addr string) {
	routes := e.Routes()

	wide := 0
	sort.Slice(routes, func(i, j int) bool {
		wide = max(wide, len(routes[i].Path))
		// sort by Method and by Path
		if routes[i].Method == routes[j].Method {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})

	if addr != "" && addr[0] == ':' {
		addr = "http://localhost" + addr
	} else {
		addr = "http://" + addr
	}
	wide += len(addr)

	var sb strings.Builder
	for _, route := range routes {
		line := fmt.Sprintf("%-5s %-*s %s\n", route.Method, wide, addr+route.Path, route.Name)
		_, _ = sb.WriteString(line)
	}

	slog.Info("routes:\n" + sb.String())
}

// configureAPIKeyAuth sets up API‑key authentication for a grp.
func (inf *Infer) configureAPIKeyAuth(grp *echo.Group) {
	if inf.Cfg.APIKey == "" {
		slog.Warn("Empty API key => disable API key security")
		return
	}

	grp.Use(middleware.KeyAuth(func(received_key string, _ echo.Context) (bool, error) {
		if received_key == inf.Cfg.APIKey {
			return true, nil
		}
		slog.Warn("Mismatched API key", "len(received)", len(received_key), "len(expected)", len(inf.Cfg.APIKey))
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

	if len(models) > 0 {
		response["models"] = models
	}

	return c.JSON(http.StatusOK, response)
}
