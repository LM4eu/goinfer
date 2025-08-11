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
	Llama   *llama.Runner
	Monitor *llama.Monitor
)

var (
	ErrModelNotFound = errors.New("model not found")
	ErrInvalidInput  = errors.New("invalid input")
)

func RestartLlamaServer(modelConf types.ModelParams) error {
	if Llama == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	isURL := conf.IsDownloadURL(modelConf.Name)
	isLocalFile := (isURL == 0)

	modelPath := modelConf.Name
	if isLocalFile {
		path, err := searchModelFile(modelConf)
		if err != nil {
			return err
		}
		modelPath = path
	} else if IsDebug {
		fmt.Println("Will download " + modelConf.Name)
	}

	Llama.Conf.ModelPathname = modelPath
	Llama.Conf.PathnameType = isURL
	Llama.Conf.ContextSize = modelConf.Ctx
	err := Llama.Restart()
	if err != nil {
		return err
	}

	Monitor.Start()

	return nil
}

func StopLlamaServer() error {
	// Llama not initialized => llama-server already stopped (never started)
	if Llama == nil {
		return nil
	}

	return Llama.Stop()
}

// GetServerStatus - Gets the current server status.
func GetServerStatus() (int, string, []string, bool, time.Duration) {
	if Llama == nil {
		return 0, "", nil, false, 0
	}

	count := Llama.StartCount()
	exe := Llama.Conf.ExePath
	args := Llama.Conf.GetCommandArgs()
	running := Llama.IsRunning()
	uptime := Llama.Uptime()

	return count, exe, args, running, uptime
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
func IsStartNeeded(modelConf types.ModelParams) bool {
	if !Llama.IsRunning() {
		return true
	}

	if modelConf.Ctx != Llama.Conf.ContextSize {
		return true
	}

	if modelConf.Name == Llama.Conf.ModelPathname {
		return false
	}

	base := filepath.Base(Llama.Conf.ModelPathname) // Just the filename without directory
	if modelConf.Name == base {
		return false
	}

	ext := filepath.Ext(base) // Get the extension
	stem := strings.TrimSuffix(base, ext)
	return modelConf.Name != stem
}

// searchModelFile checks if the model file is OK.
func searchModelFile(modelConf types.ModelParams) (string, error) {
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
