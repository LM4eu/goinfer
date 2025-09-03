// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package server

import (
	"net/http"
	"strings"

	"github.com/LM4eu/goinfer/conf"
	"github.com/mostlygeek/llama-swap/proxy"
)

func NewProxy(cfg *conf.GoInferCfg) (*http.Server, *proxy.ProxyManager) {
	for addr, services := range cfg.Server.Listen {
		if strings.Contains(services, "swap") {
			pm := proxy.New(cfg.Proxy)
			srv := &http.Server{
				Addr:    addr,
				Handler: pm,
			}
			return srv, pm
		}
	}
	return nil, nil // llama-swap not present => not enabled
}
