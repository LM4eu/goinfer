package state

import (
	"time"

	"github.com/synw/goinfer/llama"
	"github.com/synw/goinfer/types"
)

// models state.
var (
	ModelsDir     = ""
	IsModelLoaded = false
	LoadedModel   = ""
	ModelOptions  = DefaultModelOptions
)

// Inference state.
var (
	ContinueInferringController = true
	IsInferring                 = false
)

// app state.
var (
	IsVerbose = true
	IsDebug   = false
)

// tasks.
var TasksDir = "./tasks"

// OpenAi api.
var OpenAiConf types.OpenAiConf

// the language model instance.
var Lm llama.LLama

// Llama server manager state.
var (
	LlamaManager       *llama.LlamaServerManager
	LlamaConfig        *types.LlamaConfig
	LlamaMonitor       *llama.Monitor
	IsServerRunning    = false
	ServerStartTime    = time.Time{}
	ServerRestartCount = 0
)

// InitializeLlama - Initializes the Llama server manager.
func InitializeLlama(config *types.LlamaConfig) error {
	if LlamaManager != nil {
		return llama.ErrAlreadyRunning("Llama manager already initialized")
	}

	// Convert types.LlamaConfig to llama.LlamaConfig
	llamaConfig := llama.NewLlamaConfig(
		config.BinaryPath,
		config.ModelPath,
		config.Args...,
	)
	llamaConfig.Host = config.Host
	llamaConfig.Port = config.Port

	// Create manager
	LlamaManager = llama.NewLlamaServerManager(llamaConfig)
	LlamaConfig = config

	// Create monitor
	LlamaMonitor = llama.NewMonitor(llamaConfig)
	LlamaMonitor.Start()

	return nil
}

// StartLlamaServer - Starts the Llama server.
func StartLlamaServer() error {
	if LlamaManager == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	err := LlamaManager.Start()
	if err != nil {
		return err
	}

	IsServerRunning = true
	ServerStartTime = time.Now()

	return nil
}

// StopLlamaServer - Stops the Llama server.
func StopLlamaServer() error {
	if LlamaManager == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	err := LlamaManager.Stop()
	if err != nil {
		return err
	}

	IsServerRunning = false
	ServerRestartCount++

	return nil
}

// RestartLlamaServer - Restarts the Llama server.
func RestartLlamaServer() error {
	if LlamaManager == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	err := LlamaManager.Restart()
	if err != nil {
		return err
	}

	IsServerRunning = true
	ServerStartTime = time.Now()
	ServerRestartCount++

	return nil
}

// GetServerStatus - Gets the current server status.
func GetServerStatus() (bool, time.Duration, int) {
	if LlamaManager == nil {
		return false, 0, 0
	}

	return IsServerRunning, LlamaManager.GetUptime(), LlamaManager.GetRestartCount()
}

// CheckServerHealth - Performs a health check on the server.
func CheckServerHealth() bool {
	if LlamaManager == nil {
		return false
	}

	return LlamaManager.HealthCheck()
}

// UpdateServerConfig - Updates the server configuration.
func UpdateServerConfig(newConfig *types.LlamaConfig) error {
	if LlamaManager == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	return LlamaManager.UpdateConfig(llama.NewLlamaConfig(
		newConfig.BinaryPath,
		newConfig.ModelPath,
		newConfig.Args...,
	))
}

// GetServerManager - Returns the server manager instance.
func GetServerManager() *llama.LlamaServerManager {
	return LlamaManager
}

// GetServerMonitor - Returns the server monitor instance.
func GetServerMonitor() *llama.Monitor {
	return LlamaMonitor
}
