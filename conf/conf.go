// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LM4eu/goinfer/gie"
	"github.com/LM4eu/goinfer/models"
	"github.com/LM4eu/goinfer/state"
	"github.com/mostlygeek/llama-swap/proxy"

	"gopkg.in/yaml.v3"
)

type (
	GoInferCfg struct {
		Server    ServerCfg    `json:"server"               yaml:"server"`
		Llama     LlamaCfg     `json:"llama"                yaml:"llama"`
		ModelsDir string       `json:"models_dir,omitempty" yaml:"models_dir,omitempty"`
		Proxy     proxy.Config `json:"proxy"                yaml:"proxy"`
		Verbose   bool         `json:"verbose,omitempty"    yaml:"verbose,omitempty"`
	}

	ServerCfg struct {
		Listen  map[string]string `json:"listen,omitempty"  yaml:"listen,omitempty"`
		APIKeys map[string]string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
		Host    string            `json:"host,omitempty"    yaml:"host,omitempty"`
		Origins string            `json:"origins,omitempty" yaml:"origins,omitempty"`
	}

	LlamaCfg struct {
		Args map[string]string `json:"args,omitempty" yaml:"args,omitempty"`
		Exe  string            `json:"exe,omitempty"  yaml:"exe,omitempty"`
	}
)

const debugAPIKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"

var defaultGoInferConf = GoInferCfg{
	ModelsDir: "./models",
	Server: ServerCfg{
		Listen: map[string]string{
			":8888": "admin",
			":2222": "openai,goinfer,mcp",
			":5143": "llama-swap proxy",
		},
		APIKeys: map[string]string{},
		Host:    "",
		Origins: "localhost",
	},
	Llama: LlamaCfg{
		Exe: "./llama-server",
		Args: map[string]string{
			"common":  "--no-webui --no-warmup",
			"goinfer": "--jinja --chat-template-file template.jinja",
		},
	},
}

// Create a configuration file.
func Create(goinferCfgFile string, debugMode bool) error {
	cfg := defaultGoInferConf
	applyEnvVars(&cfg)

	// Set API keys
	if len(cfg.Server.APIKeys) == 0 {
		cfg.Server.APIKeys["admin"] = genAPIKey(debugMode)
		cfg.Server.APIKeys["user"] = genAPIKey(debugMode)

		if debugMode {
			fmt.Printf("WRN: Configuration file %s with DEBUG api key. This is not suitable for production use.\n", goinferCfgFile)
		} else {
			fmt.Printf("INF: Configuration file %s with secure API keys.\n", goinferCfgFile)
		}
	} else {
		fmt.Printf("INF: Configuration file %s use API keys from environment.\n", goinferCfgFile)
	}

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_MARSHAL", "failed to write config file")
	}

	err = os.WriteFile(goinferCfgFile, yml, 0o600)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "CONFIG_WRITE_FAILED", "failed to write config file")
	}

	return validate(&cfg)
}

// Load the configuration file.
func Load(goinferCfgFile string) (*GoInferCfg, error) {
	var cfg *GoInferCfg

	// Load from file if specified
	if goinferCfgFile != "" {
		yml, err := os.ReadFile(filepath.Clean(goinferCfgFile)) // TODO: Use OpenFileIn() from Go-1.25
		if err != nil {
			return cfg, gie.Wrap(err, gie.TypeConfiguration, "CONFIG_FILE_READ_FAILED", "failed to read "+goinferCfgFile)
		}

		if len(yml) > 0 {
			err := yaml.Unmarshal(yml, &cfg)
			if err != nil {
				return cfg, gie.Wrap(err, gie.TypeConfiguration, "CONFIG_UNMARSHAL_FAILED", "failed to unmarshal YAML data: "+string(yml[:100]))
			}
		}
	}

	applyEnvVars(cfg)

	// Validate configuration
	err := validate(cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func applyEnvVars(cfg *GoInferCfg) {
	// Load environment variables
	if dir := os.Getenv("GI_MODELS_DIR"); dir != "" {
		cfg.ModelsDir = dir
		if state.Verbose {
			fmt.Printf("INF: GI_MODELS_DIR set to %s\n", dir)
		}
	}

	if host := os.Getenv("GI_HOST"); host != "" {
		cfg.Server.Host = host
		if state.Verbose {
			fmt.Printf("INF: GI_HOST set to %s\n", host)
		}
	}

	if origins := os.Getenv("GI_ORIGINS"); origins != "" {
		cfg.Server.Origins = origins
		if state.Verbose {
			fmt.Printf("INF: GI_ORIGINS set to %s\n", origins)
		}
	}

	// Load user API key from environment
	if key := os.Getenv("GI_API_KEY_USER"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 2)
		}
		cfg.Server.APIKeys["user"] = key
		if state.Verbose {
			fmt.Println("INF: api_key[user] = GI_API_KEY_USER")
		}
	}

	// Load admin API key from environment
	if key := os.Getenv("GI_API_KEY_ADMIN"); key != "" {
		if cfg.Server.APIKeys == nil {
			cfg.Server.APIKeys = make(map[string]string, 1)
		}
		cfg.Server.APIKeys["admin"] = key
		if state.Verbose {
			fmt.Println("INF: api_key[admin] = GI_API_KEY_ADMIN")
		}
	}

	if exe := os.Getenv("GI_LLAMA_EXE"); exe != "" {
		cfg.Llama.Exe = exe
		if state.Verbose {
			fmt.Printf("INF: GI_LLAMA_EXE =%s\n", exe)
		}
	}
}

