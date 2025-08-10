package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/types"
)

// the language model instance.
var (
	Llama   *llama.LlamaServerManager
	Monitor *llama.Monitor
)

var (
	ErrModelNotFound = errors.New("model not found")
	ErrInvalidInput  = errors.New("invalid input")
)

func StartLlamaServer() error {
	if Llama == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	return Llama.Start()
}

func StopLlamaServer() error {
	// Llama not initialized => llama-server already stopped (never started)
	if Llama == nil {
		return nil
	}

	return Llama.Stop()
}

func RestartLlamaServer() error {
	if Llama == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	return Llama.Restart()
}

// GetServerStatus - Gets the current server status.
func GetServerStatus() (bool, time.Duration, int) {
	if Llama == nil {
		return false, 0, 0
	}

	return Llama.IsRunning(), Llama.GetUptime(), Llama.GetStartCount()
}

// CheckServerHealth - Performs a health check on the server.
func CheckServerHealth() bool {
	if Llama == nil {
		return false
	}

	return Llama.HealthCheck()
}

// UnloadModel unloads the currently loaded model.
func UnloadModel() {
	IsModelLoaded = false
	LoadedModel = ""
}

// StartLlamaWithModel returns HTTP status code + Go error.
func StartLlamaWithModel(modelConf types.ModelConf) (int, error) {
	if modelConf.Name == "" {
		return 400, fmt.Errorf("model name cannot be empty: %w", ErrInvalidInput)
	}

	filepath := filepath.Join(ModelsDir, modelConf.Name)
	// check if the model file exists
	_, err := os.Stat(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return 404, fmt.Errorf("the model file %s does not exist: %w", filepath, ErrModelNotFound)
		}
		return 500, fmt.Errorf("error checking model file %s: %w", filepath, err)
	}
	// check if the model is already loaded
	if LoadedModel == modelConf.Name {
		return 202, fmt.Errorf("the model is already loaded: %w", ErrInvalidInput)
	}

	if IsVerbose || IsDebug {
		fmt.Println("Loaded model", filepath)
		if IsDebug {
			jsonData, err := json.MarshalIndent(modelConf, "", "  ")
			if err != nil {
				return 500, fmt.Errorf("error marshalling model params: %w", err)
			}
			fmt.Println(string(jsonData))
		}
	}

	RestartLlamaServer()

	ModelConf = modelConf
	IsModelLoaded = true
	LoadedModel = modelConf.Name

	return 200, nil
}
