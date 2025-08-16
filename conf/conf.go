package conf

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"github.com/mostlygeek/llama-swap/proxy"
	"github.com/teal-finance/garcon/gg"

	"gopkg.in/yaml.v3"
)

// GoInferConf holds the configuration for GoInfer.
type GoInferConf struct {
	ModelsDir string        `json:"models_dir" yaml:"models_dir"`
	Server    WebServerConf `json:"server"     yaml:"server"`
	Llama     LlamaConf     `json:"llama"      yaml:"llama"`
	Swap      *proxy.Config `json:"swap"       yaml:"swap"`
}

// WebServerConf holds the configuration for GoInfer web server.
type WebServerConf struct {
	Origins string            `json:"origins"    yaml:"origins"`
	Port    map[string]string `json:"port"       yaml:"port"`
	ApiKey  map[string]string `json:"api_key"    yaml:"api_key"`
}

const DefaultGoInferConf = `
models_dir: ./models

server:
	api_key:
		# ⚠️ Set a 64-byte secure API keys 🚨
		admin: "PLEASE SET SECURE API KEY"
		user:  "PLEASE SET SECURE API KEY"
	origins: "localhost"
	ports:
		admin:   "9999"
		goinfer: "2222"
		mcp:     "3333"
		openai:  "5143"

llama:
	exe_path: ./llama-server
	args:
		// --props: enable changing global properties via POST /props
		// --no-webui: no Web UI server
		common: --props --no-webui --no-warmup
		goinfer: --jinja --chat-template-file template.jinja
`

// Load the config file having any extension: json, yml, ini...
func Load(cfgFile string) (GoInferConf, error) {
	var cfg GoInferConf

	// 1. Default values
	err := yaml.Unmarshal([]byte(DefaultGoInferConf), &cfg)
	if err != nil {
		return cfg, err
	}

	// 2. config file
	bytes, err := os.ReadFile(cfgFile)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return cfg, err
	}

	// 3. env. vars
	cfg.ModelsDir = gg.EnvStr("GI_MODELS_DIR", cfg.ModelsDir)
	cfg.Server.Origins = gg.EnvStr("GI_ORIGINS", cfg.Server.Origins)
	if apiKey, ok := cfg.Server.ApiKey["admin"]; ok {
		cfg.Server.ApiKey["admin"] = gg.EnvStr("GI_API_KEY_ADMIN", apiKey)
	}
	if apiKey, ok := cfg.Server.ApiKey["user"]; ok {
		cfg.Server.ApiKey["user"] = gg.EnvStr("GI_API_KEY_USER", apiKey)
	}

	cfg.Swap, err = proxy.LoadConfig("llama-swap.yml")
	if err != nil {
		fmt.Printf("Error loading llama-swap.yml config: %v\n", err)
		os.Exit(1)
	}

	err = CheckApiKeys(cfg.Server.ApiKey)
	return cfg, err
}

func CheckApiKeys(ApiKeys map[string]string) error {
	err := errors.New("missing a seriously secured server.api_key.admin")
	for k, v := range ApiKeys {
		if len(v) < 64 {
			return errors.New("secured api_key must be 64 bytes: " + v)
		}
		if v == DebugApiKey {
			fmt.Print("WARNING: Conf uses DEBUG api_key = security threat")
		}
		if k == "admin" {
			err = nil
		}
	}
	return err
}

const DebugApiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"

func GenApiKey(random bool) []byte {
	if random {
		apiKey := make([]byte, 64)
		rand.Read(apiKey)
		return apiKey
	}
	return []byte(DebugApiKey)
}

// Create a YAML configuration
func Create(fileName string, random bool) error {
	cfg := []byte(DefaultGoInferConf)
	bytes.Replace(cfg, []byte("PLEASE SET SECURE API KEY"), GenApiKey(random), 1)
	bytes.Replace(cfg, []byte("PLEASE SET SECURE API KEY"), GenApiKey(random), 1)
	err := os.WriteFile(fileName, cfg, 0644)
	return err
}

// Debug prints viper debug info and the configuration to stdout in YAML format
func (cfg *GoInferConf) Debug() error {

	// Marshal the configuration to YAML
	bytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("error Marshal(cfg) to YAML: %w", err)
	}

	// Print to stdout
	_, err = os.Stdout.Write(bytes)
	return err
}
