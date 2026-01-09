package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the complete configuration for edge-logs
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// ClickHouseConfig contains ClickHouse database configuration
type ClickHouseConfig struct {
	// Connection settings
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	TLS      bool   `yaml:"tls"`

	// Connection pool settings (edge optimized)
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`

	// Query settings
	QueryTimeout time.Duration `yaml:"query_timeout"`
	ExecTimeout  time.Duration `yaml:"exec_timeout"`

	// Performance settings
	BlockSize     uint64 `yaml:"block_size"`      // Block size for batch operations
	Compression   bool   `yaml:"compression"`     // Enable compression
	MaxBlockSize  uint64 `yaml:"max_block_size"`  // Max block size
	Async         bool   `yaml:"async"`           // Enable async operations
}

// KubernetesConfig contains Kubernetes client configuration
type KubernetesConfig struct {
	InCluster  bool   `yaml:"in_cluster"`
	ConfigPath string `yaml:"config_path"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	EnableJSON bool   `yaml:"enable_json"`
}

// Load reads configuration from file or uses defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		ClickHouse: ClickHouseConfig{
			Host:     "localhost",
			Port:     9000,
			Database: "edge_logs",
			Username: "default",
			Password: "",
			TLS:      false,
			// Connection pool defaults (edge optimized)
			MaxOpenConns:    20,
			MaxIdleConns:    10,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
			// Query timeouts
			QueryTimeout: 30 * time.Second,
			ExecTimeout:  10 * time.Second,
			// Performance settings
			BlockSize:    1048576, // 1MB
			Compression:  true,
			MaxBlockSize: 1048576, // 1MB
			Async:        false,
		},
		Kubernetes: KubernetesConfig{
			InCluster:  false,
			ConfigPath: "",
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			EnableJSON: false,
		},
	}

	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config/config.yaml"
	}

	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configFile, err)
		}
	}

	// Override with environment variables
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			config.Server.Port = p
		}
	}
	if dbHost := os.Getenv("CLICKHOUSE_HOST"); dbHost != "" {
		config.ClickHouse.Host = dbHost
	}
	if dbUser := os.Getenv("CLICKHOUSE_USERNAME"); dbUser != "" {
		config.ClickHouse.Username = dbUser
	}
	if dbPass := os.Getenv("CLICKHOUSE_PASSWORD"); dbPass != "" {
		config.ClickHouse.Password = dbPass
	}

	return config, nil
}