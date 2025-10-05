// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package infer implements a lightweight LLM proxy with streaming support.
package infer

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/LM4eu/goinfer/gie"
	"github.com/labstack/echo/v4"
)

// Query represents an inference task request.
type Query struct {
	ModelField

	Template string `json:"template,omitempty" yaml:"template,omitempty"`
	Completion
	Ctx     int `json:"ctx,omitempty"     yaml:"ctx,omitempty"`
	Timeout int `json:"timeout,omitempty" yaml:"timeout,omitempty"`
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
	ModelField: ModelField{
		Model: "default",
	},
	Ctx:     2048,
	Timeout: 30, // 30 seconds
}

// completionHandler handles llama.cpp /completions endpoint.
func (inf *Infer) completionHandler(c echo.Context) error {
	msg := defaultQuery
	err := setModelIfMissing(&msg, c.Request().Body, inf.Cfg.Main.DefaultModel)
	if err != nil {
		return err
	}

	// use the template from the query or from the config if any
	// replace {prompt} by the prompt from the query
	prompt, ok := msg.Prompt.(string) // TODO support []string
	if !ok {
		return gie.New(gie.Invalid, "only support string for prompt (TODO support []string)", "the issue is in this /completions request", msg)
	}

	// prompt parameter is mandatory
	if msg.Prompt == "" {
		return gie.New(gie.Invalid, "mandatory prompt is empty", "the issue is in this /completions request", msg)
	}

	// apply template if any
	if msg.Template != "" {
		msg.Template = inf.Cfg.Main.Templates[msg.Model]
	}
	if msg.Template != "" {
		msg.Prompt = strings.ReplaceAll(msg.Template, "{prompt}", prompt)
		msg.Template = "" // remove from JSON request
	}

	ginCtx, err := getGinCtx(c, &msg)
	if err != nil {
		return err
	}

	if msg.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ginCtx.Request.Context(), time.Duration(msg.Timeout)*time.Second)
		defer cancel()
		ginCtx.Request = ginCtx.Request.WithContext(ctx)
		msg.Timeout = 0 // remove from JSON request
	}

	inf.ProxyMan.ProxyOAIHandler(ginCtx)
	return nil
}

// chatCompletionsHandler handles the requests to the
// /v1/chat/completions endpoint (OpenAI-compatible API).
func (inf *Infer) chatCompletionsHandler(c echo.Context) error {
	var msg OpenaiChatCompletions
	err := setModelIfMissing(&msg, c.Request().Body, inf.Cfg.Main.DefaultModel)
	if err != nil {
		return err
	}

	ginCtx, err := getGinCtx(c, &msg)
	if err != nil {
		return err
	}

	inf.ProxyMan.ProxyOAIHandler(ginCtx)
	return nil
}

func (inf *Infer) proxyOAIHandler(c echo.Context) error {
	var msg AnyBody
	err := setModelIfMissing(&msg, c.Request().Body, inf.Cfg.Main.DefaultModel)
	if err != nil {
		return err
	}

	ginCtx, err := getGinCtx(c, &msg)
	if err != nil {
		return err
	}

	inf.ProxyMan.ProxyOAIHandler(ginCtx)
	return nil
}

func (inf *Infer) proxyOAIPostFormHandler(c echo.Context) error {
	var msg AnyBody
	err := setModelIfMissing(&msg, c.Request().Body, inf.Cfg.Main.DefaultModel)
	if err != nil {
		return err
	}

	ginCtx, err := getGinCtx(c, &msg)
	if err != nil {
		return err
	}

	inf.ProxyMan.ProxyOAIHandler(ginCtx)
	return nil
}

func (inf *Infer) listModelsHandler(c echo.Context) error {
	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager (llama-swap)")
	}
	inf.ProxyMan.ListModelsHandler(echo2gin(c))
	return nil
}

func (inf *Infer) streamLogsHandler(c echo.Context) error {
	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager (llama-swap)")
	}
	inf.ProxyMan.StreamLogsHandler(echo2gin(c))
	return nil
}

func (inf *Infer) proxyToFirstRunningProcess(c echo.Context) error {
	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager (llama-swap)")
	}
	inf.ProxyMan.ProxyToFirstRunningProcess(echo2gin(c))
	return nil
}

func (inf *Infer) listRunningProcessesHandler(c echo.Context) error {
	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager (llama-swap)")
	}
	inf.ProxyMan.ListRunningProcessesHandler(echo2gin(c))
	return nil
}

func (inf *Infer) unloadAllModelsHandler(c echo.Context) error {
	if inf.ProxyMan == nil {
		return gie.New(gie.InferErr, "no proxy manager (llama-swap)")
	}
	inf.ProxyMan.UnloadAllModelsHandler(echo2gin(c))
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
