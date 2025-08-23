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

	"github.com/mostlygeek/llama-swap/proxy"
	"github.com/synw/goinfer/models"
	"github.com/synw/goinfer/state"

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
	Verbose   bool         `json:"verbose,omitempty"    yaml:"verbose,omitempty"`
	ModelsDir string       `json:"models_dir,omitempty" yaml:"models_dir,omitempty"` // one or multiple directories separated by ':'
	Server    ServerConf   `json:"server,omitempty"     yaml:"server,omitempty"`     // HTTP server
	Llama     LlamaConf    `json:"llama,omitempty"      yaml:"llama,omitempty"`      // llama.cpp
	Proxy     proxy.Config `json:"proxy,omitempty"      yaml:"proxy,omitempty"`      // llama-swap proxy
}

// ServerConf = config for the GoInfer http server.
type ServerConf struct {
	Listen  map[string]string `json:"listen,omitempty"  yaml:"listen,omitempty"`
	ApiKeys map[string]string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Origins string            `json:"origins,omitempty" yaml:"origins,omitempty"`
}

// LlamaConf - configuration for llama-server proxy.
type LlamaConf struct {
	Exe  string            `json:"exe,omitempty"  yaml:"exe,omitempty"`  // Path to llama-server binary
	Args map[string]string `json:"args,omitempty" yaml:"args,omitempty"` // llama-server arguments
}

// Load the goinfer config file
func Load(goinferCfgFile string) GoInferConf {
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
			return errors.New("secured api_key must be 64 hexadecimal digits: " + v)
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
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
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
	_, _ = os.Stdout.Write(bytes)
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

// Generate llama-swap config
func GenProxyConfFromModelFiles(cfg *GoInferConf, proxyCfgFile string) {
	//	bytes, err := os.ReadFile(proxyCfgFile)
	//	if err != nil {
	//		fmt.Printf("WARNING os.ReadFile(%s) %v => Ignore config file\n", proxyCfgFile, err)
	//	} else {
	//		err := yaml.Unmarshal(bytes, &cfg.Proxy)
	//		if err != nil {
	//			fmt.Printf("WARNING yaml.Unmarshal(%s) %v => Ignore config file\n", proxyCfgFile, err)
	//		}
	//	}

	modelFiles, err := models.Dir(cfg.ModelsDir).Search()
	if err != nil {
		fmt.Println("ERROR while searching model files:", err)
		return
	}

	if len(modelFiles) == 0 {
		fmt.Println("WARNING Found zero model file => Do not generate", proxyCfgFile)
		return
	}

	for _, m := range modelFiles {
		base := filepath.Base(m)  // Keep the filename without the directory
		ext := filepath.Ext(base) // Get the extension
		stem := strings.TrimSuffix(base, ext)

		// for OpenAI API: list the models
		if state.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				fmt.Printf("Overwrite model=%s in %s\n", stem, proxyCfgFile)
			}
		}
		cfg.Proxy.Models[stem] = proxy.ModelConfig{
			Cmd:          "${llama-server-openai} --model " + m,
			Unlisted:     false,
			UseModelName: stem,
		}

		// for goinfer API: hide an prefix models with GI_
		stem = "GI_" + stem
		if state.Verbose {
			_, ok := cfg.Proxy.Models[stem]
			if ok {
				fmt.Printf("Overwrite model=%s in %s\n", stem, proxyCfgFile)
			}
		}
		cfg.Proxy.Models[stem] = proxy.ModelConfig{
			Cmd:          "${llama-server-goinfer} --model " + m,
			Unlisted:     true,
			UseModelName: stem,
		}
	}

	// Marshal the configuration to YAML
	bytes, err := yaml.Marshal(&cfg.Proxy)
	if err != nil {
		fmt.Println("ERROR yaml.Marshal: " + err.Error())
		return
	}

	err = os.WriteFile(proxyCfgFile, bytes, 0644)
	if err != nil {
		fmt.Println("ERROR os.WriteFile(" + proxyCfgFile + "): " + err.Error())
		return
	}

	fmt.Printf("File %s generated from %d model files found in subdirectories: %s",
		proxyCfgFile, len(modelFiles), cfg.ModelsDir)
}
