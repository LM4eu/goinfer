// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package proxy

import (
	"io"
	"strings"

	"github.com/LM4eu/goinfer/gie"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	D_  = "D_"
	AS_ = "AS_"
)

func (pm *ProxyManager) getSetModel(body io.ReadCloser, download, agentSmith bool) (
	bodyJsonBytes []byte, fixed string, download_, agentSmith_ bool, err error,
) {
	bodyJsonBytes, err = io.ReadAll(body)
	if err != nil {
		return nil, "", true, true, gie.Wrap(err, gie.Invalid, "cannot io.ReadAll(request body)")
	}

	// model name fom the request
	requested := gjson.GetBytes(bodyJsonBytes, "model").String()

	fixed, download, agentSmith = pm.fixModelName(requested, download, agentSmith)

	// issue #69 allow custom model names to be sent to upstream
	model, ok := pm.config.Models[fixed]
	if ok {
		useModelName := model.UseModelName
		if useModelName != "" && useModelName != requested {
			bodyJsonBytes, err = sjson.SetBytes(bodyJsonBytes, "model", useModelName)
			if err != nil {
				return nil, "", true, true, gie.Wrap(err, gie.Invalid,
					"cannot rewrite model name in JSON",
					"originalModel", requested, "newModel", useModelName)
			}
		}
	}

	return bodyJsonBytes, fixed, download, agentSmith, nil
}

func (pm *ProxyManager) fixModelName(requested string, download, agentSmith bool) (
	fixed string, download_, agentSmith_ bool,
) {
	fixed = requested

	if requested != "" {
		if strings.HasPrefix(requested, D_) {
			download = true
			fixed = fixed[len(D_):]
		}

		if strings.HasPrefix(fixed, AS_) {
			agentSmith = true
			fixed = fixed[len(AS_):]
		}

		real, found := pm.config.RealModelName(fixed)
		if found {
			fixed = real
		} else if !download {
			fixed = pm.cfg.FixModelName(fixed)
		}
	}

	if fixed == "" {
		download = false
		fixed = pm.firstRunningProcess()
	}

	return fixed, download, agentSmith
}
