package conf

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type GoInferConf struct {
	ModelsDir string
	TasksDir  string
	Origins   []string
	ApiKey    string
}

func InitConf() GoInferConf {
	viper.SetConfigName("goinfer.config")
	viper.AddConfigPath(".")
	viper.SetDefault("origins", []string{"localhost"})
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	md := viper.GetString("models_dir")
	td := viper.GetString("tasks_dir")
	or := viper.GetStringSlice("origins")
	ak := viper.GetString("api_key")
	return GoInferConf{
		ModelsDir: md,
		TasksDir:  td,
		Origins:   or,
		ApiKey:    ak,
	}
}

// Create : create a config file
func Create() {
	data := map[string]interface{}{
		"models_dir": "",
		"origins":    []string{"http://localhost:5173", "http://localhost:5143"},
		"api_key":    generateRandomKey(),
		"tasks_dir":  "./tasks",
	}
	jsonString, _ := json.MarshalIndent(data, "", "    ")
	os.WriteFile("goinfer.config.json", jsonString, os.ModePerm)
}

func generateRandomKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err.Error())
	}
	key := hex.EncodeToString(bytes)
	return key
}
