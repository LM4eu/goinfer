package conf

import (
	"crypto/rand"
	"encoding/hex"
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

// setAllDefaults sets all default configuration values in a centralized manner
func setAllDefaults() {
	// Web server defaults
	viper.SetDefault("server.origins", []string{"localhost"})
	viper.SetDefault("server.port", 5143)
	viper.SetDefault("server.openai_api", false)

	// Model defaults
	viper.SetDefault("model.dir", "./models")
	viper.SetDefault("model.ctx", 2048)

	// Llama defaults
	viper.SetDefault("llama.exe", "./llama-server")
	viper.SetDefault("llama.web_ui", false)
	viper.SetDefault("llama.threads", 8)
	viper.SetDefault("llama.t_prompt", 16)
	viper.SetDefault("llama.args", []string{"--log-colors", "--no-warmup"})
}

// setupViper configures Viper with the given path and config file name
func setupViper(path, configFile string) {
	viper.SetConfigName(configFile)
	viper.AddConfigPath(path)
	viper.AutomaticEnv()
	setAllDefaults()
}

// InitConf loads the config file.
// Does not include extension.
func InitConf(path, configFile string) (GoInferConf, error) {
	setupViper(path, configFile)

	err := viper.ReadInConfig()
	if err != nil {
		return GoInferConf{}, fmt.Errorf("config file %s/%s.(json/yaml): %w", path, configFile, err)
	}

	return GoInferConf{
		Server: WebServerConf{
			Origins:         viper.GetStringSlice("server.origins"),
			Port:            viper.GetString("server.port"),
			EnableOpenAiAPI: viper.GetBool("server.openai_api"),
			ApiKey:          viper.GetString("server.api_key"),
		},
		ModelsDir: viper.GetString("model.dir"),
		Llama: LlamaConf{
			ContextSize: viper.GetInt("model.ctx"),
			ExePath:     viper.GetString("llama.exe"),
			Threads:     viper.GetInt("llama.threads"),
			TPromptProc: viper.GetInt("llama.t_prompt"),
			Args:        viper.GetStringSlice("llama.args"),
		},
	}, nil
}

// Create creates a configuration file using Viper's WriteConfig functionality
func Create(modelsDir string, isDefault bool, fileName string) error {
	// Setup Viper for the target file
	viper.SetConfigFile(fileName) // Set the full file path

	// Set all defaults consistently
	setAllDefaults()

	if modelsDir == "" {
		modelsDir = "./models"
	}
	viper.Set("model.dir", modelsDir)

	// Set origins for web server (different from defaults)
	viper.SetDefault("server.origins", []string{"http://localhost:5173", "http://localhost:5143"})

	// Generate API key if not default
	if !isDefault {
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
	// Marshal the configuration to YAML
	bytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("error Marshal(cfg) to YAML: %w", err)
	}

	viper.Debug()

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
