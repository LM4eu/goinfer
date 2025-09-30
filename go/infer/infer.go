// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package infer implements a lightweight LLM proxy with streaming support.
package infer

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/LM4eu/goinfer/gie"
	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
)

type responseWriter struct {
	http.ResponseWriter
	size   int
	status int
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseWriter) reset(writer http.ResponseWriter) {
	w.ResponseWriter = writer
	w.size = -1
	w.status = http.StatusOK
}

func (w *responseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		if w.Written() {
			return
		}
		w.status = code
	}
}

func (w *responseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	w.WriteHeaderNow()
	n, err = w.ResponseWriter.Write(data)
	w.size += n
	return
}

func (w *responseWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	n, err = io.WriteString(w.ResponseWriter, s)
	w.size += n
	return
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Size() int {
	return w.size
}

func (w *responseWriter) Written() bool {
	return w.size != -1
}

// Hijack implements the http.Hijacker interface.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.Written() {
		return nil, nil, gie.New(gie.InferErr, "response already written")
	}
	if w.size < 0 {
		w.size = 0
	}
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotifier interface.
func (w *responseWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Flush implements the http.Flusher interface.
func (w *responseWriter) Flush() {
	w.WriteHeaderNow()
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *responseWriter) Pusher() (pusher http.Pusher) {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}

// inferHandler handles infer requests.
func (inf *Infer) inferHandler(c echo.Context) error {
	// Initialize context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)

	// Bind request parameters
	reqMap := echo.Map{}
	err := c.Bind(&reqMap)
	if err != nil {
		cancel()
		return gie.HandleValidationError(c, gie.ErrInvalidFormat)
	}

	// Parse infer parameters directly
	query, err := parseInferQuery(ctx, reqMap)
	if err != nil {
		cancel()
		return gie.HandleValidationError(c, gie.ErrInvalidParams)
	}

	if query.Timeout == 0 {
		ctx, cancel = context.WithTimeout(c.Request().Context(), time.Duration(query.Timeout)*time.Second)
	}
	defer cancel()

	ginWriter := responseWriter{
		ResponseWriter: c.Response().Writer,
		size:           -1,
		status:         http.StatusOK,
	}
	ginCtx := gin.Context{}
	ginCtx.Writer = &ginWriter
	ginCtx.Request = c.Request()
	inf.ProxyMan.ProxyOAIHandler(&ginCtx)
	return nil
}

// parseInferQuery parses infer parameters from echo.Map directly.
func parseInferQuery(ctx context.Context, m echo.Map) (*InferQuery, error) {
	query := &InferQuery{
		Prompt: "",
		Model:  DefaultModel,
		Params: DefaultInferParams,
	}

	// Parse required prompt parameter
	val, ok := m["prompt"].(string)
	if !ok {
		return query, gie.ErrInvalidPrompt
	}
	query.Prompt = val

	if val, ok := m["model"].(string); ok {
		query.Model.Name = val
	}

	query.Model.Ctx = getInt(ctx, m, "ctx")

	if val, ok := m["stream"].(bool); ok {
		query.Params.Stream = val
	}

	query.Params.Sampling.Temperature = getFloat(ctx, m, "temperature")
	query.Params.Sampling.MinP = getFloat(ctx, m, "min_p")
	query.Params.Sampling.TopP = getFloat(ctx, m, "top_p")
	query.Params.Sampling.PresencePenalty = getFloat(ctx, m, "presence_penalty")
	query.Params.Sampling.FrequencyPenalty = getFloat(ctx, m, "frequency_penalty")
	query.Params.Sampling.RepeatPenalty = getFloat(ctx, m, "repeat_penalty")
	query.Params.Sampling.TailFreeSamplingZ = getFloat(ctx, m, "tfs")
	query.Params.Sampling.TopK = getInt(ctx, m, "top_k")
	query.Params.Generation.MaxTokens = getInt(ctx, m, "max_tokens")

	// Parse stop prompts array
	err := populateStopPrompts(m, &query.Params.Generation)
	if err != nil {
		return query, err
	}

	// Parse media arrays
	if images, ok := m["images"].([]string); ok {
		query.Params.Media.Images = images
	}
	if audios, ok := m["audios"].([]string); ok {
		query.Params.Media.Audios = audios
	}

	return query, nil
}

func getInt(ctx context.Context, m echo.Map, param string) int {
	v, ok := m[param]
	if ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		default:
			slog.WarnContext(ctx, "expected int (or float64) but received", "param", param, "value", m[param])
		}
	}
	return 0
}

func getFloat(ctx context.Context, m echo.Map, param string) float32 {
	v, ok := m[param]
	if ok {
		switch val := v.(type) {
		case int:
			return float32(val)
		case float64:
			return float32(val)
		default:
			slog.WarnContext(ctx, "expected float64 (or int) but received", "param", param, "value", m[param])
		}
	}
	return 0
}

// execute inference.
func (inf *Infer) execute(c echo.Context, query *InferQuery) (*StreamedMsg, error) {
	// Execute infer through Infer
	resChan := make(chan StreamedMsg)
	errChan := make(chan StreamedMsg)
	defer close(resChan)
	defer close(errChan)

	err := inf.forwardInference(c.Request().Context(), query, c, resChan, errChan)
	if err != nil {
		return nil, gie.Wrap(err, gie.InferErr, "proxy manager forward inference failed")
	}

	// Process response
	select {
	case resCh, ok := <-resChan:
		if ok {
			return &resCh, nil
		}
		return nil, gie.ErrChanClosed

	case errCh, ok := <-errChan:
		if ok {
			return nil, gie.Wrap(errCh.Error, gie.InferErr, string(errCh.MsgType))
		}
		return nil, gie.ErrChanClosed

	case <-c.Request().Context().Done():
		// Client canceled request
		inf.mu.Lock()
		inf.ContinueInferringController = false
		inf.mu.Unlock()
		return nil, gie.ErrClientCanceled
	}
}

// abortHandler aborts ongoing inference.
func (inf *Infer) abortHandler(c echo.Context) error {
	err := inf.abortInference()
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "abortInference", "error", err)
		return c.NoContent(http.StatusAccepted)
	}

	slog.DebugContext(c.Request().Context(), "Aborting inference")

	return c.NoContent(http.StatusNoContent)
}

// populateStopPrompts extracts and validates the "stop" parameter from the request map.
func populateStopPrompts(m echo.Map, gen *Generation) error {
	v, ok := m["stop"]
	if !ok {
		return nil
	}
	slice, ok := v.([]any)
	if !ok {
		return gie.New(gie.Invalid, "invalid parameter type: stop must be an array")
	}
	if len(slice) > 10 {
		return gie.New(gie.Invalid, "invalid size: stop array too large (max 10) got="+strconv.Itoa(len(slice)))
	}
	if len(slice) == 0 {
		return nil
	}
	gen.StopPrompts = make([]string, len(slice))
	for i, val := range slice {
		str, ok := val.(string)
		if !ok {
			return gie.New(gie.Invalid, "invalid parameter type: stop["+strconv.Itoa(i)+"] must be a string")
		}
		gen.StopPrompts[i] = str
	}
	return nil
}
