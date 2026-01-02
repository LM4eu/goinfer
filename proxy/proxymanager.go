// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LynxAIeu/garcon/gerr"
	"github.com/LynxAIeu/goinfer/conf"
	"github.com/LynxAIeu/goinfer/event"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	PROFILE_SPLIT_CHAR = ":"
)

type proxyCtxKey string

type ProxyManager struct {
	shutdownCtx    context.Context
	muxLogger      *LogMonitor
	metricsMonitor *metricsMonitor
	ginEngine      *gin.Engine
	proxyLogger    *LogMonitor
	upstreamLogger *LogMonitor
	cfg            *conf.Cfg
	processGroups  map[string]*ProcessGroup
	shutdownCancel context.CancelFunc
	buildDate      string
	commit         string
	version        string
	sync.Mutex
}

func New(cfg *conf.Cfg) *ProxyManager {
	// set up loggers
	stdoutLogger := NewLogMonitorWriter(os.Stdout)
	upstreamLogger := NewLogMonitorWriter(stdoutLogger)
	proxyLogger := NewLogMonitorWriter(stdoutLogger)

	if cfg.Swap.LogRequests {
		proxyLogger.Warn("LogRequests configuration is deprecated. Use logLevel instead.")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Swap.LogLevel)) {
	case "debug":
		proxyLogger.SetLogLevel(LevelDebug)
		upstreamLogger.SetLogLevel(LevelDebug)
	case "info":
		proxyLogger.SetLogLevel(LevelInfo)
		upstreamLogger.SetLogLevel(LevelInfo)
	case "warn":
		proxyLogger.SetLogLevel(LevelWarn)
		upstreamLogger.SetLogLevel(LevelWarn)
	case "error":
		proxyLogger.SetLogLevel(LevelError)
		upstreamLogger.SetLogLevel(LevelError)
	default:
		proxyLogger.SetLogLevel(LevelInfo)
		upstreamLogger.SetLogLevel(LevelInfo)
	}

	// see: https://go.dev/src/time/format.go
	timeFormats := map[string]string{
		"ansic":       time.ANSIC,
		"unixdate":    time.UnixDate,
		"rubydate":    time.RubyDate,
		"rfc822":      time.RFC822,
		"rfc822z":     time.RFC822Z,
		"rfc850":      time.RFC850,
		"rfc1123":     time.RFC1123,
		"rfc1123z":    time.RFC1123Z,
		"rfc3339":     time.RFC3339,
		"rfc3339nano": time.RFC3339Nano,
		"kitchen":     time.Kitchen,
		"stamp":       time.Stamp,
		"stampmilli":  time.StampMilli,
		"stampmicro":  time.StampMicro,
		"stampnano":   time.StampNano,
	}

	if timeFormat, ok := timeFormats[strings.ToLower(strings.TrimSpace(cfg.Swap.LogTimeFormat))]; ok {
		proxyLogger.SetLogTimeFormat(timeFormat)
		upstreamLogger.SetLogTimeFormat(timeFormat)
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	var maxMetrics int
	if cfg.Swap.MetricsMaxInMemory <= 0 {
		maxMetrics = 1000 // Default fallback
	} else {
		maxMetrics = cfg.Swap.MetricsMaxInMemory
	}

	pm := &ProxyManager{
		cfg:       cfg,
		ginEngine: gin.New(),

		proxyLogger:    proxyLogger,
		muxLogger:      stdoutLogger,
		upstreamLogger: upstreamLogger,

		metricsMonitor: newMetricsMonitor(proxyLogger, maxMetrics),

		processGroups: make(map[string]*ProcessGroup),

		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,

		buildDate: "unknown",
		commit:    "abcd1234",
		version:   "0",
	}

	// create the process groups
	for groupID := range cfg.Swap.Groups {
		processGroup := NewProcessGroup(groupID, cfg.Swap, proxyLogger, upstreamLogger)
		pm.processGroups[groupID] = processGroup
	}

	pm.setupGinEngine()

	// run any startup hooks
	if len(cfg.Swap.Hooks.OnStartup.Preload) > 0 {
		// do it in the background, don't block startup -- not sure if good idea yet
		go func() {
			discardWriter := &DiscardWriter{}
			for _, realModelName := range cfg.Swap.Hooks.OnStartup.Preload {
				proxyLogger.Infof("Preloading model: %s", realModelName)
				processGroup, _, err := pm.swapProcessGroup(realModelName)

				if err != nil {
					event.Emit(ModelPreloadedEvent{
						ModelName: realModelName,
						Success:   false,
					})
					proxyLogger.Errorf("Failed to preload model %s: %v", realModelName, err)
					continue
				} else {
					req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
					processGroup.proxyRequest(realModelName, discardWriter, req)
					event.Emit(ModelPreloadedEvent{
						ModelName: realModelName,
						Success:   true,
					})
				}
			}
		}()
	}

	return pm
}

