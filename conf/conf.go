package conf

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"github.com/mostlygeek/llama-swap/proxy"

	"gopkg.in/yaml.v3"
)

const DefaultGoInferConf = `
models_dir: ./models

server:
	api_key:
		# ⚠️ Set 64-byte secure API keys 🚨
		admin: "PLEASE SET SECURE API KEY"
		user:  "PLEASE SET SECURE API KEY"
	origins: "localhost"
	ports:
		admin:   "9999"
		goinfer: "2222"
		mcp:     "3333"
		openai:  "5143"

llama:
	exe: ./llama-server
	args:
		// --props: enable changing global properties via POST /props
		// --no-webui: no Web UI server
		common: --props --no-webui --no-warmup
		goinfer: --jinja --chat-template-file template.jinja
`

// GoInferConf holds the configuration for GoInfer.
type GoInferConf struct {
	ModelsDir string        `json:"models_dir" yaml:"models_dir"`
	Server    ServerConf    `json:"server"     yaml:"server"`
	Llama     LlamaConf     `json:"llama"      yaml:"llama"`
	Swap      *proxy.Config `json:"swap"       yaml:"swap"`
}

// ServerConf holds the configuration for GoInfer web server.
type ServerConf struct {
	Origins string            `json:"origins"    yaml:"origins"`
	Port    map[string]string `json:"port"       yaml:"port"`
	ApiKeys map[string]string `json:"api_key"    yaml:"api_key"`
}

// LlamaConf - configuration for llama-server proxy.
type LlamaConf struct {
	Exe  string            `json:"exe"       yaml:"exe"`       // Path to llama-server binary
	Args map[string]string `json:"args"           yaml:"args"` // Additional arguments
}

// Load the goinfer config file
func Load(goinferFile string, swapFile string) (*GoInferConf, error) {
	var cfg GoInferConf

	// Default values
	err := yaml.Unmarshal([]byte(DefaultGoInferConf), &cfg)
	if err != nil {
		return nil, fmt.Errorf("Error yaml.Unmarshal(DefaultGoInferConf) %w", err)
	}

	// Config file
	bytes, err := os.ReadFile(goinferFile)
	if err != nil {
		return nil, fmt.Errorf("Error os.ReadFile(%s) %w", goinferFile, err)
	}
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("Error yaml.Unmarshal(%s) %w", goinferFile, err)
	}

	// Env. vars
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
	cfg.Swap, err = proxy.LoadConfig(swapFile)
	if err != nil {
		return nil, fmt.Errorf("Error LoadConfig(%s) %w\n", swapFile, err)
	}

	err = CheckValues(&cfg)
	return &cfg, err
}

// CheckValues will check other values, for the moment only API keys
func CheckValues(cfg *GoInferConf) error {
	err := errors.New("missing a seriously secured server.api_key.admin")
	for k, v := range cfg.Server.ApiKeys {
		if len(v) < 64 {
			return errors.New("secured api_key must be 64 bytes: " + v)
		}
		if v == DebugApiKey {
			fmt.Print("WARNING: Config uses DEBUG api_key => security threat")
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
	// Set API keys
	bytes.Replace(cfg, []byte("PLEASE SET SECURE API KEY"), GenApiKey(random), 1)
	bytes.Replace(cfg, []byte("PLEASE SET SECURE API KEY"), GenApiKey(random), 1)
	err := os.WriteFile(fileName, cfg, 0644)
	return err
}

// Print prints viper debug info and the configuration to stdout in YAML format
func (cfg *GoInferConf) Print() {

	// Env. vars
	fmt.Println("GI_MODELS_DIR    = " + os.Getenv("GI_MODELS_DIR"))
	fmt.Println("GI_ORIGINS       = " + os.Getenv("GI_ORIGINS"))
	fmt.Println("GI_API_KEY_ADMIN = " + os.Getenv("GI_API_KEY_ADMIN"))
	fmt.Println("GI_API_KEY_USER  = " + os.Getenv("GI_API_KEY_USER"))

	// Marshal the configuration to YAML
	bytes, err := yaml.Marshal(&cfg)
	if err != nil {
		fmt.Println("Error yaml.Marshal: " + err.Error())
	}

	// Print conf
	_, _ = os.Stdout.Write(bytes)
}
