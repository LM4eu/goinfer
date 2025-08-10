package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/synw/goinfer/conf"
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

func RestartLlamaServer(modelConf types.ModelConf) error {
	if Llama == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	modelPath := modelConf.Name
	if conf.IsDownloadURL(modelConf.Name) != 0 {
		path, err := searchModelFile(modelConf)
		if err != nil {
			return err
		}
		modelPath = path
	}

	Llama.Conf.ModelPath = modelPath
	Llama.Conf.ContextSize = modelConf.Ctx
	return Llama.Restart()
}

func StopLlamaServer() error {
	// Llama not initialized => llama-server already stopped (never started)
	if Llama == nil {
		return nil
	}

	return Llama.Stop()
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

//	if modelConf.Name == "" {
//		return errors.New("model name cannot be empty")
//	}

// IsStartNeeded returns true if we need to start/restart llama-server.
// Check if llama-server is already running with the right model and context size.
func IsStartNeeded(modelConf types.ModelConf) bool {
	if !Llama.IsRunning() {
		return true
	}

	if modelConf.Ctx != Llama.Conf.ContextSize {
		return true
	}

	if modelConf.Name == Llama.Conf.ModelPath {
		return false
	}

	base := filepath.Base(Llama.Conf.ModelPath) // Just the filename without directory
	if modelConf.Name == base {
		return false
	}

	ext := filepath.Ext(base) // Get the extension
	stem := strings.TrimSuffix(base, ext)
	return modelConf.Name != stem
}

// searchModelFile checks if the model file is OK.
func searchModelFile(modelConf types.ModelConf) (string, error) {
	path1 := filepath.Join(ModelsDir, modelConf.Name)

	_, err := os.Stat(path1)
	if err == nil {
		return path1, nil
	}

	if !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot verify if model file exist %s: %w", path1, err)
	}

	path2 := path1 + ".gguf"
	_, err = os.Stat(path2)
	if err == nil {
		return path2, nil
	}

	if os.IsNotExist(err) {
		return "", fmt.Errorf("no model file %s, neither %s", path1, path2)
	}

	return "", fmt.Errorf("error checking model file %s: %w", path1, err)
}
