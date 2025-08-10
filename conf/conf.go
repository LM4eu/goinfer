package conf

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// GoInferConf holds the configuration for GoInfer.
type GoInferConf struct {
	WebServer WebServerConf
	ModelsDir string
	Llama     LlamaConf
}

// WebServerConf holds the configuration for GoInfer web server.
type WebServerConf struct {
	Origins      []string `json:"origins"`
	Port         string   `json:"port"`
	EnableOaiAPI bool     `json:"openai_api"`
	ApiKey       string   `json:"api_key"`
}

// InitConf loads the config file.
// Does not include extension.
func InitConf(path, configFile string) (GoInferConf, error) {
	viper.SetConfigName(configFile)
	viper.AddConfigPath(path)

	viper.SetDefault("origins", []string{"localhost"})
	viper.SetDefault("port", 5143)
	viper.SetDefault("openai_api", false)

	viper.SetDefault("models_dir", "./models")
	viper.SetDefault("ctx", 2048)
	viper.SetDefault("gpu_layers", 999)
	viper.SetDefault("flash_attention", true)

	viper.SetDefault("llama_path", "./llama-server")
	viper.SetDefault("llama_host", "localhost")
	viper.SetDefault("llama_port", 8080)
	viper.SetDefault("llama_webui", false)
	viper.SetDefault("llama_threads", 8)
	viper.SetDefault("llama_thrPromptProc", 16)
	viper.SetDefault("llama_args", []string{"--log-colors", "--no-warmup"})

	err := viper.ReadInConfig()
	if err != nil {
		return GoInferConf{}, fmt.Errorf("config file %s/%s.(json/yaml): %w", path, configFile, err)
	}

	return GoInferConf{
		WebServer: WebServerConf{
			Origins:      viper.GetStringSlice("origins"),
			Port:         viper.GetString("port"),
			EnableOaiAPI: viper.GetBool("openai_api"),
			ApiKey:       viper.GetString("api_key"),
		},
		ModelsDir: viper.GetString("models_dir"),
		Llama: LlamaConf{
			ModelPath:      viper.GetString("default_model"),
			ContextSize:    viper.GetInt("ctx"),
			GpuLayers:      viper.GetInt("gpu_layers"),
			FlashAttention: viper.GetBool("flash_attention"),

			BinaryPath:   viper.GetString("llama_path"),
			Host:         viper.GetString("llama_host"),
			Port:         viper.GetInt("llama_port"),
			WebUI:        viper.GetBool("llama_webui"),
			Threads:      viper.GetInt("llama_threads"),
			ThPromptProc: viper.GetInt("llama_thrPromptProc"),
			Args:         viper.GetStringSlice("llama_args"),
		},
	}, nil
}

// Create : create a config file
func Create(modelsDir string, isDefault bool, fileName string) {
	key := "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"
	if !isDefault {
		key = generateRandomKey()
	}

	// configuration defaults
	data := map[string]any{
		"origins":    []string{"http://localhost:5173", "http://localhost:5143"},
		"port":       "5143",
		"openai_api": false,
		"api_key":    key,

		"models_dir":      modelsDir,
		"default_model":   "",
		"download_url":    "",
		"ctx":             2048,
		"gpu_layers":      999,
		"flash_attention": true,

		"llama_path":          "./llama-server",
		"llama_host":          "localhost",
		"llama_port":          8080,
		"llama_threads":       8,
		"llama_thrPromptProc": 16,
		"llama_args":          []string{"--log-colors", "--no-warmup"},
	}
	jsonString, _ := json.MarshalIndent(data, "", "    ")
	err := os.WriteFile(fileName, jsonString, os.ModePerm&^0o111)
	if err != nil {
		fmt.Printf("Cannot write %s - %v", fileName, err)
	}
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