func (pm *ProxyManager) setupGinEngine() {
	pm.ginEngine.Use(func(c *gin.Context) {
		// don't log the Wake on Lan proxy health check
		if c.Request.URL.Path == "/wol-health" {
			c.Next()
			return
		}

		// Start timer
		start := time.Now()

		// capture these because /upstream/:model rewrites them in c.Next()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Stop timer
		duration := time.Since(start)

		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()

		pm.proxyLogger.Infof("Request %s \"%s %s %s\" %d %d \"%s\" %v",
			clientIP,
			method,
			path,
			c.Request.Proto,
			statusCode,
			bodySize,
			c.Request.UserAgent(),
			duration,
		)
	})

	// see: issue: #81, #77 and #42 for CORS issues
	// respond with permissive OPTIONS for any endpoint
	pm.ginEngine.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

			// allow whatever the client requested by default
			if headers := c.Request.Header.Get("Access-Control-Request-Headers"); headers != "" {
				sanitized := SanitizeAccessControlRequestHeaderValues(headers)
				c.Header("Access-Control-Allow-Headers", sanitized)
			} else {
				c.Header(
					"Access-Control-Allow-Headers",
					"Content-Type, Authorization, Accept, X-Requested-With",
				)
			}
			c.Header("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Set up routes using the Gin engine
	pm.ginEngine.POST("/v1/chat/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d1/chat/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a1/chat/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A1/chat/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })
	// Support legacy /v1/completions api, see issue #12
	pm.ginEngine.POST("/v1/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d1/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a1/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A1/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })

	// Support embeddings and reranking
	pm.ginEngine.POST("/v1/embeddings", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d1/embeddings", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a1/embeddings", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A1/embeddings", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })

	// llama-server's /reranking endpoint + aliases
	pm.ginEngine.POST("/reranking", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/rerank", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/v1/rerank", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d1/rerank", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a1/rerank", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A1/rerank", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })
	pm.ginEngine.POST("/v1/reranking", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d1/reranking", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a1/reranking", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A1/reranking", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })

	// llama-server's /infill endpoint for code infilling
	pm.ginEngine.POST("/v1/infill", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d1/infill", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a1/infill", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A1/infill", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })

	// llama-server's /completion endpoint  (legacy)
	pm.ginEngine.POST("/completion", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d/completion", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a/completion", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A/completion", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })

	// llama-server's /completions endpoint  (official)
	pm.ginEngine.POST("/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A/completions", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })

	// Support audio/speech endpoint
	pm.ginEngine.POST("/v1/audio/speech", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, false) })
	pm.ginEngine.POST("/d1/audio/speech", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, false) })
	pm.ginEngine.POST("/a1/audio/speech", func(c *gin.Context) { pm.ProxyOAIHandler(c, false, true) })
	pm.ginEngine.POST("/A1/audio/speech", func(c *gin.Context) { pm.ProxyOAIHandler(c, true, true) })
	pm.ginEngine.POST("/v1/audio/transcriptions", func(c *gin.Context) { pm.proxyOAIPostFormHandler(c, false, false) })
	pm.ginEngine.POST("/d1/audio/transcriptions", func(c *gin.Context) { pm.proxyOAIPostFormHandler(c, true, false) })
	pm.ginEngine.POST("/a1/audio/transcriptions", func(c *gin.Context) { pm.proxyOAIPostFormHandler(c, false, true) })
	pm.ginEngine.POST("/A1/audio/transcriptions", func(c *gin.Context) { pm.proxyOAIPostFormHandler(c, true, true) })

	pm.ginEngine.GET("/v1/models", pm.ListModelsHandler)
	pm.ginEngine.GET("/d1/models", pm.ListModelsHandler)
	pm.ginEngine.GET("/a1/models", pm.ListModelsHandler)
	pm.ginEngine.GET("/A1/models", pm.ListModelsHandler)
	pm.ginEngine.GET("/gi/models", pm.giListModelsHandler)

	// in proxymanager_loghandlers.go
	pm.ginEngine.GET("/logs", pm.sendLogsHandlers)
	pm.ginEngine.GET("/logs/stream", pm.StreamLogsHandler)
	pm.ginEngine.GET("/logs/stream/:logMonitorID", pm.StreamLogsHandler)

	/**
	 * User Interface Endpoints
	 */
	// pm.ginEngine.GET("/", func(c *gin.Context) {
	// 	c.Redirect(http.StatusFound, "/ui")
	// })
	pm.ginEngine.GET("/", pm.ProxyToFirstRunningProcess)
	pm.ginEngine.GET("/props", pm.ProxyToFirstRunningProcess)

	pm.ginEngine.GET("/upstream", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui/models")
	})
	pm.ginEngine.Any("/upstream/*upstreamPath", pm.proxyToUpstream)
	pm.ginEngine.GET("/unload", pm.UnloadAllModelsHandler)
	pm.ginEngine.GET("/running", pm.ListRunningProcessesHandler)
	pm.ginEngine.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// see cmd/wol-proxy/wol-proxy.go, not logged
	pm.ginEngine.GET("/wol-health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	pm.ginEngine.GET("/favicon.ico", func(c *gin.Context) {
		if data, err := reactStaticFS.ReadFile("ui_dist/favicon.ico"); err == nil {
			c.Data(http.StatusOK, "image/x-icon", data)
		} else {
			c.String(http.StatusInternalServerError, err.Error())
		}
	})

	reactFS, err := GetReactFS()
	if err != nil {
		pm.proxyLogger.Errorf("Failed to load React filesystem: %v", err)
	} else {
		// serve files that exist under /ui/*
		pm.ginEngine.StaticFS("/ui", reactFS)

		// server SPA for UI under /ui/*
		pm.ginEngine.NoRoute(func(c *gin.Context) {
			if !strings.HasPrefix(c.Request.URL.Path, "/ui") {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			file, err := reactFS.Open("index.html")
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
			defer file.Close()
			http.ServeContent(c.Writer, c.Request, "index.html", time.Now(), file)
		})
	}

	// see: proxymanager_api.go
	// add API handler functions
	addApiHandlers(pm)

	// Disable console color for testing
	gin.DisableConsoleColor()
}

// ServeHTTP implements http.Handler interface.
func (pm *ProxyManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pm.ginEngine.ServeHTTP(w, r)
}

// StopProcesses acquires a lock and stops all running upstream processes.
// This is the public method safe for concurrent calls.
// Unlike Shutdown, this method only stops the processes but doesn't perform
// a complete shutdown, allowing for process replacement without full termination.
func (pm *ProxyManager) StopProcesses(strategy StopStrategy) {
	pm.Lock()
	defer pm.Unlock()

	// stop Processes in parallel
	var wg sync.WaitGroup
	for _, processGroup := range pm.processGroups {
		wg.Add(1)
		go func(processGroup *ProcessGroup) {
			defer wg.Done()
			processGroup.StopProcesses(strategy)
		}(processGroup)
	}

	wg.Wait()
}

// Shutdown stops all processes managed by this ProxyManager.
func (pm *ProxyManager) Shutdown() {
	pm.Lock()
	defer pm.Unlock()

	pm.proxyLogger.Debug("Shutdown() called in proxy manager")

	var wg sync.WaitGroup
	// Send shutdown signal to all process in groups
	for _, processGroup := range pm.processGroups {
		wg.Add(1)
		go func(processGroup *ProcessGroup) {
			defer wg.Done()
			processGroup.Shutdown()
		}(processGroup)
	}
	wg.Wait()
	pm.shutdownCancel()
}

func (pm *ProxyManager) swapProcessGroup(requestedModel string) (*ProcessGroup, string, error) {
	// de-alias the real model name and get a real one
	realModelName, found := pm.cfg.Swap.RealModelName(requestedModel)
	if !found {
		return nil, realModelName, fmt.Errorf("could not find real modelID for %s", requestedModel)
	}

	processGroup := pm.findGroupByModelName(realModelName)
	if processGroup == nil {
		return nil, realModelName, fmt.Errorf("could not find process group for model %s", requestedModel)
	}

	if processGroup.exclusive {
		pm.proxyLogger.Debugf("Exclusive mode for group %s, stopping other process groups", processGroup.id)
		for groupId, otherGroup := range pm.processGroups {
			if groupId != processGroup.id && !otherGroup.persistent {
				otherGroup.StopProcesses(StopWaitForInflightRequest)
			}
		}
	}

	return processGroup, realModelName, nil
}

func (pm *ProxyManager) ListModelsHandler(c *gin.Context) {
	data := make([]gin.H, 0, len(pm.cfg.Swap.Models))
	createdTime := time.Now().Unix()

	for id, modelConfig := range pm.cfg.Swap.Models {
		if modelConfig.Unlisted {
			continue
		}

		newRecord := func(modelId string) gin.H {
			record := gin.H{
				"id":       modelId,
				"object":   "model",
				"created":  createdTime,
				"owned_by": "llama-swap",
			}

			if name := strings.TrimSpace(modelConfig.Name); name != "" {
				record["name"] = name
			}
			if desc := strings.TrimSpace(modelConfig.Description); desc != "" {
				record["description"] = desc
			}

			// Add metadata if present
			if len(modelConfig.Metadata) > 0 {
				record["meta"] = gin.H{
					"llamaswap": modelConfig.Metadata,
				}
			}
			return record
		}

		data = append(data, newRecord(id))

		// Include aliases
		if pm.cfg.Swap.IncludeAliasesInList {
			for _, alias := range modelConfig.Aliases {
				if alias = strings.TrimSpace(alias); alias != "" {
					data = append(data, newRecord(alias))
				}
			}
		}
	}

	// Sort by the "id" key
	sort.Slice(data, func(i, j int) bool {
		si, _ := data[i]["id"].(string)
		sj, _ := data[j]["id"].(string)
		return si < sj
	})

	// Set CORS headers if origin exists
	if origin := c.GetHeader("Origin"); origin != "" {
		c.Header("Access-Control-Allow-Origin", origin)
	}

	// Use gin's JSON method which handles content-type and encoding
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func (pm *ProxyManager) giListModelsHandler(c *gin.Context) {
	models := pm.cfg.ListModels()

	if len(models) == 0 {
		c.String(http.StatusNoContent, `{"count":0, "models":[]}`)
		return
	}

	// Use gin's JSON method which handles content-type and encoding
	c.JSON(http.StatusOK, gin.H{"count": len(models), "models": models})
}

func (pm *ProxyManager) proxyToUpstream(c *gin.Context) {
	upstreamPath := c.Param("upstreamPath")

	// split the upstream path by / and search for the model name
	parts := strings.Split(strings.TrimSpace(upstreamPath), "/")
	if len(parts) == 0 {
		pm.sendErrorResponse(c, http.StatusBadRequest, "model id required in path")
		return
	}

	modelFound := false
	searchModelName := ""
	var modelName, remainingPath string
	for i, part := range parts {
		if parts[i] == "" {
			continue
		}

		if searchModelName == "" {
			searchModelName = part
		} else {
			searchModelName = searchModelName + "/" + parts[i]
		}

		if realName, ok := pm.cfg.Swap.RealModelName(searchModelName); ok {
			modelName = realName
			remainingPath = "/" + strings.Join(parts[i+1:], "/")
			modelFound = true

			// Check if this is exactly a model name with no additional path
			// and doesn't end with a trailing slash
			if remainingPath == "/" && !strings.HasSuffix(upstreamPath, "/") {
				// Build new URL with query parameters preserved
				newPath := "/upstream/" + searchModelName + "/"
				if c.Request.URL.RawQuery != "" {
					newPath += "?" + c.Request.URL.RawQuery
				}

				// Use 308 for non-GET/HEAD requests to preserve method
				if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
					c.Redirect(http.StatusMovedPermanently, newPath)
				} else {
					c.Redirect(http.StatusPermanentRedirect, newPath)
				}
				return
			}
			break
		}
	}

	if !modelFound {
		pm.sendErrorResponse(c, http.StatusBadRequest, "model id required in path")
		return
	}

	processGroup, realModelName, err := pm.swapProcessGroup(modelName)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error swapping process group: "+err.Error())
		return
	}

	// rewrite the path
	originalPath := c.Request.URL.Path
	c.Request.URL.Path = remainingPath

	// attempt to record metrics if it is a POST request
	if pm.metricsMonitor != nil && c.Request.Method == http.MethodPost {
		err := pm.metricsMonitor.wrapHandler(realModelName, c.Writer, c.Request, processGroup.proxyRequest)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying metrics wrapped request: "+err.Error())
			pm.proxyLogger.Errorf("Error proxying wrapped upstream request for model %s, path=%s", realModelName, originalPath)
			return
		}
	} else {
		err := processGroup.proxyRequest(realModelName, c.Writer, c.Request)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying request: "+err.Error())
			pm.proxyLogger.Errorf("Error proxying upstream request for model %s, path=%s", realModelName, originalPath)
			return
		}
	}
}

