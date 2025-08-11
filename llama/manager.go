package llama

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/synw/goinfer/conf"
)

// LlamaServerManager - process manager for llama-server.
type LlamaServerManager struct {
	Conf          *conf.LlamaConf
	process       *os.Process
	cmd           *exec.Cmd
	stopChan      chan struct{}
	mu            sync.RWMutex
	startTime     time.Time
	startCount    int
	TokenCallback func(string) bool // TokenCallback sets the prompts that will stop predictions.
}

// NewLlamaServerManager - Creates a new LlamaServerManager.
func NewLlamaServerManager(config *conf.LlamaConf) *LlamaServerManager {
	return &LlamaServerManager{
		Conf:     config,
		stopChan: make(chan struct{}, 1),
	}
}

// Restart - restart with minimal downtime.
func (m *LlamaServerManager) Restart() error {
	err := m.Stop()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create command line
	m.cmd = exec.Command(m.Conf.ExePath, m.Conf.GetCommandArgs()...)

	// Preserve system environment
	// m.cmd.Env = os.Environ()

	fmt.Println("Starting...", m.cmd.String())

	err = m.cmd.Start()
	if err != nil {
		return ErrRestartFailed("failed to restart process: " + err.Error())
	}

	m.startTime = time.Now()
	m.process = m.cmd.Process
	m.startCount++

	return nil
}

// Stop with SIGKILL (9) = faster than Interrupt/SIGINT (graceful shutdown)
func (m *LlamaServerManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// fast return if server is already stopped
	if m.process == nil {
		return nil
	}

	err := m.process.Kill()
	if err != nil {
		if isProcessStillRunning(err) {
			return ErrStopFailed("failed to stop process: " + err.Error())
		}
	}

	// Wait for process to terminate with short timeout
	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	// Terminated successfully
	case <-done:
		m.process = nil
		m.cmd = nil
		return nil
	// Timeout - process might be stuck, but we've sent Kill signal
	case <-time.After(500 * time.Millisecond):
		m.process = nil
		m.cmd = nil
		return nil
	}
}

// HealthCheck - health verification in <2ms.
func (m *LlamaServerManager) HealthCheck() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// check if process exists
	if m.process == nil {
		return false
	}

	//  TCP connection check with short timeout
	start := time.Now()
	conn, err := net.DialTimeout("tcp", m.Conf.GetAddress(), 1*time.Millisecond)
	if err != nil {
		fmt.Printf("Failed HealthCheck DialTimeout: %v", err)
		return false
	}

	err = conn.Close()
	if err != nil {
		fmt.Printf("Failed closing HealthCheck connection: %v", err)
		return false
	}

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

func (m *LlamaServerManager) GetStartTime() time.Time {
	// m.mu.RLock()
	// defer m.mu.RUnlock()
	return m.startTime
}

// GetStartCount - number of start (and restart) times
func (m *LlamaServerManager) GetStartCount() int {
	// m.mu.RLock()
	// defer m.mu.RUnlock()
	return m.startCount
}

func (m *LlamaServerManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.process != nil
}

// monitor goroutine.
func (m *LlamaServerManager) monitor() {
	// Non-blocking monitoring
	for {
		select {
		case <-m.stopChan:
			return
		case <-time.After(5 * time.Second):
			// Periodic health check with minimal logging
			if !m.HealthCheck() && m.process != nil {
				// restart on failure
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

// isProcessStillRunning - check if process is terminated.
func isProcessStillRunning(err error) bool {
	if err == nil {
		return false
	}
	// Check for specific process-related errors
	return err.Error() != "process already finished" &&
		err.Error() != "no such process" &&
		err.Error() != "child process not found"
}
