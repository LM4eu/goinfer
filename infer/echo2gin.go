// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"

	"github.com/LynxAIeu/goinfer/gie"
	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
)

type responseWriter struct {
	http.ResponseWriter

	size   int
	status int
}

// inferHandler handles infer requests.
func echo2gin(c echo.Context) *gin.Context {
	return &gin.Context{
		Writer: &responseWriter{
			ResponseWriter: c.Response().Writer,
			size:           -1,
			status:         http.StatusOK,
		},
		Request: c.Request(),
	}
}

func echo2ginWithBody(c echo.Context, body []byte) *gin.Context {
	ginCtx := echo2gin(c)
	ginCtx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return ginCtx
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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
	return n, err
}

func (w *responseWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	n, err = io.WriteString(w.ResponseWriter, s)
	w.size += n
	return n, err
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
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, gie.New(gie.ServerErr, "w.ResponseWriter is not http.Hijacker")
	}
	return hijacker.Hijack()
}

// CloseNotify implements the http.CloseNotifier interface.
func (w *responseWriter) CloseNotify() <-chan bool {
	notifier, ok := w.ResponseWriter.(http.CloseNotifier)
	if !ok {
		return nil
	}
	return notifier.CloseNotify()
}

// Flush implements the http.Flusher interface.
func (w *responseWriter) Flush() {
	w.WriteHeaderNow()
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (w *responseWriter) Pusher() (pusher http.Pusher) {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}
