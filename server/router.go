package server

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/synw/goinfer/conf"
)

//go:embed all:dist
var embeddedFiles embed.FS

func RunServers(cfg conf.GoInferConf) {
	var wg sync.WaitGroup
	wg.Add(len(cfg.Server.Ports))

	for port, services := range cfg.Server.Ports {
		e := newEcho(cfg, port, services)

		if cfg.Verbose {
			fmt.Println("-----------------------------")
			fmt.Println("Starting http server:")
			fmt.Println("- services: ", services)
			fmt.Println("- port:     ", port)
			fmt.Println("- origins:  ", cfg.Server.Origins)
		}

		go start(e, port)
	}

	wg.Wait()
	fmt.Println("All http servers have stoped")
}

func newEcho(cfg conf.GoInferConf, port, services string) *echo.Echo {
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

	// ------- Admin web frontend -------
	if strings.Contains(services, "admin") {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "dist",
			Index:      "index.html",
			Browse:     false,
			HTML5:      true,
			Filesystem: http.FS(embeddedFiles),
		}))
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
		dir := ModelsDir(cfg.ModelsDir)
		grp.GET("/state", dir.ModelsStateHandler)
	}

	// ----- Inference (llama.cpp) -----
	if strings.Contains(services, "llama") {
		grp := e.Group("/completion")
		apiKey := conf.ApiKey(cfg.Server.ApiKeys, "goinfer")
		if apiKey != "" {
			grp.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
				return key == apiKey, nil
			}))
		}
		grp.POST("", InferHandler)
		grp.GET("/abort", AbortLlamaHandler)
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
	}

	return e
}

func start(e *echo.Echo, port string) {
	err := e.Start("localhost:" + port)
	if err != nil {
		fmt.Printf("WARNING e.Start(localhost:%s) %v\n", port, err)
	}
}