func (pm *ProxyManager) ProxyOAIHandler(c *gin.Context, download, agentSmith bool) {
	if download || agentSmith {
		if c.Request.URL.Path[2] == '/' {
			c.Request.URL.Path = c.Request.URL.Path[2:]
		} else {
			c.Request.URL.Path = "/v" + c.Request.URL.Path[2:]
		}
	}

	bodyBytes, realModelName, _, _, err := pm.getSetModel(c.Request.Body, download, agentSmith)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	if realModelName == "" {
		c.JSON(http.StatusBadRequest, gerr.New(gerr.Invalid, "no model provided and no model loaded"))
		return
	}

	processGroup, realModelName, err := pm.swapProcessGroup(realModelName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gerr.Wrap(err, gerr.Invalid, "error swapping process group", "model", realModelName))
		return
	}

	// issue #174 strip parameters from the JSON body
	stripParams, err := pm.cfg.Swap.Models[realModelName].Filters.SanitizedStripParams()
	if err != nil { // just log it and continue
		pm.proxyLogger.Warnf("Error sanitizing strip params string: %s, %s", pm.cfg.Swap.Models[realModelName].Filters.StripParams, err.Error())
	} else {
		for _, param := range stripParams {
			pm.proxyLogger.Debugf("<%s> stripping param: %s", realModelName, param)
			bodyBytes, err = sjson.DeleteBytes(bodyBytes, param)
			if err != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("error deleting parameter %s from request", param))
				return
			}
		}
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// dechunk it as we already have all the body bytes see issue #11
	c.Request.Header.Del("Transfer-Encoding")
	c.Request.Header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
	c.Request.ContentLength = int64(len(bodyBytes))

	// issue #366 extract values that downstream handlers may need
	isStreaming := gjson.GetBytes(bodyBytes, "stream").Bool()
	ctx := context.WithValue(c.Request.Context(), proxyCtxKey("streaming"), isStreaming)
	ctx = context.WithValue(ctx, proxyCtxKey("model"), realModelName)
	c.Request = c.Request.WithContext(ctx)

	if pm.metricsMonitor != nil && c.Request.Method == http.MethodPost {
		err = pm.metricsMonitor.wrapHandler(realModelName, c.Writer, c.Request, processGroup.proxyRequest)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying metrics wrapped request: "+err.Error())
			pm.proxyLogger.Errorf("Error Proxying Metrics Wrapped Request for processGroup %s and model %s", processGroup.id, realModelName)
		}
		return
	}

	err = processGroup.proxyRequest(realModelName, c.Writer, c.Request)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying request: "+err.Error())
		pm.proxyLogger.Errorf("Error Proxying Request for processGroup %s and model %s", processGroup.id, realModelName)
		return
	}
}

