package conf

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/viper"
	"github.com/synw/goinfer/types"
)

// goInferConf holds the configuration for GoInfer.
type goInferConf struct {
	ModelsDir   string
	WebServer   WebServerConf
	LlamaConfig *types.LlamaConfig
}

// WebServerConf holds the configuration for GoInfer web server.
type WebServerConf struct {
	Port            string
	EnableApiOpenAi bool `json:"enableApiOpenAi"`
	Origins         []string
	ApiKey          string
}

// InitConf loads the config file.
// Does not include extension.
func InitConf(path, configFile string) (goInferConf, error) {
	viper.SetConfigName(configFile)
	viper.AddConfigPath(path)
	viper.SetDefault("origins", []string{"localhost"})
	viper.SetDefault("openai_api", false)

	// Llama configuration defaults
	viper.SetDefault("llama.binary_path", "")
	viper.SetDefault("llama.model_path", "")
	viper.SetDefault("llama.host", "localhost")
	viper.SetDefault("llama.port", 8080)
	viper.SetDefault("llama.args", []string{})

	err := viper.ReadInConfig()
	if err != nil {
		return goInferConf{}, fmt.Errorf("config file %s/%s.???: %w", path, configFile, err)
	}

	md := viper.GetString("models_dir")
	or := viper.GetStringSlice("origins")
	ak := viper.GetString("api_key")
	oaiEnable := viper.GetBool("openai_api")

	// Llama configuration
	llamaBinaryPath := viper.GetString("llama.binary_path")
	llamaModelPath := viper.GetString("llama.model_path")
	llamaHost := viper.GetString("llama.host")
	llamaPort := viper.GetInt("llama.port")
	llamaArgs := viper.GetStringSlice("llama.args")

	return goInferConf{
		ModelsDir: md,
		WebServer: WebServerConf{
			Port:            ":5143",
			Origins:         or,
			ApiKey:          ak,
			EnableApiOpenAi: oaiEnable,
		},
		LlamaConfig: &types.LlamaConfig{
			BinaryPath: llamaBinaryPath,
			ModelPath:  llamaModelPath,
			Host:       llamaHost,
			Port:       llamaPort,
			Args:       llamaArgs,
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
		"models_dir": modelsDir,
		"origins":    []string{"http://localhost:5173", "http://localhost:5143"},
		"api_key":    key,
		"llama": map[string]any{
			"binary_path": "",
			"model_path":  "",
			"host":        "localhost",
			"port":        8080,
			"args":        []string{},
		},
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
