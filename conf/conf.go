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

func InitConf() types.GoInferConf {
	viper.SetConfigName("goinfer.config")
	viper.AddConfigPath(".")
	viper.SetDefault("origins", []string{"localhost"})
	viper.SetDefault("tasks_dir", "./tasks")
	viper.SetDefault("oai.enable", false)
	viper.SetDefault("oai.threads", 4)
	viper.SetDefault("oai.template", "{system}\n\n{prompt}")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	md := viper.GetString("models_dir")
	td := viper.GetString("tasks_dir")
	or := viper.GetStringSlice("origins")
	ak := viper.GetString("api_key")
	oaiEnable := viper.GetBool("oai.enable")
	oaiThreads := viper.GetInt("oai.threads")
	oaiTemplate := viper.GetString("oai.template")

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
	}
}

// Create : create a config file.
func Create(modelsDir string, isDefault bool) {
	CreateWithFileName(modelsDir, isDefault, "goinfer.config.json")
}

// CreateWithFileName : create a config file with a specific name.
func CreateWithFileName(modelsDir string, isDefault bool, fileName string) {
	key := "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"
	if !isDefault {
		key = generateRandomKey()
	}

	data := map[string]interface{}{
		"models_dir": modelsDir,
		"origins":    []string{"http://localhost:5173", "http://localhost:5143"},
		"api_key":    key,
		"tasks_dir":  "./tasks",
	}
	jsonString, _ := json.MarshalIndent(data, "", "    ")
	os.WriteFile(fileName, jsonString, os.ModePerm&^0o111)
}

func generateRandomKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err.Error())
	}
	key := hex.EncodeToString(bytes)
	return key
}