func (pm *ProxyManager) proxyOAIPostFormHandler(c *gin.Context, download, agentSmith bool) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory, larger files go to tmp disk
		pm.sendErrorResponse(c, http.StatusBadRequest, "error parsing multipart form: "+err.Error())
		return
	}

	// Get model parameter from the form
	requestedModel := c.Request.FormValue("model")
	fixed, _, _ := pm.fixModelName(requestedModel, download, agentSmith)

	processGroup, realModelName, err := pm.swapProcessGroup(fixed)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error swapping process group: "+err.Error())
		return
	}

	// We need to reconstruct the multipart form in any case since the body is consumed
	// Create a new buffer for the reconstructed request
	var requestBuffer bytes.Buffer
	multipartWriter := multipart.NewWriter(&requestBuffer)

	// Copy all form values
	for key, values := range c.Request.MultipartForm.Value {
		for _, val := range values {
			// If this is the model field and we have a profile, use just the model name
			if key == "model" {
				// # issue #69 allow custom model names to be sent to upstream
				useModelName := pm.cfg.Swap.Models[realModelName].UseModelName

				if useModelName != "" {
					val = useModelName
				} else {
					val = requestedModel
				}
			}
			field, er := multipartWriter.CreateFormField(key)
			if er != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error recreating form field")
				return
			}
			_, err = field.Write([]byte(val))
			if err != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error writing form field")
				return
			}
		}
	}

	// Copy all files from the original request
	for key, fileHeaders := range c.Request.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			formFile, er := multipartWriter.CreateFormFile(key, fileHeader.Filename)
			if er != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error recreating form file")
				return
			}

			file, er := fileHeader.Open()
			if er != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error opening uploaded file")
				return
			}

			if _, er = io.Copy(formFile, file); er != nil {
				file.Close()
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error copying file data")
				return
			}
			file.Close()
		}
	}

	// Close the multipart writer to finalize the form
	err = multipartWriter.Close()
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error finalizing multipart form")
		return
	}

	// Create a new request with the reconstructed form data
	modifiedReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		c.Request.Method,
		c.Request.URL.String(),
		&requestBuffer,
	)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error creating modified request")
		return
	}

	// Copy the headers from the original request
	modifiedReq.Header = c.Request.Header.Clone()
	modifiedReq.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	// set the content length of the body
	modifiedReq.Header.Set("Content-Length", strconv.Itoa(requestBuffer.Len()))
	modifiedReq.ContentLength = int64(requestBuffer.Len())

	// Use the modified request for proxying
	if err := processGroup.proxyRequest(realModelName, c.Writer, modifiedReq); err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying request: "+err.Error())
		pm.proxyLogger.Errorf("Error Proxying Request for processGroup %s and model %s", processGroup.id, realModelName)
		return
	}
}

