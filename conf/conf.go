// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/models"
	"github.com/LM4eu/goinfer/state"
	"github.com/mostlygeek/llama-swap/proxy"

	"gopkg.in/yaml.v3"
)

type (
	GoInferCfg struct {
		Llama     LlamaCfg     `json:"llama,omitempty"      yaml:"llama,omitempty"`
		Server    ServerCfg    `json:"server,omitempty"     yaml:"server,omitempty"`
		ModelsDir string       `json:"models_dir,omitempty" yaml:"models_dir,omitempty"`
		Proxy     proxy.Config `json:"proxy,omitempty"      yaml:"proxy,omitempty"`
		Verbose   bool         `json:"verbose,omitempty"    yaml:"verbose,omitempty"`
	}

	ServerCfg struct {
		Listen  map[string]string `json:"listen,omitempty"  yaml:"listen,omitempty"`
		APIKeys map[string]string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
		Origins string            `json:"origins,omitempty" yaml:"origins,omitempty"`
	}

	LlamaCfg struct {
		Args map[string]string `json:"args,omitempty" yaml:"args,omitempty"`
		Exe  string            `json:"exe,omitempty"  yaml:"exe,omitempty"`
	}
)

const (
	pleaseSetSecureAPIKey = "PLEASE SET SECURE API KEY"

	debugAPIKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"

	defaultGoInferConf = `# Configuration of https://github.com/LM4eu/goinfer

# Recursively search *.gguf files (one or multiple directories separated by ':')
models_dir: ./models

server:
  api_key:
    # ⚠️ Set your private 32-byte API keys (64 hex digits) 🚨
    "admin": ` + pleaseSetSecureAPIKey + `
    "user":  ` + pleaseSetSecureAPIKey + `
  origins: localhost
  listen:
    ":8080": admin
    ":2222": openai,goinfer,mcp
    ":5143": llama-swap proxy

llama:
  exe: ./llama-server
  args:
    # --props: enable changing global properties via POST /props
    # --no-webui: no Web UI server
    "common": --props --no-webui --no-warmup
    "goinfer": --jinja --chat-template-file template.jinja
`
)

// Load configuration with simplified loading.
func Load(goinferCfgFile string) (*GoInferCfg, error) {
	var cfg *GoInferCfg

	// Load default config
	err := yaml.Unmarshal([]byte(defaultGoInferConf), &cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse default config: %w", err)
	}

	// Load from file if specified
	if goinferCfgFile != "" { // Use OpenFileIn() from Go-1.25
		data, er := os.ReadFile(filepath.Clean(goinferCfgFile))
		if er != nil {
			return cfg, fmt.Errorf("failed to read %s: %w", goinferCfgFile, er)
		}

		err = yaml.Unmarshal(data, &cfg)
		if err != nil {
			return cfg, fmt.Errorf("failed to unmarshal %s: %w", goinferCfgFile, err)
		}
	}

	// Load environment variables
	if dir := os.Getenv("GI_MODELS_DIR"); dir != "" {
		cfg.ModelsDir = dir
		if state.Verbose {
			fmt.Printf("INFO: GI_MODELS_DIR set to %s\n", dir)
		}
	}

	if origins := os.Getenv("GI_ORIGINS"); origins != "" {
		cfg.Server.Origins = origins
		if state.Verbose {
			fmt.Printf("INFO: GI_ORIGINS set to %s\n", origins)
		}
	}

	// Initialize API keys if empty
	if cfg.Server.APIKeys == nil {
		cfg.Server.APIKeys = make(map[string]string)
	}

	// Load API keys from environment
	if key := os.Getenv("GI_API_KEY_ADMIN"); key != "" {
		cfg.Server.APIKeys["admin"] = key
		if state.Verbose {
			fmt.Printf("INFO: GI_API_KEY_ADMIN set\n")
		}
	}

	if key := os.Getenv("GI_API_KEY_USER"); key != "" {
		cfg.Server.APIKeys["user"] = key
		if state.Verbose {
			fmt.Printf("INFO: GI_API_KEY_USER set\n")
		}
	}

	// Validate configuration
	err = validateCfg(cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to validate %s: %w", goinferCfgFile, err)
	}

	return cfg, nil
}

