package server

import (
	"embed"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/synw/goinfer/conf"
	"github.com/synw/goinfer/models"
)

//go:embed all:dist
var embeddedFiles embed.FS

func NewEchoServer(cfg conf.GoInferConf, addr, services string) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// logger
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

	atLeastOneService := false

	// ------- Admin web frontend -------
	if strings.Contains(services, "admin") {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "dist",
			Index:      "index.html",
			Browse:     false,
			HTML5:      true,
			Filesystem: http.FS(embeddedFiles),
		}))
		atLeastOneService = true
	}

	// ------------ Models ------------
	if strings.Contains(services, "model") {
		grp := e.Group("/model")
		apiKey := conf.ApiKey(cfg.Server.ApiKeys, "model")
		if apiKey != "" {
			grp.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
				return key == apiKey, nil
			}))
		}
		grp.GET("/state", models.Dir(cfg.ModelsDir).StateHandler)
		atLeastOneService = true
	}

	// ----- Inference (llama.cpp) -----
	if strings.Contains(services, "goinfer") {
		grp := e.Group("/completion")
		apiKey := conf.ApiKey(cfg.Server.ApiKeys, "goinfer")
		if apiKey != "" {
			grp.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
				return key == apiKey, nil
			}))
		}
		grp.POST("", InferHandler)
		grp.GET("/abort", AbortLlamaHandler)
		atLeastOneService = true
	}

	// ----- Inference OpenAI API -----
	if strings.Contains(services, "openai") {
		oai := e.Group("/v1")
		apiKey := conf.ApiKey(cfg.Server.ApiKeys, "openai")
		if apiKey != "" {
			oai.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
				return key == apiKey, nil
			}))
		}
		// oai.POST("/chat/completions", CreateCompletionHandler)
		// oai.GET("/models", OpenAiListModels)
		atLeastOneService = true
	}

	if atLeastOneService {
		return e
	}
	return nil
}