func (pm *ProxyManager) sendErrorResponse(c *gin.Context, statusCode int, message string) {
	acceptHeader := c.GetHeader("Accept")

	if strings.Contains(acceptHeader, "application/json") {
		c.JSON(statusCode, gin.H{"error": message})
	} else {
		c.String(statusCode, message)
	}
}

func (pm *ProxyManager) UnloadAllModelsHandler(c *gin.Context) {
	pm.StopProcesses(StopImmediately)
	c.String(http.StatusOK, "OK")
}

func (pm *ProxyManager) ListRunningProcessesHandler(ctx *gin.Context) {
	ctx.Header("Content-Type", "application/json")

	// Put the results under the `running` key.
	response := gin.H{
		"running": pm.listRunningProcesses(),
	}

	ctx.JSON(http.StatusOK, response) // Always return 200 OK
}

func (pm *ProxyManager) listRunningProcesses() []gin.H {
	count := pm.countRunningProcesses()
	runningProcesses := make([]gin.H, 0, count) // Default to an empty response.

	for _, processGroup := range pm.processGroups {
		for _, process := range processGroup.processes {
			if process.CurrentState() == StateReady {
				runningProcesses = append(runningProcesses, gin.H{
					"model": process.ID,
					"state": process.state,
				})
			}
		}
	}
	return runningProcesses
}

