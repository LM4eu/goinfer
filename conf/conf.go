package conf

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// GoInferConf holds the configuration for GoInfer.
type GoInferConf struct {
	Server    WebServerConf
	ModelsDir string `json:"models_dir" yaml:"models_dir"`
	Llama     LlamaConf
}

// WebServerConf holds the configuration for GoInfer web server.
type WebServerConf struct {
	Origins         []string `json:"origins"    yaml:"origins"`
	Port            string   `json:"port"       yaml:"port"`
	EnableOpenAiAPI bool     `json:"openai_api" yaml:"openai_api"`
	ApiKey          string   `json:"api_key"    yaml:"api_key"`
}

// setDefaultConf sets all default configuration values in a centralized manner
func setDefaultConf() {
	viper.SetDefault("server.origins", []string{"localhost"})
	viper.SetDefault("server.port", 5143)
	viper.SetDefault("server.openai_api", false)
	viper.SetDefault("models_dir", "./models")
	viper.SetDefault("llama.exe_path", "./llama-server")
	viper.SetDefault("llama.threads", 8)
	viper.SetDefault("llama.t_prompt_proc", 16) // more threads to boost prompt processing
	viper.SetDefault("llama.args", []string{"--log-colors", "--no-warmup"})

	// broken AutomaticEnv() since viper-1.19 (Jun 2024)
	// https://github.com/spf13/viper/issues/1895
	// Manual binding:
	_ = viper.BindEnv("server.origins", "SERVER_ORIGINS")
	_ = viper.BindEnv("server.port", "SERVER_PORT")
	_ = viper.BindEnv("server.openai_api", "SERVER_OPENAI_API")
	_ = viper.BindEnv("server.api_key", "SERVER_API_KEY")
	_ = viper.BindEnv("models_dir", "MODELS_DIR")
	_ = viper.BindEnv("llama.exe_path", "LLAMA_EXE_PATH")
	_ = viper.BindEnv("llama.threads", "LLAMA_THREADS")
	_ = viper.BindEnv("llama.t_prompt_proc", "LLAMA_T_PROMPT_PROC")
	_ = viper.BindEnv("llama.args", "LLAMA_ARGS")
}

// Load the config file having any extension: json, yml, ini...
func Load(path, configFile string) (GoInferConf, error) {
	viper.SetConfigName(configFile)
	viper.AddConfigPath(path)

	setDefaultConf() // Set defaults first

	err := viper.ReadInConfig()
	if err != nil {
		return GoInferConf{}, fmt.Errorf("config file %s/%s.(json/yaml): %w", path, configFile, err)
	}

	cfg := GoInferConf{
		Server: WebServerConf{
			Origins:         viper.GetStringSlice("server.origins"),
			Port:            viper.GetString("server.port"),
			EnableOpenAiAPI: viper.GetBool("server.openai_api"),
			ApiKey:          viper.GetString("server.api_key"),
		},
		ModelsDir: viper.GetString("models_dir"),
		Llama: LlamaConf{
			ExePath:     viper.GetString("llama.exe_path"),
			Threads:     viper.GetInt("llama.threads"),
			TPromptProc: viper.GetInt("llama.t_prompt_proc"), // more threads to boost prompt processing
			Args:        viper.GetStringSlice("llama.args"),
		},
	}

	if cfg.Server.ApiKey == "" {
		return cfg, errors.New("missing mandatory server.api_key in " + configFile +
			" (use -conf or -localconf to generate a default config file)")
	}

	return cfg, nil
}

// Create a YAML configuration file using Viper's WriteConfig functionality
func Create(fileName string, random bool) error {
	viper.SetConfigFile(fileName) // full file path
	setDefaultConf()

	// Set origins for web server (different from defaults)
	viper.SetDefault("server.origins", []string{"http://localhost:5173", "http://localhost:5143"})

	if random {
		viper.SetDefault("server.api_key", generateRandomKey())
	} else {
		viper.SetDefault("server.api_key", "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465")
	}

	// Write the configuration file using Viper
	if err := viper.WriteConfigAs(fileName); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", fileName, err)
	}

	return nil
}

// Debug prints viper debug info and the configuration to stdout in YAML format
func (cfg *GoInferConf) Debug() error {
	viper.Debug()

	// Marshal the configuration to YAML
	bytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("error Marshal(cfg) to YAML: %w", err)
	}

	// Print to stdout
	_, err = os.Stdout.Write(bytes)
	return err
}

func generateRandomKey() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err.Error())
	}
	key := hex.EncodeToString(bytes)
	return key
}
