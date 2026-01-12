// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

// Package infer implements a lightweight LLM proxy with streaming support.
package infer

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/LynxAIeu/garcon/gerr"
	"github.com/labstack/echo/v4"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Query represents an inference task request.
type Query struct {
	ModelField

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
	body, err := setModelIfMissing(inf, &msg, c.Request().Body)
	if err != nil {
		return err
	}

	// replace {prompt} by the prompt from the query
	prompt := gjson.GetBytes(body, "prompt")
	if prompt.Type != gjson.String { // TODO support []string
		return gerr.New(gerr.Invalid, "only support string for prompt (TODO support []string)", "the issue is in this /completions request", msg)
	}

	// prompt parameter is mandatory
	if prompt.Str == "" {
		return gerr.New(gerr.Invalid, "mandatory prompt is empty", "the issue is in this /completions request", msg)
	}

	var timeout int64
	timeoutJSON := gjson.GetBytes(body, "timeout")
	if timeoutJSON.Index != 0 {
		timeout = timeoutJSON.Int()
		body, err = sjson.DeleteBytes(body, "timeout")
		if err != nil {
			return gerr.Wrap(err, gerr.Invalid, "cannot delete the timeout field in the JSON", "the issue is in this request body", body)
		}
	}

	ginCtx := echo2ginWithBody(c, body)

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(ginCtx.Request.Context(), time.Duration(timeout)*time.Second)
		defer cancel()
		ginCtx.Request = ginCtx.Request.WithContext(ctx)
	}

	inf.ProxyMan.ProxyInferenceHandler(ginCtx)
	return nil
}

// chatCompletionsHandler handles the requests to the
// /v1/chat/completions endpoint (OpenAI-compatible API).
func (inf *Infer) chatCompletionsHandler(c echo.Context) error {
	var msg OpenaiChatCompletions
	body, err := setModelIfMissing(inf, &msg, c.Request().Body)
	if err != nil {
		return err
	}

	ginCtx := echo2ginWithBody(c, body)
	inf.ProxyMan.ProxyInferenceHandler(ginCtx)
	return nil
}

func (inf *Infer) proxyInferenceHandler(c echo.Context) error {
	var msg anyBody
	body, err := setModelIfMissing(inf, &msg, c.Request().Body)
	if err != nil {
		return err
	}

	ginCtx := echo2ginWithBody(c, body)
	inf.ProxyMan.ProxyInferenceHandler(ginCtx)
	return nil
}

func (inf *Infer) proxyOAIPostFormHandler(c echo.Context) error {
	var msg anyBody
	body, err := setModelIfMissing(inf, &msg, c.Request().Body)
	if err != nil {
		return err
	}

	ginCtx := echo2ginWithBody(c, body)
	inf.ProxyMan.ProxyInferenceHandler(ginCtx)
	return nil
}

func (inf *Infer) listModelsHandler(c echo.Context) error {
	inf.ProxyMan.ListModelsHandler(echo2gin(c))
	return nil
}

func (inf *Infer) streamLogsHandler(c echo.Context) error {
	inf.ProxyMan.StreamLogsHandler(echo2gin(c))
	return nil
}

func (inf *Infer) proxyToFirstRunningProcess(c echo.Context) error {
	inf.ProxyMan.ProxyToFirstRunningProcess(echo2gin(c))
	return nil
}

func (inf *Infer) listRunningProcessesHandler(c echo.Context) error {
	inf.ProxyMan.ListRunningProcessesHandler(echo2gin(c))
	return nil
}

func (inf *Infer) unloadAllModelsHandler(c echo.Context) error {
	inf.ProxyMan.UnloadAllModelsHandler(echo2gin(c))
	return nil
}

// abortInference aborts an ongoing inference.
func (inf *Infer) abortInference() error {
	inf.mu.Lock()
	defer inf.mu.Unlock()
	if !inf.isInferring {
		return gerr.New(gerr.InferErr, "no inference running, nothing to abort")
	}

	slog.Debug("Aborting inference")

	inf.stopInferring = false
	return nil
}

// abortHandler aborts ongoing inference.
func (inf *Infer) abortHandler(c echo.Context) error {
	err := inf.abortInference()
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "abortInference", "err", err)
		return c.NoContent(http.StatusAccepted)
	}

	slog.DebugContext(c.Request().Context(), "Aborting inference")

	return c.NoContent(http.StatusNoContent)
}
