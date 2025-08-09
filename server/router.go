package server

import (
	"embed"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/synw/goinfer/types"
)

//go:embed all:dist
var embeddedFiles embed.FS

func RunServer(conf types.WebServerConf, localMode bool, disableApiKey bool) {
	e := echo.New()
	e.HideBanner = true

	// logger
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method} ${status} ${uri}  ${latency_human} ${remote_ip} ${error}\n",
	}))

	if l, ok := e.Logger.(*log.Logger); ok {
		l.SetHeader("[${time_rfc3339}] ${level}")
	}

	// cors
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     conf.Origins,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowMethods:     []string{http.MethodGet, http.MethodOptions, http.MethodPost},
		AllowCredentials: true,
	}))

	if localMode {
		// static
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "dist",
			Index:      "index.html",
			Browse:     false,
			HTML5:      true,
			Filesystem: http.FS(embeddedFiles),
		}))
	}

	// inference
	inf := e.Group("/completion")
	if !disableApiKey {
		inf.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
			return key == conf.ApiKey, nil
		}))
	}

	inf.POST("", InferHandler)
	inf.GET("/abort", AbortHandler)

	// models
	mod := e.Group("/model")
	if !disableApiKey {
		mod.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
			return key == conf.ApiKey, nil
		}))
	}

	mod.GET("/state", ModelsStateHandler)
	mod.POST("/load", LoadModelHandler)
	mod.GET("/unload", UnloadModelHandler)

	// tasks
	tas := e.Group("/task")
	if !disableApiKey {
		tas.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
			return key == conf.ApiKey, nil
		}))
	}

	if conf.EnableApiOpenAi {
		// openai api
		oai := e.Group("/v1")
		if !disableApiKey {
			oai.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
				return key == conf.ApiKey, nil
			}))
		}

		oai.POST("/chat/completions", CreateCompletionHandler)
		oai.GET("/models", OpenAiListModels)
	}

	e.Start(conf.Port)
}
