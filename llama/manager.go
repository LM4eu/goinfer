package llama

import (
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

// LlamaServerManager - Performance-optimized process manager for llama-server.
type LlamaServerManager struct {
	config       *LlamaConfig
	process      *os.Process
	cmd          *exec.Cmd
	stopChan     chan struct{}
	mu           sync.RWMutex
	startTime    time.Time
	restartCount int
}

// NewLlamaServerManager - Creates a new LlamaServerManager with minimal overhead.
func NewLlamaServerManager(config *LlamaConfig) *LlamaServerManager {
	return &LlamaServerManager{
		config:   config,
		stopChan: make(chan struct{}, 1),
	}
}

// Start - process launch with minimal validation.
func (m *LlamaServerManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	//  validation - only essential checks
	err := m.config.Validate()
	if err != nil {
		return err
	}

	// Check if already running
	if m.process != nil {
		return ErrAlreadyRunning("server is already running")
	}

	// Create command with minimal overhead
	m.cmd = exec.Command(m.config.BinaryPath, m.config.GetCommandArgs()...)

	// Preserve system environment for performance
	m.cmd.Env = os.Environ()

	// Start process with minimal setup
	err = m.cmd.Start()
	if err != nil {
		return ErrStartFailed("failed to start process: " + err.Error())
	}

	m.process = m.cmd.Process
	m.startTime = time.Now()

	// Start monitoring goroutine with minimal overhead
	go m.monitor()

	return nil
}

// Stop - Quick termination with Kill() instead of graceful shutdown.
func (m *LlamaServerManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.process == nil {
		return ErrNotRunning("server is not running")
	}

	//  termination with Kill() for performance
	err := m.process.Kill()
	if err != nil {
		// Process might already be terminated
		if !isProcessTerminated(err) {
			return ErrStopFailed("failed to stop process: " + err.Error())
		}
	}

	// Wait for process to terminate with short timeout
	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case <-done:
		// Process terminated successfully
		m.process = nil
		m.cmd = nil
		return nil
	case <-time.After(100 * time.Millisecond):
		// Timeout - process might be stuck, but we've sent Kill signal
		m.process = nil
		m.cmd = nil
		return nil
	}
}

// Restart - restart with minimal downtime.
func (m *LlamaServerManager) Restart() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	//  stop
	if m.process != nil {
		err := m.process.Kill()
		if err != nil && !isProcessTerminated(err) {
			return ErrRestartFailed("failed to stop process: " + err.Error())
		}
	}

	// Quick restart
	m.restartCount++

	// Create new command with minimal overhead
	m.cmd = exec.Command(m.config.BinaryPath, m.config.GetCommandArgs()...)
	m.cmd.Env = os.Environ()

	err := m.cmd.Start()
	if err != nil {
		return ErrRestartFailed("failed to restart process: " + err.Error())
	}

	m.process = m.cmd.Process
	m.startTime = time.Now()

	return nil
}

// HealthCheck - Quick health verification in <2ms.
func (m *LlamaServerManager) HealthCheck() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Quick check if process exists
	if m.process == nil {
		return false
	}

	//  TCP connection check with short timeout
	start := time.Now()
	conn, err := net.DialTimeout("tcp", m.config.GetAddress(), 1*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()

	// Verify health check time target
	healthCheckTime := time.Since(start)
	return healthCheckTime < 2*time.Millisecond
}

// GetUptime - uptime calculation.
func (m *LlamaServerManager) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.process == nil {
		return 0
	}
	return time.Since(m.startTime)
}

// GetPID - PID retrieval.
func (m *LlamaServerManager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.process == nil {
		return -1
	}
	return m.process.Pid
}

// GetStartTime - start time retrieval.
func (m *LlamaServerManager) GetStartTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.startTime
}

// GetRestartCount - restart count retrieval.
func (m *LlamaServerManager) GetRestartCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.restartCount
}

// IsRunning - running state check.
func (m *LlamaServerManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.process != nil
}

// monitor - Lightweight monitoring goroutine.
func (m *LlamaServerManager) monitor() {
	// Non-blocking monitoring with minimal overhead
	for {
		select {
		case <-m.stopChan:
			return
		case <-time.After(5 * time.Second):
			// Periodic health check with minimal logging
			if !m.HealthCheck() && m.process != nil {
				// Quick restart on failure
				go func() {
					err := m.Restart()
					if err != nil {
						// Minimal error logging
						_ = err
					}
				}()
			}
		}
	}
}

// UpdateConfig - configuration update.
func (m *LlamaServerManager) UpdateConfig(newConfig *LlamaConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate new config
	err := newConfig.Validate()
	if err != nil {
		return err
	}

	// Update config and restart if needed
	if m.process != nil {
		m.config = newConfig.Clone()
		return m.Restart()
	}

	m.config = newConfig.Clone()
	return nil
}

// GetConfig - config retrieval.
func (m *LlamaServerManager) GetConfig() *LlamaConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.Clone()
}

// Close - Cleanup resources.
func (m *LlamaServerManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop monitoring
	close(m.stopChan)

	// Stop process if running
	if m.process != nil {
		return m.Stop()
	}

	return nil
}

// Error types for performance-critical error handling.
type ErrAlreadyRunning string

func (e ErrAlreadyRunning) Error() string { return "server already running: " + string(e) }

type ErrNotRunning string

func (e ErrNotRunning) Error() string { return "server not running: " + string(e) }

type ErrStartFailed string

func (e ErrStartFailed) Error() string { return "start failed: " + string(e) }

type ErrStopFailed string

func (e ErrStopFailed) Error() string { return "stop failed: " + string(e) }

type ErrRestartFailed string

func (e ErrRestartFailed) Error() string { return "restart failed: " + string(e) }

// isProcessTerminated - check if process is terminated.
func isProcessTerminated(err error) bool {
	if err == nil {
		return true
	}
	// Check for specific process-related errors
	return err.Error() == "process already finished" ||
		err.Error() == "no such process" ||
		err.Error() == "child process not found"
}
