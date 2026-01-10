package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/metrics"
)

// ConnectionPoolManager provides enhanced connection pool management with monitoring
type ConnectionPoolManager struct {
	db                   *sql.DB
	metrics              *metrics.QueryPerformanceMetrics
	config               *PoolConfig
	healthCheckInterval  time.Duration
	lastHealthCheck      time.Time
	connectionHealth     bool
	mu                   sync.RWMutex
	cancelHealthCheck    context.CancelFunc
	healthCheckDone      chan struct{}
}

// PoolConfig contains connection pool configuration
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	HealthCheckFreq time.Duration
	Dataset         string
}

// NewConnectionPoolManager creates an enhanced connection pool manager
func NewConnectionPoolManager(db *sql.DB, config *PoolConfig, metrics *metrics.QueryPerformanceMetrics) *ConnectionPoolManager {
	cpm := &ConnectionPoolManager{
		db:                  db,
		metrics:             metrics,
		config:              config,
		healthCheckInterval: config.HealthCheckFreq,
		connectionHealth:    true,
		healthCheckDone:     make(chan struct{}),
	}

	// Configure connection pool
	cpm.configureConnectionPool()

	// Start health check routine
	cpm.startHealthCheck()

	klog.InfoS("增强型连接池管理器已初始化",
		"dataset", config.Dataset,
		"max_open_conns", config.MaxOpenConns,
		"max_idle_conns", config.MaxIdleConns,
		"health_check_interval", config.HealthCheckFreq.Seconds())

	return cpm
}

// configureConnectionPool applies connection pool configuration
func (cpm *ConnectionPoolManager) configureConnectionPool() {
	cpm.db.SetMaxOpenConns(cpm.config.MaxOpenConns)
	cpm.db.SetMaxIdleConns(cpm.config.MaxIdleConns)
	cpm.db.SetConnMaxLifetime(cpm.config.ConnMaxLifetime)
	cpm.db.SetConnMaxIdleTime(cpm.config.ConnMaxIdleTime)

	klog.InfoS("连接池配置已应用",
		"dataset", cpm.config.Dataset,
		"max_open_conns", cpm.config.MaxOpenConns,
		"max_idle_conns", cpm.config.MaxIdleConns,
		"conn_max_lifetime", cpm.config.ConnMaxLifetime.Seconds(),
		"conn_max_idle_time", cpm.config.ConnMaxIdleTime.Seconds())
}

// startHealthCheck starts the background health check routine
func (cpm *ConnectionPoolManager) startHealthCheck() {
	ctx, cancel := context.WithCancel(context.Background())
	cpm.cancelHealthCheck = cancel

	go func() {
		ticker := time.NewTicker(cpm.healthCheckInterval)
		defer ticker.Stop()
		defer close(cpm.healthCheckDone)

		for {
			select {
			case <-ticker.C:
				cpm.performHealthCheck()
			case <-ctx.Done():
				klog.InfoS("连接池健康检查已停止",
					"dataset", cpm.config.Dataset)
				return
			}
		}
	}()

	klog.InfoS("连接池健康检查已启动",
		"dataset", cpm.config.Dataset,
		"interval_seconds", cpm.healthCheckInterval.Seconds())
}

// performHealthCheck performs a connection health check
func (cpm *ConnectionPoolManager) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check database connection
	err := cpm.db.PingContext(ctx)

	cpm.mu.Lock()
	previousHealth := cpm.connectionHealth
	cpm.connectionHealth = (err == nil)
	cpm.lastHealthCheck = time.Now()
	cpm.mu.Unlock()

	// Log health status changes
	if previousHealth != cpm.connectionHealth {
		if cpm.connectionHealth {
			klog.InfoS("连接池健康状态已恢复",
				"dataset", cpm.config.Dataset)
		} else {
			klog.ErrorS(err, "连接池健康检查失败",
				"dataset", cpm.config.Dataset)
		}
	}

	// Update connection pool statistics
	stats := cpm.db.Stats()
	if cpm.metrics != nil {
		cpm.metrics.UpdateConnectionPoolStats(
			cpm.config.Dataset,
			stats.OpenConnections,
			stats.Idle,
			stats.InUse,
		)
	}

	// Log connection pool statistics
	klog.V(4).InfoS("连接池统计",
		"dataset", cpm.config.Dataset,
		"open_connections", stats.OpenConnections,
		"idle_connections", stats.Idle,
		"in_use_connections", stats.InUse,
		"wait_count", stats.WaitCount,
		"wait_duration", stats.WaitDuration.Seconds(),
		"max_idle_closed", stats.MaxIdleClosed,
		"max_lifetime_closed", stats.MaxLifetimeClosed)
}

// GetDB returns the database connection
func (cpm *ConnectionPoolManager) GetDB() *sql.DB {
	return cpm.db
}

// IsHealthy returns the current health status
func (cpm *ConnectionPoolManager) IsHealthy() bool {
	cpm.mu.RLock()
	defer cpm.mu.RUnlock()
	return cpm.connectionHealth
}

// GetStats returns connection pool statistics
func (cpm *ConnectionPoolManager) GetStats() sql.DBStats {
	return cpm.db.Stats()
}