func validateCfg(config *GoInferCfg) error {
	// Ensure admin API key exists
	if _, exists := config.Server.APIKeys["admin"]; !exists {
		return errors.New("admin API key is missing")
	}

	// Validate API keys
	for k, v := range config.Server.APIKeys {
		if v == pleaseSetSecureAPIKey {
			return fmt.Errorf("please set your private '%s' API key", k)
		}
		if len(v) < 64 {
			return fmt.Errorf("invalid API key '%s': must be 64 hex digits", k)
		}
		if v == debugAPIKey {
			fmt.Printf("WARN: api_key[%s]=DEBUG => security threat\n", k)
		}
	}

	return nil
}

func GenerateAPIKey(debugMode bool) ([]byte, error) {
	if debugMode {
		return []byte(debugAPIKey), nil
	}

	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	apiKey := make([]byte, 64)
	hex.Encode(apiKey, buf)
	return apiKey, nil
}

// Create configuration file.
func Create(goinferCfgFile string, debugMode bool) error {
	cfg := []byte(defaultGoInferConf)

	// Set API keys
	key, err := GenerateAPIKey(debugMode)
	if err != nil {
		return fmt.Errorf("failed to generate first API key: %w", err)
	}
	cfg = bytes.Replace(cfg, []byte(pleaseSetSecureAPIKey), key, 1)

	key, err = GenerateAPIKey(debugMode)
	if err != nil {
		return fmt.Errorf("failed to generate second API key: %w", err)
	}
	cfg = bytes.Replace(cfg, []byte(pleaseSetSecureAPIKey), key, 1)

	err = os.WriteFile(goinferCfgFile, cfg, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if debugMode {
		fmt.Printf("WARNING: Configuration file %s created with DEBUG api key. This is not suitable for production use.\n", goinferCfgFile)
	} else {
		fmt.Printf("Configuration file %s created successfully with secure API keys.\n", goinferCfgFile)
	}

	return nil
}

// Print configuration.
func (cfg *GoInferCfg) Print() {
	fmt.Println("-----------------------------")
	fmt.Println("Environment Variables:")
	fmt.Printf("  GI_MODELS_DIR    = %s\n", os.Getenv("GI_MODELS_DIR"))
	fmt.Printf("  GI_ORIGINS       = %s\n", os.Getenv("GI_ORIGINS"))
	fmt.Printf("  GI_API_KEY_ADMIN = set\n")
	fmt.Printf("  GI_API_KEY_USER  = set\n")
	fmt.Println("-----------------------------")

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		fmt.Printf("ERROR yaml.Marshal: %s\n", err.Error())
		return
	}

	os.Stdout.Write(yml)
}

// GetAPIKey with preference order.
func GetAPIKey(apiKeys map[string]string, preferred string) string {
	if key, exists := apiKeys[preferred]; exists {
		return key
	}

	if key, exists := apiKeys["user"]; exists {
		return key
	}

	return apiKeys["admin"]
}

// GenerateProxyCfg generates the llama-swap-proxy configuration.
func GenerateProxyCfg(cfg *GoInferCfg, proxyCfgFile string) error {
	modelFiles, err := models.Dir(cfg.ModelsDir).Search()
	if err != nil {
		return fmt.Errorf("failed to find model files: %w", err)
	}

	if len(modelFiles) == 0 {
		return fmt.Errorf("no model files found in directory: %s", cfg.ModelsDir)
	}

	for _, model := range modelFiles {
		base := filepath.Base(model)
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)

		// OpenAI API
		if state.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				fmt.Printf("Overwrite model=%s in %s\n", stem, proxyCfgFile)
			}
		}
		cfg.Proxy.Models[stem] = proxy.ModelConfig{
			Cmd:          "${llama-server-openai} -m " + model,
			Unlisted:     false,
			UseModelName: stem,
		}

		// GoInfer API: hide the model + prefix GI_
		prefixedModelName := "GI_" + stem
		if state.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				fmt.Printf("Overwrite model=%s in %s\n", stem, proxyCfgFile)
			}
		}
		cfg.Proxy.Models[prefixedModelName] = proxy.ModelConfig{
			Cmd:          "${llama-server-goinfer} -m " + model,
			Unlisted:     true,
			UseModelName: prefixedModelName,
		}
	}

	yml, err := yaml.Marshal(&cfg.Proxy)
	if err != nil {
		return fmt.Errorf("failed to marshal the llama-swap-proxy config: %w", err)
	}

	err = os.WriteFile(proxyCfgFile, yml, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", proxyCfgFile, err)
	}

	if state.Verbose {
		fmt.Printf("Generated %s with %d models\n", proxyCfgFile, len(modelFiles))
	}

	return nil
}
