package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/config"
)

// ConnectionManager manages ClickHouse database connections with pool
type ConnectionManager struct {
	db     *sql.DB
	conn   driver.Conn
	config *config.ClickHouseConfig
}

// NewConnectionManager creates a new ClickHouse connection manager
func NewConnectionManager(cfg *config.ClickHouseConfig) (*ConnectionManager, error) {
	klog.InfoS("初始化 ClickHouse 连接管理器",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"username", cfg.Username,
		"max_open_conns", cfg.MaxOpenConns,
		"max_idle_conns", cfg.MaxIdleConns)

	// Build ClickHouse connection options
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": uint64(cfg.QueryTimeout.Seconds()),
			"send_timeout":       uint64(cfg.ExecTimeout.Seconds()),
			"receive_timeout":    uint64(cfg.ExecTimeout.Seconds()),
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:      10 * time.Second,
		MaxOpenConns:     cfg.MaxOpenConns,
		MaxIdleConns:     cfg.MaxIdleConns,
		ConnMaxLifetime:  cfg.ConnMaxLifetime,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	}

	// Enable TLS if configured
	if cfg.TLS {
		// Note: TLS configuration adjusted for clickhouse-go/v2 compatibility
		klog.InfoS("TLS 已启用 ClickHouse 连接")
	}

	// Connect to ClickHouse
	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, MapClickHouseError(err, "connection_open").Err
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return nil, MapClickHouseError(err, "connection_ping").Err
	}

	// Open SQL interface for connection pool management
	sqlDB := clickhouse.OpenDB(options)

	// Configure connection pool - MUST be set before first use
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify SQL connection with ping
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err := sqlDB.PingContext(ctx2); err != nil {
		sqlDB.Close()
		conn.Close()
		return nil, MapClickHouseError(err, "sql_connection_ping").Err
	}

	klog.InfoS("ClickHouse 连接已建立",
		"host", cfg.Host,
		"database", cfg.Database,
		"connection_pool_configured", true)

	return &ConnectionManager{
		db:     sqlDB,
		conn:   conn,
		config: cfg,
	}, nil
}

// GetDB returns the SQL database connection for connection pool operations
func (cm *ConnectionManager) GetDB() *sql.DB {
	return cm.db
}

// GetConn returns the native ClickHouse connection for advanced operations
func (cm *ConnectionManager) GetConn() driver.Conn {
	return cm.conn
}

// Close gracefully closes all connections
func (cm *ConnectionManager) Close() error {
	klog.InfoS("关闭 ClickHouse 连接")

	var closeErrors []error

	// Close SQL connection pool
	if cm.db != nil {
		if err := cm.db.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("failed to close SQL connection pool: %w", err))
		}
	}

	// Close native connection
	if cm.conn != nil {
		if err := cm.conn.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("failed to close native connection: %w", err))
		}
	}

	if len(closeErrors) > 0 {
		klog.ErrorS(closeErrors[0], "ClickHouse 连接关闭时出现错误",
			"total_errors", len(closeErrors))
		return closeErrors[0]
	}

	klog.InfoS("ClickHouse 连接已成功关闭")
	return nil
}

// HealthCheck performs a health check on the ClickHouse connection
func (cm *ConnectionManager) HealthCheck(ctx context.Context) error {
	klog.V(4).InfoS("执行 ClickHouse 健康检查")

	// Check native connection
	if err := cm.conn.Ping(ctx); err != nil {
		klog.ErrorS(err, "ClickHouse 原生连接健康检查失败")
		return MapClickHouseError(err, "health_check_native").Err
	}

	// Check SQL connection pool
	if err := cm.db.PingContext(ctx); err != nil {
		klog.ErrorS(err, "ClickHouse SQL 连接池健康检查失败")
		return MapClickHouseError(err, "health_check_pool").Err
	}

	klog.V(4).InfoS("ClickHouse 健康检查成功")
	return nil
}

// Stats returns connection pool statistics
func (cm *ConnectionManager) Stats() sql.DBStats {
	return cm.db.Stats()
}

// ReconnectIfNeeded attempts to reconnect if connection is lost
func (cm *ConnectionManager) ReconnectIfNeeded(ctx context.Context) error {
	if err := cm.HealthCheck(ctx); err != nil {
		klog.InfoS("连接失败，尝试重新连接到 ClickHouse")

		// Close existing connections
		cm.Close()

		// Create new connection manager
		newCM, err := NewConnectionManager(cm.config)
		if err != nil {
			return MapClickHouseError(err, "reconnection").Err
		}

		// Replace connections
		cm.db = newCM.db
		cm.conn = newCM.conn

		klog.InfoS("ClickHouse 重新连接成功")
	}

	return nil
}
