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
	ModelConf     = types.DefaultModelConf
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

// the language model instance.
var (
	Llama   *llama.LlamaServerManager
	Monitor *llama.Monitor
)

// StartLlamaServer - Starts the Llama server.
func StartLlamaServer() error {
	if Llama == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	err := Llama.Start()
	if err != nil {
		return err
	}

	return nil
}

// StopLlamaServer - Stops the Llama server.
func StopLlamaServer() error {
	if Llama == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	err := Llama.Stop()
	if err != nil {
		return err
	}

	return nil
}

// RestartLlamaServer - Restarts the Llama server.
func RestartLlamaServer() error {
	if Llama == nil {
		return llama.ErrNotRunning("Llama manager not initialized")
	}

	err := Llama.Restart()
	if err != nil {
		return err
	}

	return nil
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