func validate(cfg *GoInferCfg) error {
	modelFiles, err := models.Dir(cfg.ModelsDir).Search()
	if err != nil {
		return err
	}
	if len(modelFiles) == 0 {
		fmt.Printf("WRN: No *.gguf files found in %s\n", cfg.ModelsDir)
	} else if cfg.Verbose {
		fmt.Printf("INF: Found %d model files in %s\n", len(modelFiles), cfg.ModelsDir)
	}

	// Ensure admin API key exists
	if _, exists := cfg.Server.APIKeys["admin"]; !exists {
		return gie.Wrap(gie.ErrAPIKeyMissing, gie.TypeConfiguration, "ADMIN_API_MISSING", "admin API key is missing")
	}

	// Validate API keys
	for k, v := range cfg.Server.APIKeys {
		if strings.Contains(v, "PLEASE") {
			return gie.Wrap(gie.ErrInvalidAPIKey, gie.TypeConfiguration, "API_KEY_NOT_SET", fmt.Sprintf("please set your private '%s' API key", k))
		}
		if len(v) < 64 {
			return gie.Wrap(gie.ErrInvalidAPIKey, gie.TypeConfiguration, "API_KEY_INVALID", fmt.Sprintf("invalid API key '%s': must be 64 hex digits", k))
		}
		if v == debugAPIKey {
			fmt.Printf("WRN: api_key[%s]=DEBUG => security threat\n", k)
		}
	}

	return nil
}

func genAPIKey(debugMode bool) string {
	if debugMode {
		return debugAPIKey
	}

	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		fmt.Printf("WRN: rand.Read %v\n", err)
		return ""
	}

	key := make([]byte, 64)
	hex.Encode(key, buf)
	return string(key)
}

// Print configuration.
func (cfg *GoInferCfg) Print() {
	fmt.Println("-----------------------------")
	fmt.Println("Environment Variables:")
	fmt.Printf("  GI_MODELS_DIR    = %s\n", os.Getenv("GI_MODELS_DIR"))
	fmt.Printf("  GI_HOST          = %s\n", os.Getenv("GI_HOST"))
	fmt.Printf("  GI_ORIGINS       = %s\n", os.Getenv("GI_ORIGINS"))
	fmt.Printf("  GI_API_KEY_ADMIN = %d characters\n", len(os.Getenv("GI_API_KEY_ADMIN")))
	fmt.Printf("  GI_API_KEY_USER  = %d characters\n", len(os.Getenv("GI_API_KEY_USER")))
	fmt.Printf("  GI_LLAMA_EXE     = %s\n", os.Getenv("GI_LLAMA_EXE"))

	fmt.Println("-----------------------------")

	yml, err := yaml.Marshal(&cfg)
	if err != nil {
		fmt.Printf("ERROR yaml.Marshal: %s\n", err.Error())
		return
	}

	os.Stdout.Write(yml)

	fmt.Println("-----------------------------")
}

// GenerateProxyCfg generates the llama-swap-proxy configuration.
func GenProxyCfg(cfg *GoInferCfg, proxyCfgFile string) error {
	modelFiles, err := models.Dir(cfg.ModelsDir).Search()
	if err != nil {
		return err
	}

	if len(modelFiles) == 0 {
		return err
	}

	for _, model := range modelFiles {
		base := filepath.Base(model)
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)

		flags := models.ExtractFlags(model)

		// OpenAI API
		if state.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				fmt.Printf("INF: Overwrite model=%s in %s\n", stem, proxyCfgFile)
			}
		}
		cfg.Proxy.Models[stem] = proxy.ModelConfig{
			Cmd:          "${llama-server-openai} -m " + model + " " + flags,
			Unlisted:     false,
			UseModelName: stem,
		}

		// GoInfer API: hide the model + prefix GI_
		prefixedModelName := "GI_" + stem
		if state.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				fmt.Printf("INF: Overwrite model=%s in %s\n", stem, proxyCfgFile)
			}
		}
		cfg.Proxy.Models[prefixedModelName] = proxy.ModelConfig{
			Cmd:          "${llama-server-goinfer} -m " + model + " " + flags,
			Unlisted:     true,
			UseModelName: prefixedModelName,
		}
	}

	yml, er := yaml.Marshal(&cfg.Proxy)
	if er != nil {
		return gie.Wrap(er, gie.TypeConfiguration, "CONFIG_MARSHAL_FAILED", "failed to marshal the llama-swap-proxy config")
	}

	err = os.WriteFile(proxyCfgFile, yml, 0o600)
	if err != nil {
		return gie.Wrap(err, gie.TypeConfiguration, "PROXY_WRITE_FAILED", "failed to write "+proxyCfgFile)
	}

	if state.Verbose {
		fmt.Printf("INF: Generated %s with %d models\n", proxyCfgFile, len(modelFiles))
	}

	return nil
}
