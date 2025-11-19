// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package infer

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/LM4eu/goinfer/gie"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type (
	ModelRequest interface {
		// GetModel using m.Model or m["model"].
		GetModel() string
		// SetModel using m.Model or m["model"].
		SetModel(model string)
	}

	ModelField struct {
		Model string `json:"model,omitempty" yaml:"model,omitempty"`
	}

	anyBody map[string]any
)

const (
	// Debug=true enables json.Unmarshal/Marshal, more reliable than gjson.GetBytes/SetBytes, but consumes much more CPU.
	debug = false
)

// GetModel implements ModelRequest interface.
func (m *ModelField) GetModel() string { return m.Model }

// SetModel implements ModelRequest interface.
func (m *ModelField) SetModel(model string) { m.Model = model }

// GetModel uses map["model"] to get the model name.
// A model name containing '"' could be a symptom of a malformed JSON.
func (m *anyBody) GetModel() string {
	modelAny, ok := (*m)["model"]
	if !ok {
		return ""
	}
	if modelAny == nil {
		return ""
	}
	model, ok := modelAny.(string)
	if ok {
		return model
	}
	return fmt.Sprint(modelAny)
}

// SetModel implements ModelRequest interface.
func (m *anyBody) SetModel(model string) {
	(*m)["model"] = model
}

func setModelIfMissing[T ModelRequest](inf *Infer, msg T, bodyReader io.ReadCloser) ([]byte, error) {
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, gie.Wrap(err, gie.Invalid, "cannot io.ReadAll(request body)")
	}

	model := gjson.GetBytes(body, "model").String()
	if model != "" && model != "default" {
		fixed := inf.Cfg.FixModelName(model)
		if model == fixed {
			return body, nil
		}
	}

	if model == "" {
		model, err = selectModel(inf)
		if err != nil {
			slog.Warn("Cannot prob /running", "err", err)
		}
	}
	if model == "" {
		return nil, gie.Wrap(err, gie.Invalid,
			"no model loaded and no default_model in goinfer.ini => specify the field model in the request")
	}

	// set the model in the JSON body
	if debug {
		// The debug mode use a reliable conversion
		// from the JSON bytes into a Go struct.
		// But this consumes more CPU and requires
		// to convert back the Go struct into aJSON bytes.
		err = json.Unmarshal(body, &msg)
		if err != nil {
			return nil, gie.Wrap(err, gie.Invalid, "invalid or malformed JSON", "received body", string(body))
		}
		msg.SetModel(model)

		body, err = json.Marshal(msg)
		if err != nil {
			return nil, gie.Wrap(err, gie.Invalid, "error json.Marshal back the body", "input msg", msg)
		}
	} else {
		body, err = sjson.SetBytes(body, "model", model)
		if err != nil {
			return nil, gie.Wrap(err, gie.Invalid, "cannot update model in JSON body", "body", body, "new model", model)
		}
	}

	return body, nil
}

func selectModel(inf *Infer) (string, error) {
	var body []byte

	req := httptest.NewRequest(http.MethodGet, "/running", http.NoBody)
	rec := httptest.NewRecorder()
	inf.ProxyMan.ListRunningProcessesHandler(&gin.Context{
		Writer:  &responseWriter{ResponseWriter: rec, size: -1, status: http.StatusOK},
		Request: req,
	})
	body = rec.Body.Bytes()

	// Assuming the JSON structure has a "running" field
	var response struct {
		Running []struct {
			Model string `json:"model"` // Exported and with JSON tags
			State string `json:"state"`
		} `json:"running"` // Specify the actual JSON field name
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return inf.Cfg.DefaultModel, gie.Wrap(err, gie.InferErr, "invalid or malformed JSON", "received response body from /running", string(body))
	}

	// Check for ready models first
	for _, m := range response.Running {
		if m.State == "ready" {
			return m.Model, nil
		}
	}

	// Check for starting models
	for _, m := range response.Running {
		if m.State == "starting" {
			return m.Model, nil
		}
	}

	// no ready / starting model => use DefaultModel
	return inf.Cfg.DefaultModel, nil
}
