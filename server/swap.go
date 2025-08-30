package server

import (
	"net/http"
	"strings"

	"github.com/mostlygeek/llama-swap/proxy"
	"github.com/synw/goinfer/conf"
)

func NewProxyServer(cfg *conf.GoInferCfg) (*http.Server, *proxy.ProxyManager) {
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
