package server

import (
	"embed"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/synw/goinfer/conf"
)

//go:embed all:dist
var embeddedFiles embed.FS

func RunServer(cfg conf.ServerConf, localMode bool, disableApiKey bool) {
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
		AllowOrigins:     strings.Split(cfg.Origins, ","),
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowMethods:     []string{http.MethodGet, http.MethodOptions, http.MethodPost},
		AllowCredentials: true,
	}))

	if localMode {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "dist",
			Index:      "index.html",
			Browse:     false,
			HTML5:      true,
			Filesystem: http.FS(embeddedFiles),
		}))
	}

	// // ------------ Models ------------
	//
	// mod := e.Group("/model")
	//
	//	if !disableApiKey {
	//		mod.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
	//			return key == cfg.ApiKey, nil
	//		}))
	//	}
	//
	// mod.GET("/state", ModelsStateHandler)
	// mod.POST("/start", StartLlamaHandler)
	// mod.GET("/stop", StopLlamaHandler)
	//
	// // ----- Inference (llama.cpp) -----
	//
	// inf := e.Group("/completion")
	//
	//	if !disableApiKey {
	//		inf.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
	//			return key == cfg.ApiKey, nil
	//		}))
	//	}
	//
	// inf.POST("", InferHandler)
	// inf.GET("/abort", AbortLlamaHandler)
	//
	// // ----- Inference OpenAI API -----
	//
	//	if cfg.EnableOpenAiAPI {
	//		oai := e.Group("/v1")
	//		if !disableApiKey {
	//			oai.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
	//				return key == cfg.ApiKey, nil
	//			}))
	//		}
	//
	//		oai.POST("/chat/completions", CreateCompletionHandler)
	//		oai.GET("/models", OpenAiListModels)
	//	}
	//
	// err := e.Start("localhost:" + cfg.Port)
	//
	//	if err != nil {
	//		fmt.Println(err.Error())
	//	}
}
