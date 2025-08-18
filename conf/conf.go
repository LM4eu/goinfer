package conf

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/mostlygeek/llama-swap/proxy"

	"gopkg.in/yaml.v3"
)

const DefaultGoInferConf = `# Configuration of https://github.com/LM4eu/goinfer

# Recursively search *.gguf files (one or multiple directories separated by ':')
models_dir: ./models

server:
  api_key:
    # ⚠️ Set 64-byte secure API keys 🚨
    "admin": PLEASE SET SECURE API KEY
    "user":  PLEASE SET SECURE API KEY
  origins: localhost
  listen:
    ":8080": admin
    ":5143": openai,goinfer,mcp

llama:
  exe: ./llama-server
  args:
    # --props: enable changing global properties via POST /props
    # --no-webui: no Web UI server
    "common": --props --no-webui --no-warmup
    "goinfer": --jinja --chat-template-file template.jinja
`

// GoInferConf holds the configuration for GoInfer.
type GoInferConf struct {
	Verbose   bool         `json:"verbose"    yaml:"verbose"`
	ModelsDir string       `json:"models_dir" yaml:"models_dir"` // one or multiple directories separated by ':'
	Server    ServerConf   `json:"server"     yaml:"server"`     // HTTP server
	Llama     LlamaConf    `json:"llama"      yaml:"llama"`      // llama.cpp
	Proxy     proxy.Config `json:"proxy"      yaml:"proxy"`      // llama-swap proxy
}

// ServerConf = config for the GoInfer http server.
type ServerConf struct {
	Listen  map[string]string `json:"listen"         yaml:"listen"`
	ApiKeys map[string]string `json:"api_key"        yaml:"api_key"`
	Origins string            `json:"origins"        yaml:"origins"`
}

// LlamaConf - configuration for llama-server proxy.
type LlamaConf struct {
	Exe  string            `json:"exe"  yaml:"exe"`  // Path to llama-server binary
	Args map[string]string `json:"args" yaml:"args"` // llama-server arguments
}

// Load the goinfer config file
func Load(goinferCfgFile string, proxyCfgFile string) GoInferConf {
	var cfg GoInferConf

	// Default values
	err := yaml.Unmarshal([]byte(DefaultGoInferConf), &cfg)
	if err != nil {
		panic(fmt.Errorf("error yaml.Unmarshal(DefaultGoInferConf) %w", err))
	}

	// Config file
	bytes, err := os.ReadFile(goinferCfgFile)
	if err != nil {
		fmt.Printf("WARNING os.ReadFile(%s) %v => Ignore config file\n", goinferCfgFile, err)
	} else {
		err := yaml.Unmarshal(bytes, &cfg)
		if err != nil {
			panic(fmt.Errorf("error yaml.Unmarshal(%s) %w", goinferCfgFile, err))
		}
	}

	// Env. vars (prefix GI = GoInfer)
	if dir, ok := os.LookupEnv("GI_MODELS_DIR"); ok {
		cfg.ModelsDir = dir
	}
	if dir, ok := os.LookupEnv("GI_ORIGINS"); ok {
		cfg.Server.Origins = dir
	}
	if apiKey, ok := os.LookupEnv("GI_API_KEY_ADMIN"); ok {
		cfg.Server.ApiKeys["admin"] = apiKey
	}
	if apiKey, ok := os.LookupEnv("GI_API_KEY_USER"); ok {
		cfg.Server.ApiKeys["user"] = apiKey
	}

	// Load also the llama-swap config
	cfg.Proxy, err = proxy.LoadConfig(proxyCfgFile)
	if err != nil {
		panic(fmt.Errorf("error LoadConfig(%s) %w", proxyCfgFile, err))
	}

	err = CheckValues(&cfg)
	if err != nil {
		panic(fmt.Errorf("error CheckValues(%s) %w", goinferCfgFile, err))
	}

	return cfg
}

// CheckValues will check other values, for the moment only API keys
func CheckValues(cfg *GoInferConf) error {
	err := errors.New("missing a seriously secured server.api_key.admin")
	for k, v := range cfg.Server.ApiKeys {
		if len(v) < 64 {
			return errors.New("secured api_key must be 64 bytes: " + v)
		}
		if v == DebugApiKey {
			fmt.Printf("WARNING api_key[%s]=DEBUG => security threat\n", k)
		}
		if k == "admin" {
			err = nil
		}
	}
	return err
}

const DebugApiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"

func GenApiKey(debug bool) []byte {
	if debug {
		return []byte(DebugApiKey)
	}
	bytes := make([]byte, 32)
	rand.Read(bytes)
	apiKey := make([]byte, 64)
	hex.Encode(apiKey, bytes)
	return apiKey
}

// Create a YAML configuration
func Create(goinferCfgFile string, debug bool) {
	cfg := []byte(DefaultGoInferConf)

	// Set API keys
	cfg = bytes.Replace(cfg, []byte("PLEASE SET SECURE API KEY"), GenApiKey(debug), 1)
	cfg = bytes.Replace(cfg, []byte("PLEASE SET SECURE API KEY"), GenApiKey(debug), 1)
	err := os.WriteFile(goinferCfgFile, cfg, 0600)

	if err != nil {
		fmt.Printf("WARNING os.WriteFile(%s) %v\n", goinferCfgFile, err)
	} else if debug {
		fmt.Println("File " + goinferCfgFile + " created with DEBUG api key")
	} else {
		fmt.Println("File " + goinferCfgFile + " created with RANDOM api key")
	}
}

// Print prints viper debug info and the configuration to stdout in YAML format
func (cfg *GoInferConf) Print() {
	// Env. vars
	fmt.Println("-----------------------------")
	fmt.Println("GI_MODELS_DIR    = " + os.Getenv("GI_MODELS_DIR"))
	fmt.Println("GI_ORIGINS       = " + os.Getenv("GI_ORIGINS"))
	fmt.Println("GI_API_KEY_ADMIN = " + os.Getenv("GI_API_KEY_ADMIN"))
	fmt.Println("GI_API_KEY_USER  = " + os.Getenv("GI_API_KEY_USER"))
	fmt.Println("-----------------------------")

	// Marshal the configuration to YAML
	bytes, err := yaml.Marshal(&cfg)
	if err != nil {
		fmt.Println("ERROR yaml.Marshal: " + err.Error())
		return
	}

	// Print the YAML
	os.Stdout.Write(bytes)
}

func ApiKey(keys map[string]string, favorite string) string {
	k, ok := keys[favorite]
	if ok {
		return k
	}
	k, ok = keys["user"]
	if ok {
		return k
	}
	return keys["admin"]
}
