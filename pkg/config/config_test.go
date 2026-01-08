package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Clear any existing environment variables
	clearEnvVars()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Verify default values
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.ClickHouse.Host != "localhost" {
		t.Errorf("Expected default ClickHouse host 'localhost', got '%s'", cfg.ClickHouse.Host)
	}
	if cfg.ClickHouse.Database != "edge_logs" {
		t.Errorf("Expected default database 'edge_logs', got '%s'", cfg.ClickHouse.Database)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", cfg.Logging.Level)
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Clear any existing environment variables
	clearEnvVars()

	// Set environment variables
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("CLICKHOUSE_HOST", "remote-db")
	os.Setenv("CLICKHOUSE_USERNAME", "testuser")
	os.Setenv("CLICKHOUSE_PASSWORD", "testpass")

	defer clearEnvVars()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config with environment variables: %v", err)
	}

	// Verify environment overrides
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.ClickHouse.Host != "remote-db" {
		t.Errorf("Expected ClickHouse host 'remote-db', got '%s'", cfg.ClickHouse.Host)
	}
	if cfg.ClickHouse.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", cfg.ClickHouse.Username)
	}
	if cfg.ClickHouse.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", cfg.ClickHouse.Password)
	}
}

func TestLoad_InvalidPortEnvironment(t *testing.T) {
	// Clear any existing environment variables
	clearEnvVars()

	// Set invalid port
	os.Setenv("SERVER_PORT", "invalid")
	defer clearEnvVars()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config with invalid port: %v", err)
	}

	// Should fall back to default port
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected fallback to default port 8080, got %d", cfg.Server.Port)
	}
}

func TestConfig_Timeouts(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	expectedTimeout := 30 * time.Second
	if cfg.Server.ReadTimeout != expectedTimeout {
		t.Errorf("Expected read timeout %v, got %v", expectedTimeout, cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != expectedTimeout {
		t.Errorf("Expected write timeout %v, got %v", expectedTimeout, cfg.Server.WriteTimeout)
	}
}

func clearEnvVars() {
	envVars := []string{
		"CONFIG_FILE",
		"SERVER_HOST",
		"SERVER_PORT",
		"CLICKHOUSE_HOST",
		"CLICKHOUSE_USERNAME",
		"CLICKHOUSE_PASSWORD",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}