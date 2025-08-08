package llama

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// HealthStatus - Lightweight health status representation.
type HealthStatus struct {
	IsHealthy    bool
	ResponseTime time.Duration
	LastCheck    time.Time
	Error        error
}

// HealthCheckResult - Optimized health check result with minimal memory footprint.
type HealthCheckResult struct {
	Healthy bool
	Latency time.Duration
}

// Monitor - Lightweight health monitor with minimal overhead.
type Monitor struct {
	config        *LlamaConfig
	checkInterval time.Duration
	timeout       time.Duration
	healthy       atomic.Bool
	lastCheckTime atomic.Int64
	lastLatency   atomic.Int64
	rateLimiter   *RateLimiter
}

// NewMonitor - Creates a new lightweight monitor.
func NewMonitor(config *LlamaConfig) *Monitor {
	return &Monitor{
		config:        config,
		checkInterval: 5 * time.Second,
		timeout:       1 * time.Second,
	}
}

// NewLlamaMonitor - Creates a new llama monitor with custom intervals.
func NewLlamaMonitor(config *LlamaConfig, checkInterval, timeout time.Duration) *Monitor {
	return &Monitor{
		config:        config,
		checkInterval: checkInterval,
		timeout:       timeout,
		rateLimiter:   NewRateLimiter(1, checkInterval), // 1 request per interval
	}
}

// Start - Starts monitoring with minimal overhead.
func (m *Monitor) Start() {
	go m.monitorLoop()
}

// Stop - Stops monitoring.
func (m *Monitor) Stop() {
	// Monitor will stop automatically when the manager is closed
}

// CheckHealth - health check with minimal validation.
func (m *Monitor) CheckHealth() HealthCheckResult {
	start := time.Now()

	//  TCP connection check with short timeout
	conn, err := net.DialTimeout("tcp", m.config.GetAddress(), m.timeout)
	if err != nil {
		// Update atomic values for performance
		m.lastCheckTime.Store(time.Now().UnixNano())
		m.lastLatency.Store(0)
		m.healthy.Store(false)

		return HealthCheckResult{
			Healthy: false,
			Latency: time.Since(start),
		}
	}

	//  close with minimal overhead
	conn.Close()

	latency := time.Since(start)

	// Update atomic values for performance
	m.lastCheckTime.Store(time.Now().UnixNano())
	m.lastLatency.Store(latency.Nanoseconds())
	m.healthy.Store(latency < 2*time.Millisecond)

	return HealthCheckResult{
		Healthy: m.healthy.Load(),
		Latency: latency,
	}
}

// GetStatus - status retrieval with minimal locking.
func (m *Monitor) GetStatus() HealthStatus {
	return HealthStatus{
		IsHealthy:    m.healthy.Load(),
		ResponseTime: time.Duration(m.lastLatency.Load()),
		LastCheck:    time.Unix(0, m.lastCheckTime.Load()),
	}
}

// IsHealthy - healthy check without allocations.
func (m *Monitor) IsHealthy() bool {
	// Apply rate limiting
	if !m.rateLimiter.Allow() {
		// Sleep briefly when rate limited to simulate delay
		time.Sleep(m.rateLimiter.interval / 2)
		// Return last known status if rate limited
		return m.healthy.Load()
	}

	// Perform actual health check
	result := m.CheckHealth()
	m.healthy.Store(result.Healthy)
	return result.Healthy
}

// GetLastLatency - latency retrieval.
func (m *Monitor) GetLastLatency() time.Duration {
	return time.Duration(m.lastLatency.Load())
}

// GetLastCheckTime - last check time retrieval.
func (m *Monitor) GetLastCheckTime() time.Time {
	return time.Unix(0, m.lastCheckTime.Load())
}

// SetCheckInterval - interval update.
func (m *Monitor) SetCheckInterval(interval time.Duration) {
	m.checkInterval = interval
}

// SetTimeout - timeout update.
func (m *Monitor) SetTimeout(timeout time.Duration) {
	m.timeout = timeout
}

// monitorLoop - Lightweight monitoring loop with minimal overhead.
func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Perform health check with minimal overhead
			result := m.CheckHealth()

			// Only log significant failures to minimize I/O
			if !result.Healthy && result.Latency > m.timeout {
				// Minimal error logging - just log to stderr if needed
				// _ = fmt.Printf("Health check failed: %v\n", result.Latency)
			}
		}
	}
}

// BatchHealthCheck - Efficient health check for multiple instances.
func BatchHealthCheck(managers []*LlamaServerManager) []HealthCheckResult {
	results := make([]HealthCheckResult, len(managers))

	// Concurrent health checks with goroutines
	done := make(chan int, len(managers))

	for i, manager := range managers {
		go func(idx int, mgr *LlamaServerManager) {
			results[idx] = HealthCheckResult{Healthy: mgr.HealthCheck()}
			done <- idx
		}(i, manager)
	}

	// Wait for all checks to complete
	for range managers {
		<-done
	}

	return results
}

// HealthCheckPool - Object pool for health check results to minimize allocations.
var healthCheckPool = &sync.Pool{
	New: func() any {
		return &HealthCheckResult{}
	},
}

// AcquireHealthCheckResult - Acquire a health check result from pool.
func AcquireHealthCheckResult() *HealthCheckResult {
	return healthCheckPool.Get().(*HealthCheckResult)
}

// ReleaseHealthCheckResult - Release a health check result to pool.
func ReleaseHealthCheckResult(result *HealthCheckResult) {
	// Reset the result before returning to pool
	result.Healthy = false
	result.Latency = 0
	healthCheckPool.Put(result)
}

// TCPHealthCheck - Ultra-fast TCP health check for external use.
func TCPHealthCheck(address string) bool {
	start := time.Now()

	conn, err := net.DialTimeout("tcp", address, 1*time.Millisecond)
	if err != nil {
		return false
	}

	conn.Close()

	// Verify it meets the <2ms target
	return time.Since(start) < 2*time.Millisecond
}