func (pm *ProxyManager) countRunningProcesses() int {
	count := 0

	for _, processGroup := range pm.processGroups {
		for _, process := range processGroup.processes {
			if process.CurrentState() == StateReady {
				count++
			}
		}
	}

	return count
}

func (pm *ProxyManager) findGroupByModelName(modelName string) *ProcessGroup {
	for _, group := range pm.processGroups {
		if group.HasMember(modelName) {
			return group
		}
	}
	return nil
}

func (pm *ProxyManager) SetVersion(buildDate, commit, version string) {
	pm.Lock()
	defer pm.Unlock()
	pm.buildDate = buildDate
	pm.commit = commit
	pm.version = version
}

func (pm *ProxyManager) firstRunningProcess() string {
	var starting string
	for _, processGroup := range pm.processGroups {
		for _, process := range processGroup.processes {
			if process.state == StateReady {
				return process.ID
			}
			if process.state == StateStarting {
				starting = process.ID
			}
		}
	}
	return starting
}

// ProxyToFirstRunningProcess forwards the request to a running process (llama-server).
func (pm *ProxyManager) ProxyToFirstRunningProcess(c *gin.Context) {
	var starting *Process
	for _, processGroup := range pm.processGroups {
		for _, process := range processGroup.processes {
			if process.state == StateReady {
				process.ProxyRequest(c.Writer, c.Request)
				return
			}
			if process.state == StateStarting {
				starting = process
			}
		}
	}
	if starting != nil {
		starting.ProxyRequest(c.Writer, c.Request)
	}
	pm.sendErrorResponse(c, http.StatusInternalServerError, "No model currently running. Please select a model.")
}
