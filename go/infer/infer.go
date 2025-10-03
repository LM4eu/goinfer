// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package infer implements a lightweight LLM proxy with streaming support.
package infer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// Query represents an inference task request.
type Query struct {
	Template   string `json:"template,omitempty" yaml:"template,omitempty"`
	Model      string `json:"model,omitempty"    yaml:"model,omitempty"`
	Completion        //nolint:embeddedstructfieldcheck // moving Completion on top will increase the size struct from 200 to 376 bytes
	Ctx        int    `json:"ctx,omitempty"     yaml:"ctx,omitempty"`
	Timeout    int    `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

var defaultQuery = Query{
	Completion: Completion{
		Stream:           false,
		MaxTokens:        512, // llama-server accept both "n_predict" and "max_tokens"
		TopK:             40,
		TopP:             0.95,
		MinP:             0.05,
		Temperature:      0.2,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		RepeatPenalty:    1.0,
		Stop:             []string{"</s>"},
	},
	Model: "default",
	Ctx:   2048,
}

// completionHandler handles infer requests.
func (inf *Infer) completionHandler(c echo.Context) error {
	query := defaultQuery
	err := json.NewDecoder(c.Request().Body).Decode(&query)
	if errors.Is(err, io.EOF) {
		return gie.New(gie.NotFound, "the infer request is empty")
	} else if err != nil {
		return gie.Wrap(err, gie.Invalid, "expect JSON format (see documentation)")
	}

	// use the template from the query or from the config if any
	// replace {prompt} by the prompt from the query
	prompt, ok := query.Prompt.(string) // TODO support []string
	if ok {
		if query.Template != "" {
			query.Template = inf.Cfg.Templates[query.Model]
		}
		if query.Template != "" {
			query.Prompt = strings.ReplaceAll(query.Template, "{prompt}", prompt)
			query.Template = "" // remove from JSON request
		}
		// prompt parameter is mandatory
		if query.Prompt == "" {
			return gie.New(gie.Invalid, "mandatory prompt is empty in the /completions request")
		}
	} else if query.Prompt == nil {
		return gie.New(gie.Invalid, "mandatory prompt field is missing in the /completions request")
	}

	ginCtx := echo2gin(c)

	if query.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ginCtx.Request.Context(), time.Duration(query.Timeout)*time.Second)
		defer cancel()
		ginCtx.Request = ginCtx.Request.WithContext(ctx)
		query.Timeout = 0 // remove from JSON request
	}

	body, err := json.Marshal(query)
	if err != nil {
		return gie.Wrap(err, gie.ServerErr, "failed to marshal infer request")
	}

	slog.Debug("JSON", "query", query)
	_, _ = os.Stdout.Write(body)

	ginCtx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	ginCtx.Request.URL.Path = "/completions"

	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager configured")
	}
	inf.ProxyMan.ProxyOAIHandler(ginCtx)
	return nil
}

// chatCompletionsHandler handles the requests to the
// /v1/chat/completions endpoint (OpenAI-compatible API).
func (inf *Infer) chatCompletionsHandler(c echo.Context) error {
	var query OpenaiChatCompletions
	err := json.NewDecoder(c.Request().Body).Decode(&query)
	if errors.Is(err, io.EOF) {
		return gie.New(gie.NotFound, "the OpenAI request is empty")
	} else if err != nil {
		return gie.Wrap(err, gie.Invalid, "the request format is not OpenaiChatCompletions")
	}

	ginCtx := echo2gin(c)

	body, err := json.Marshal(query)
	if err != nil {
		return gie.Wrap(err, gie.ServerErr, "failed to marshal infer request")
	}

	ginCtx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	ginCtx.Request.URL.Path = "/v1/chat/completions"

	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager configured")
	}
	inf.ProxyMan.ProxyOAIHandler(ginCtx)
	return nil
}

// abortInference aborts an ongoing inference.
func (inf *Infer) abortInference() error {
	inf.mu.Lock()
	defer inf.mu.Unlock()
	if !inf.isInferring {
		return gie.New(gie.InferErr, "no inference running, nothing to abort")
	}

	slog.Debug("Aborting inference")

	inf.stopInferring = false
	return nil
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