// GetPerformanceMetrics returns detailed performance metrics
func (cpm *ConnectionPoolManager) GetPerformanceMetrics() *PoolPerformanceMetrics {
	stats := cpm.db.Stats()

	cpm.mu.RLock()
	health := cpm.connectionHealth
	lastCheck := cpm.lastHealthCheck
	cpm.mu.RUnlock()

	return &PoolPerformanceMetrics{
		Dataset:              cpm.config.Dataset,
		OpenConnections:      stats.OpenConnections,
		IdleConnections:      stats.Idle,
		InUseConnections:     stats.InUse,
		WaitCount:            stats.WaitCount,
		WaitDuration:         stats.WaitDuration,
		MaxIdleClosed:        stats.MaxIdleClosed,
		MaxLifetimeClosed:    stats.MaxLifetimeClosed,
		IsHealthy:            health,
		LastHealthCheck:      lastCheck,
		HealthCheckInterval:  cpm.healthCheckInterval,
		MaxOpenConns:         cpm.config.MaxOpenConns,
		MaxIdleConns:         cpm.config.MaxIdleConns,
		Timestamp:            time.Now(),
	}
}

// PoolPerformanceMetrics contains detailed connection pool performance data
type PoolPerformanceMetrics struct {
	Dataset             string    `json:"dataset"`
	OpenConnections     int       `json:"open_connections"`
	IdleConnections     int       `json:"idle_connections"`
	InUseConnections    int       `json:"in_use_connections"`
	WaitCount           int64     `json:"wait_count"`
	WaitDuration        time.Duration `json:"wait_duration_ms"`
	MaxIdleClosed       int64     `json:"max_idle_closed"`
	MaxLifetimeClosed   int64     `json:"max_lifetime_closed"`
	IsHealthy           bool      `json:"is_healthy"`
	LastHealthCheck     time.Time `json:"last_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval_ms"`
	MaxOpenConns        int       `json:"max_open_conns"`
	MaxIdleConns        int       `json:"max_idle_conns"`
	Timestamp           time.Time `json:"timestamp"`
}

// Close gracefully closes the connection pool manager
func (cpm *ConnectionPoolManager) Close() error {
	klog.InfoS("关闭增强型连接池管理器",
		"dataset", cpm.config.Dataset)

	// Stop health check routine
	if cpm.cancelHealthCheck != nil {
		cpm.cancelHealthCheck()
	}

	// Wait for health check to finish
	select {
	case <-cpm.healthCheckDone:
		klog.InfoS("连接池健康检查已停止",
			"dataset", cpm.config.Dataset)
	case <-time.After(5 * time.Second):
		klog.InfoS("连接池健康检查停止超时",
			"dataset", cpm.config.Dataset)
	}

	// Close database connection
	if cpm.db != nil {
		if err := cpm.db.Close(); err != nil {
			klog.ErrorS(err, "关闭数据库连接失败",
				"dataset", cpm.config.Dataset)
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}

	klog.InfoS("增强型连接池管理器已成功关闭",
		"dataset", cpm.config.Dataset)

	return nil
}

// Warmup warms up the connection pool by establishing connections
func (cpm *ConnectionPoolManager) Warmup(ctx context.Context, connectionCount int) error {
	klog.InfoS("连接池预热开始",
		"dataset", cpm.config.Dataset,
		"target_connections", connectionCount)

	connections := make([]*sql.Conn, 0, connectionCount)
	errors := make([]error, 0)

	for i := 0; i < connectionCount; i++ {
		conn, err := cpm.db.Conn(ctx)
		if err != nil {
			errors = append(errors, err)
			klog.ErrorS(err, "连接池预热连接失败",
				"dataset", cpm.config.Dataset,
				"connection_number", i+1)
			continue
		}
		connections = append(connections, conn)
	}

	// Close the temporary connections
	for _, conn := range connections {
		if err := conn.Close(); err != nil {
			klog.ErrorS(err, "关闭预热连接失败",
				"dataset", cpm.config.Dataset)
		}
	}

	if len(errors) > 0 {
		klog.InfoS("连接池预热部分失败",
			"dataset", cpm.config.Dataset,
			"success_count", len(connections),
			"error_count", len(errors))
		return fmt.Errorf("warmup completed with %d errors", len(errors))
	}

	klog.InfoS("连接池预热成功",
		"dataset", cpm.config.Dataset,
		"connections_established", len(connections))

	return nil
}

// UpdateConfig dynamically updates connection pool configuration
func (cpm *ConnectionPoolManager) UpdateConfig(newConfig *PoolConfig) {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	oldConfig := cpm.config
	cpm.config = newConfig

	// Apply new configuration
	cpm.configureConnectionPool()

	// Update health check interval if changed
	if newConfig.HealthCheckFreq != cpm.healthCheckInterval {
		cpm.healthCheckInterval = newConfig.HealthCheckFreq
		// Restart health check with new interval
		if cpm.cancelHealthCheck != nil {
			cpm.cancelHealthCheck()
		}
		cpm.startHealthCheck()
	}

	klog.InfoS("连接池配置已更新",
		"dataset", cpm.config.Dataset,
		"old_max_open_conns", oldConfig.MaxOpenConns,
		"new_max_open_conns", newConfig.MaxOpenConns,
		"old_max_idle_conns", oldConfig.MaxIdleConns,
		"new_max_idle_conns", newConfig.MaxIdleConns)
}

// GetOptimalPoolSize calculates optimal pool size based on current load
func (cpm *ConnectionPoolManager) GetOptimalPoolSize() (minConns, maxConns int) {
	stats := cpm.db.Stats()

	// Calculate optimal sizes based on usage patterns
	inUse := stats.InUse
	waitCount := stats.WaitCount

	// If we're seeing waits, we need more connections
	if waitCount > 0 {
		minConns = inUse + 2
		maxConns = min(cpm.config.MaxOpenConns*2, 100) // Cap at 100
	} else {
		// Reduce connections if we're not using them
		minConns = max(inUse, 2)
		maxConns = max(minConns+2, cpm.config.MaxOpenConns/2)
	}

	// Ensure we don't go below minimum requirements
	minConns = max(minConns, 2)
	maxConns = max(maxConns, 5)

	return minConns, maxConns
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}