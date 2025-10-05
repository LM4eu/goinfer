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
	"strings"

	"github.com/LM4eu/goinfer/gie"
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

	AnyBody map[string]any
)

// debug=true enables json.Unmarshal/Marshal, more reliable than gjson.GetBytes/SetBytes, but consumes much more CPU.
const debug = false

func (m *ModelField) GetModel() string      { return m.Model }
func (m *ModelField) SetModel(model string) { m.Model = model }

func (m *AnyBody) GetModel() string {
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
	model = fmt.Sprint(modelAny)
	if strings.ContainsAny(model, "{[':,") {
		return ""
	}
	return model
}

func (m *AnyBody) SetModel(model string) {
	(*m)["model"] = model
}

func setModelIfMissing[T ModelRequest](msg T, bodyReader io.ReadCloser, defaultModel string) ([]byte, error) {
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, gie.Wrap(err, gie.Invalid, "cannot io.ReadAll(request body)")
	}

	model := gjson.GetBytes(body, "model").String()
	if model != "" && model != "default" {
		// TODO: verify the model is known to work
		return body, nil
	}

	model = selectModel(defaultModel)
	if model == "" {
		return body, gie.Wrap(err, gie.Invalid,
			"no model loaded and no default_model in goinfer.yml => specify the field model in the request")
	}

	// set model in the
	if debug {
		// convert the JSON bytes into a Go struct
		err = json.Unmarshal(body, &msg)
		if err != nil {
			return body, gie.Wrap(err, gie.Invalid, "invalid or malformed JSON", "received body", string(body))
		}
		msg.SetModel(model)

		body, err = json.Marshal(msg)
		if err != nil {
			return body, gie.Wrap(err, gie.Invalid, "error json.Marshal back the body", "input msg", msg)
		}
	} else {
		body, err = sjson.SetBytes(body, "model", model)
		if err != nil {
			return body, gie.Wrap(err, gie.Invalid, "cannot update model in JSON body", "body", body, "new model", model)
		}
	}

	return body, nil
}

func selectModel(defaultModel string) string {
	res, err := http.Get("http://localhost:5555/running")
	if err != nil {
		return defaultModel
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Debug("cannot io.ReadAll(response body) from /running", "err", err)
		return defaultModel
	}

	// Assuming the JSON structure has a "running" field
	var response struct {
		Running []struct {
			Model string `json:"model"` // Exported and with JSON tags
			State string `json:"state"`
		} `json:"running"` // Specify the actual JSON field name
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		slog.Debug("invalid or malformed JSON", "received response body from /running", string(body), "err", err)
		return defaultModel
	}

	// Check for ready models first
	for _, m := range response.Running {
		if m.State == "ready" {
			return m.Model
		}
	}

	// Check for starting models
	for _, m := range response.Running {
		if m.State == "starting" {
			return m.Model
		}
	}

	return defaultModel
}
