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

// InitConf loads the config file.
// Does not include extension.
func InitConf(path, configFile string) (types.GoInferConf, error) {
	viper.SetConfigName(configFile)
	viper.AddConfigPath(path)
	viper.SetDefault("origins", []string{"localhost"})
	viper.SetDefault("tasks_dir", "./tasks")
	viper.SetDefault("oai.enable", false)
	viper.SetDefault("oai.threads", 4)
	viper.SetDefault("oai.template", "{system}\n\n{prompt}")

	// Llama configuration defaults
	viper.SetDefault("llama.binary_path", "")
	viper.SetDefault("llama.model_path", "")
	viper.SetDefault("llama.host", "localhost")
	viper.SetDefault("llama.port", 8080)
	viper.SetDefault("llama.args", []string{})

	err := viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return types.GoInferConf{}, fmt.Errorf("No config file %s/%s.??? found: %w", path, configFile, err)
		} else {
			return types.GoInferConf{}, fmt.Errorf("Error inside config file %s/%s.???: %w", path, configFile, err)
		}
	}

	md := viper.GetString("models_dir")
	td := viper.GetString("tasks_dir")
	or := viper.GetStringSlice("origins")
	ak := viper.GetString("api_key")
	oaiEnable := viper.GetBool("oai.enable")
	oaiThreads := viper.GetInt("oai.threads")
	oaiTemplate := viper.GetString("oai.template")

	// Llama configuration
	llamaBinaryPath := viper.GetString("llama.binary_path")
	llamaModelPath := viper.GetString("llama.model_path")
	llamaHost := viper.GetString("llama.host")
	llamaPort := viper.GetInt("llama.port")
	llamaArgs := viper.GetStringSlice("llama.args")

	return types.GoInferConf{
		ModelsDir: md,
		TasksDir:  td,
		Origins:   or,
		ApiKey:    ak,
		OpenAiConf: types.OpenAiConf{
			Enable:   oaiEnable,
			Threads:  oaiThreads,
			Template: oaiTemplate,
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

// Create : create a config file.
func Create(modelsDir string, isDefault bool) {
	CreateWithFileName(modelsDir, isDefault, "goinfer.json")
}

// CreateWithFileName : create a config file with a specific name.
func CreateWithFileName(modelsDir string, isDefault bool, fileName string) {
	key := "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"
	if !isDefault {
		key = generateRandomKey()
	}

	data := map[string]any{
		"models_dir": modelsDir,
		"origins":    []string{"http://localhost:5173", "http://localhost:5143"},
		"api_key":    key,
		"tasks_dir":  "./tasks",
		// Llama configuration defaults
		"llama": map[string]any{
			"binary_path": "",
			"model_path":  "",
			"host":        "localhost",
			"port":        8080,
			"args":        []string{},
		},
	}
	jsonString, _ := json.MarshalIndent(data, "", "    ")
	os.WriteFile(fileName, jsonString, os.ModePerm&^0o111)
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
