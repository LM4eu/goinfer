// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LynxAIeu/goinfer/conf"
	"github.com/LynxAIeu/goinfer/event"
	"github.com/LynxAIeu/goinfer/proxy/config"
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
	metricsMonitor *metricsMonitor
	ginEngine      *gin.Engine
	proxyLogger    *LogMonitor
	upstreamLogger *LogMonitor
	muxLogger      *LogMonitor
	processGroups  map[string]*ProcessGroup
	cfg            *conf.Cfg
	shutdownCancel context.CancelFunc
	peerProxy      *PeerProxy
	buildDate      string
	commit         string
	version        string
	sync.Mutex
}

func New(cfg *conf.Cfg) *ProxyManager {
	// set up loggers

	var muxLogger, upstreamLogger, proxyLogger *LogMonitor
	switch cfg.Swap.LogToStdout {
	case config.LogToStdoutNone:
		muxLogger = NewLogMonitorWriter(io.Discard)
		upstreamLogger = NewLogMonitorWriter(io.Discard)
		proxyLogger = NewLogMonitorWriter(io.Discard)
	case config.LogToStdoutBoth:
		muxLogger = NewLogMonitorWriter(os.Stdout)
		upstreamLogger = NewLogMonitorWriter(muxLogger)
		proxyLogger = NewLogMonitorWriter(muxLogger)
	case config.LogToStdoutUpstream:
		muxLogger = NewLogMonitorWriter(os.Stdout)
		upstreamLogger = NewLogMonitorWriter(muxLogger)
		proxyLogger = NewLogMonitorWriter(io.Discard)
	default:
		// same as config.LogToStdoutProxy
		// helpful because some old tests create a config.Config directly and it
		// may not have LogToStdout set explicitly
		muxLogger = NewLogMonitorWriter(os.Stdout)
		upstreamLogger = NewLogMonitorWriter(io.Discard)
		proxyLogger = NewLogMonitorWriter(muxLogger)
	}

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

	peerProxy, err := NewPeerProxy(cfg.Swap.Peers, proxyLogger)
	if err != nil {
		proxyLogger.Errorf("Disabling Peering. Failed to create proxy peers: %v", err)
		peerProxy = nil
	}

	pm := &ProxyManager{
		cfg:       cfg,
		ginEngine: gin.New(),

		proxyLogger:    proxyLogger,
		muxLogger:      muxLogger,
		upstreamLogger: upstreamLogger,

		metricsMonitor: newMetricsMonitor(proxyLogger, maxMetrics),

		processGroups: make(map[string]*ProcessGroup),

		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,

		buildDate: "unknown",
		commit:    "abcd1234",
		version:   "0",

		peerProxy: peerProxy,
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
			for _, preloadModelName := range cfg.Swap.Hooks.OnStartup.Preload {
				modelID, ok := cfg.Swap.RealModelName(preloadModelName)

				if !ok {
					proxyLogger.Warnf("Preload model %s not found in config", preloadModelName)
					continue
				}

				proxyLogger.Infof("Preloading model: %s", modelID)
				processGroup, err := pm.swapProcessGroup(modelID)

				if err != nil {
					event.Emit(ModelPreloadedEvent{
						ModelName: modelID,
						Success:   false,
					})
					proxyLogger.Errorf("Failed to preload model %s: %v", modelID, err)
					continue
				} else {
					req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
					processGroup.ProxyRequest(modelID, discardWriter, req)
					event.Emit(ModelPreloadedEvent{
						ModelName: modelID,
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
	// Protected routes use pm.apiKeyAuth() middleware
	pm.ginEngine.POST("/v1/chat/completions", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	pm.ginEngine.POST("/v1/responses", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	// Support legacy /v1/completions api, see issue #12
	pm.ginEngine.POST("/v1/completions", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	// Support anthropic /v1/messages (added https://github.com/ggml-org/llama.cpp/pull/17570)
	pm.ginEngine.POST("/v1/messages", pm.apiKeyAuth(), pm.ProxyInferenceHandler)

	// Support embeddings and reranking
	pm.ginEngine.POST("/v1/embeddings", pm.apiKeyAuth(), pm.ProxyInferenceHandler)

	// llama-server's /reranking endpoint + aliases
	pm.ginEngine.POST("/reranking", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	pm.ginEngine.POST("/rerank", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	pm.ginEngine.POST("/v1/rerank", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	pm.ginEngine.POST("/v1/reranking", pm.apiKeyAuth(), pm.ProxyInferenceHandler)

	// llama-server's /infill endpoint for code infilling
	pm.ginEngine.POST("/infill", pm.apiKeyAuth(), pm.ProxyInferenceHandler)

	// llama-server's /completion endpoint
	pm.ginEngine.POST("/completion", pm.apiKeyAuth(), pm.ProxyInferenceHandler)

	// Support audio/speech endpoint
	pm.ginEngine.POST("/v1/audio/speech", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	pm.ginEngine.POST("/v1/audio/voices", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	pm.ginEngine.POST("/v1/audio/transcriptions", pm.apiKeyAuth(), pm.ProxyOAIPostFormHandler)
	pm.ginEngine.POST("/v1/images/generations", pm.apiKeyAuth(), pm.ProxyInferenceHandler)
	pm.ginEngine.POST("/v1/images/edits", pm.apiKeyAuth(), pm.ProxyOAIPostFormHandler)

	pm.ginEngine.GET("/v1/models", pm.apiKeyAuth(), pm.ListModelsHandler)

	// in proxymanager_loghandlers.go
	pm.ginEngine.GET("/logs", pm.apiKeyAuth(), pm.sendLogsHandlers)
	pm.ginEngine.GET("/logs/stream", pm.apiKeyAuth(), pm.StreamLogsHandler)
	pm.ginEngine.GET("/logs/stream/*logMonitorID", pm.apiKeyAuth(), pm.StreamLogsHandler)

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
	pm.ginEngine.Any("/upstream/*upstreamPath", pm.apiKeyAuth(), pm.proxyToUpstream)
	pm.ginEngine.GET("/unload", pm.apiKeyAuth(), pm.UnloadAllModelsHandler)
	pm.ginEngine.GET("/running", pm.apiKeyAuth(), pm.ListRunningProcessesHandler)
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

func (pm *ProxyManager) swapProcessGroup(realModelName string) (*ProcessGroup, error) {
	processGroup := pm.findGroupByModelName(realModelName)
	if processGroup == nil {
		return nil, fmt.Errorf("could not find process group for model %s", realModelName)
	}

	if processGroup.exclusive {
		pm.proxyLogger.Debugf("Exclusive mode for group %s, stopping other process groups", processGroup.id)
		for groupId, otherGroup := range pm.processGroups {
			if groupId != processGroup.id && !otherGroup.persistent {
				otherGroup.StopProcesses(StopWaitForInflightRequest)
			}
		}
	}

	return processGroup, nil
}

func (pm *ProxyManager) ListModelsHandler(c *gin.Context) {
	data := make([]gin.H, 0, len(pm.cfg.Swap.Models))
	createdTime := time.Now().Unix()

	newRecord := func(modelId string, modelConfig *config.ModelConfig) gin.H {
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

	for id, modelConfig := range pm.cfg.Swap.Models {
		if modelConfig.Unlisted {
			continue
		}

		data = append(data, newRecord(id, modelConfig))

		// Include aliases
		if pm.cfg.Swap.IncludeAliasesInList {
			for _, alias := range modelConfig.Aliases {
				if alias := strings.TrimSpace(alias); alias != "" {
					data = append(data, newRecord(alias, modelConfig))
				}
			}
		}
	}

	if pm.peerProxy != nil {
		for peerID, peer := range pm.peerProxy.ListPeers() {
			// add peer models
			for _, modelID := range peer.Models {
				// Skip unlisted models if not showing them
				record := newRecord(modelID, &config.ModelConfig{
					Name: fmt.Sprintf("%s: %s", peerID, modelID),
					Metadata: map[string]any{
						"peerID": peerID,
					},
				})

				data = append(data, record)
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

// findModelInPath searches for a valid model name in a path with slashes.
// It iteratively builds up path segments until it finds a matching model.
// Returns: (searchModelName, realModelName, remainingPath, found)
// Example: "/author/model/endpoint" with model "author/model" -> ("author/model", "author/model", "/endpoint", true).
func (pm *ProxyManager) findModelInPath(path string) (searchName, realName, remainingPath string, found bool) {
	parts := strings.Split(strings.TrimSpace(path), "/")
	searchModelName := ""

	for i, part := range parts {
		if part == "" {
			continue
		}

		if searchModelName == "" {
			searchModelName = part
		} else {
			searchModelName = searchModelName + "/" + part
		}

		if modelID, ok := pm.cfg.Swap.RealModelName(searchModelName); ok {
			return searchModelName, modelID, "/" + strings.Join(parts[i+1:], "/"), true
		}
	}

	return "", "", "", false
}

func (pm *ProxyManager) proxyToUpstream(c *gin.Context) {
	upstreamPath := c.Param("upstreamPath")

	searchModelName, modelID, remainingPath, modelFound := pm.findModelInPath(upstreamPath)

	if !modelFound {
		pm.sendErrorResponse(c, http.StatusBadRequest, "model id required in path")
		return
	}

	// Redirect /upstream/modelname to /upstream/modelname/ for URL consistency.
	// This ensures relative URLs in upstream responses resolve correctly and
	// provides canonical URL form. Uses 308 for POST/PUT/etc to preserve the
	// HTTP method (301 would downgrade to GET).
	if remainingPath == "/" && !strings.HasSuffix(upstreamPath, "/") {
		newPath := "/upstream/" + searchModelName + "/"
		if c.Request.URL.RawQuery != "" {
			newPath += "?" + c.Request.URL.RawQuery
		}
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Redirect(http.StatusMovedPermanently, newPath)
		} else {
			c.Redirect(http.StatusPermanentRedirect, newPath)
		}
		return
	}

	processGroup, err := pm.swapProcessGroup(modelID)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error swapping process group: "+err.Error())
		return
	}

	// rewrite the path
	originalPath := c.Request.URL.Path
	c.Request.URL.Path = remainingPath

	// attempt to record metrics if it is a POST request
	if pm.metricsMonitor != nil && c.Request.Method == http.MethodPost {
		err := pm.metricsMonitor.wrapHandler(modelID, c.Writer, c.Request, processGroup.ProxyRequest)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying metrics wrapped request: "+err.Error())
			pm.proxyLogger.Errorf("Error proxying wrapped upstream request for model %s, path=%s", modelID, originalPath)
			return
		}
	} else {
		err := processGroup.ProxyRequest(modelID, c.Writer, c.Request)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying request: "+err.Error())
			pm.proxyLogger.Errorf("Error proxying upstream request for model %s, path=%s", modelID, originalPath)
			return
		}
	}
}

func (pm *ProxyManager) ProxyInferenceHandler(c *gin.Context) {
	const download, agentSmith = false, false
	if download || agentSmith {
		if c.Request.URL.Path[2] == '/' {
			c.Request.URL.Path = c.Request.URL.Path[2:]
		} else {
			c.Request.URL.Path = "/v" + c.Request.URL.Path[2:]
		}
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "could not ready request body")
		return
	}

	requestedModel := gjson.GetBytes(bodyBytes, "model").String()
	if requestedModel == "" {
		// fallback: the first running process we find
		for _, processGroup := range pm.processGroups {
			for _, process := range processGroup.processes {
				if process.CurrentState() == StateReady {
					requestedModel = process.ID
				}
			}
		}
		if requestedModel == "" {
			pm.sendErrorResponse(c, http.StatusBadRequest, "missing or invalid 'model' key")
			return
		}
	}

	// Look for a matching local model first
	var nextHandler func(modelID string, w http.ResponseWriter, r *http.Request) error

	modelID, found := pm.cfg.Swap.RealModelName(requestedModel)
	if found {
		processGroup, err := pm.swapProcessGroup(modelID)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error swapping process group: "+err.Error())
			return
		}

		// issue #69 allow custom model names to be sent to upstream
		useModelName := pm.cfg.Swap.Models[modelID].UseModelName
		if useModelName != "" {
			bodyBytes, err = sjson.SetBytes(bodyBytes, "model", useModelName)
			if err != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error rewriting model name in JSON: "+err.Error())
				return
			}
		}

		// issue #174 strip parameters from the JSON body
		stripParams, err := pm.cfg.Swap.Models[modelID].Filters.SanitizedStripParams()
		if err != nil { // just log it and continue
			pm.proxyLogger.Errorf("Error sanitizing strip params string: %s, %s", pm.cfg.Swap.Models[modelID].Filters.StripParams, err.Error())
		} else {
			for _, param := range stripParams {
				pm.proxyLogger.Debugf("<%s> stripping param: %s", modelID, param)
				bodyBytes, err = sjson.DeleteBytes(bodyBytes, param)
				if err != nil {
					pm.sendErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("error deleting parameter %s from request", param))
					return
				}
			}
		}

		pm.proxyLogger.Debugf("ProxyManager using local Process for model: %s", requestedModel)
		nextHandler = processGroup.ProxyRequest
	} else if pm.peerProxy != nil && pm.peerProxy.HasPeerModel(requestedModel) {
		pm.proxyLogger.Debugf("ProxyManager using ProxyPeer for model: %s", requestedModel)
		modelID = requestedModel
		nextHandler = pm.peerProxy.ProxyRequest
	}

	if nextHandler == nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "could not find suitable inference handler for "+requestedModel)
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// dechunk it as we already have all the body bytes see issue #11
	c.Request.Header.Del("Transfer-Encoding")
	c.Request.Header.Set("Content-Length", strconv.Itoa(len(bodyBytes)))
	c.Request.ContentLength = int64(len(bodyBytes))

	// issue #366 extract values that downstream handlers may need
	isStreaming := gjson.GetBytes(bodyBytes, "stream").Bool()
	ctx := context.WithValue(c.Request.Context(), proxyCtxKey("streaming"), isStreaming)
	ctx = context.WithValue(ctx, proxyCtxKey("model"), modelID)
	c.Request = c.Request.WithContext(ctx)

	if pm.metricsMonitor != nil && c.Request.Method == http.MethodPost {
		err := pm.metricsMonitor.wrapHandler(modelID, c.Writer, c.Request, nextHandler)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying metrics wrapped request: "+err.Error())
			pm.proxyLogger.Errorf("Error Proxying Metrics Wrapped Request model %s", modelID)
			return
		}
	} else {
		err := nextHandler(modelID, c.Writer, c.Request)
		if err != nil {
			pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying request: "+err.Error())
			pm.proxyLogger.Errorf("Error Proxying Request for model %s", modelID)
			return
		}
	}
}

func (pm *ProxyManager) ProxyOAIPostFormHandler(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory, larger files go to tmp disk
		pm.sendErrorResponse(c, http.StatusBadRequest, "error parsing multipart form: "+err.Error())
		return
	}

	// Get model parameter from the form
	requestedModel := c.Request.FormValue("model")
	if requestedModel == "" {
		// fallback: the first running process we find
		for _, processGroup := range pm.processGroups {
			for _, process := range processGroup.processes {
				if process.CurrentState() == StateReady {
					requestedModel = process.ID
				}
			}
		}
		if requestedModel == "" {
			pm.sendErrorResponse(c, http.StatusBadRequest, "missing or invalid 'model' parameter in form data")
			return
		}
	}

	modelID, found := pm.cfg.Swap.RealModelName(requestedModel)
	if !found {
		pm.sendErrorResponse(c, http.StatusBadRequest, "could not find real modelID for "+requestedModel)
		return
	}

	processGroup, err := pm.swapProcessGroup(modelID)
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
		for _, value := range values {
			// If this is the model field and we have a profile, use just the model name
			if key == "model" {
				// # issue #69 allow custom model names to be sent to upstream
				useModelName := pm.cfg.Swap.Models[modelID].UseModelName

				if useModelName != "" {
					value = useModelName
				} else {
					value = requestedModel
				}
			}
			field, err := multipartWriter.CreateFormField(key)
			if err != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error recreating form field")
				return
			}
			if _, err = field.Write([]byte(value)); err != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error writing form field")
				return
			}
		}
	}

	// Copy all files from the original request
	for key, fileHeaders := range c.Request.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			formFile, err := multipartWriter.CreateFormFile(key, fileHeader.Filename)
			if err != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error recreating form file")
				return
			}

			file, err := fileHeader.Open()
			if err != nil {
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error opening uploaded file")
				return
			}

			if _, err = io.Copy(formFile, file); err != nil {
				file.Close()
				pm.sendErrorResponse(c, http.StatusInternalServerError, "error copying file data")
				return
			}
			file.Close()
		}
	}

	// Close the multipart writer to finalize the form
	if err := multipartWriter.Close(); err != nil {
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
	if err := processGroup.ProxyRequest(modelID, c.Writer, modifiedReq); err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, "error proxying request: "+err.Error())
		pm.proxyLogger.Errorf("Error Proxying Request for processGroup %s and model %s", processGroup.id, modelID)
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

// apiKeyAuth returns a middleware that validates API keys if configured.
// Returns a pass-through handler if no API keys are configured.
func (pm *ProxyManager) apiKeyAuth() gin.HandlerFunc {
	if len(pm.cfg.Swap.RequiredAPIKeys) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		xApiKey := c.GetHeader("x-api-key")

		var bearerKey string
		var basicKey string
		if auth := c.GetHeader("Authorization"); auth != "" {
			if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
				bearerKey = after
			} else if after, ok := strings.CutPrefix(auth, "Basic "); ok {
				// Basic Auth: base64(username:password), password is the API key
				encoded := after
				if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
					parts := strings.SplitN(string(decoded), ":", 2)
					if len(parts) == 2 {
						basicKey = parts[1] // password is the API key
					}
				}
			}
		}

		// Use first key found: Basic, then Bearer, then x-api-key
		var providedKey string
		if basicKey != "" {
			providedKey = basicKey
		} else if bearerKey != "" {
			providedKey = bearerKey
		} else {
			providedKey = xApiKey
		}

		// Validate key
		valid := slices.Contains(pm.cfg.Swap.RequiredAPIKeys, providedKey)

		if !valid {
			c.Header("WWW-Authenticate", `Basic realm="llama-swap"`)
			pm.sendErrorResponse(c, http.StatusUnauthorized, "unauthorized: invalid or missing API key")
			c.Abort()
			return
		}

		// Strip auth headers to prevent leakage to upstream
		c.Request.Header.Del("Authorization")
		c.Request.Header.Del("X-Api-Key")

		c.Next()
	}
}

func (pm *ProxyManager) UnloadAllModelsHandler(c *gin.Context) {
	pm.StopProcesses(StopImmediately)
	c.String(http.StatusOK, "OK")
}

func (pm *ProxyManager) ListRunningProcessesHandler(context *gin.Context) {
	context.Header("Content-Type", "application/json")
	runningProcesses := make([]gin.H, 0) // Default to an empty response.

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

	// Put the results under the `running` key.
	response := gin.H{
		"running": runningProcesses,
	}

	context.JSON(http.StatusOK, response) // Always return 200 OK
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

// ProxyToFirstRunningProcess forwards the request to the any running process (llama-server).
func (pm *ProxyManager) ProxyToFirstRunningProcess(c *gin.Context) {
	for _, processGroup := range pm.processGroups {
		for _, process := range processGroup.processes {
			if process.CurrentState() == StateReady {
				process.ProxyRequest(c.Writer, c.Request)
				return
			}
		}
	}
	pm.sendErrorResponse(c, http.StatusInternalServerError, "No model currently running. Please select a model.")
}
