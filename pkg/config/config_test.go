package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaultConfig(t *testing.T) {
	// Test loading with default values when no config file exists
	os.Unsetenv("CONFIG_FILE")

	config, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify default values
	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.Port)
	}

	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host 0.0.0.0, got %s", config.Server.Host)
	}

	if config.ClickHouse.Host != "localhost" {
		t.Errorf("Expected ClickHouse host localhost, got %s", config.ClickHouse.Host)
	}

	if config.ClickHouse.Database != "edge_logs" {
		t.Errorf("Expected database edge_logs, got %s", config.ClickHouse.Database)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("CLICKHOUSE_HOST", "remote-clickhouse")
	os.Setenv("CLICKHOUSE_USERNAME", "testuser")
	os.Setenv("CLICKHOUSE_PASSWORD", "testpass")

	defer func() {
		// Cleanup
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("CLICKHOUSE_HOST")
		os.Unsetenv("CLICKHOUSE_USERNAME")
		os.Unsetenv("CLICKHOUSE_PASSWORD")
	}()

	config, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify environment overrides
	if config.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", config.Server.Host)
	}

	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}

	if config.ClickHouse.Host != "remote-clickhouse" {
		t.Errorf("Expected ClickHouse host remote-clickhouse, got %s", config.ClickHouse.Host)
	}

	if config.ClickHouse.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", config.ClickHouse.Username)
	}

	if config.ClickHouse.Password != "testpass" {
		t.Errorf("Expected password testpass, got %s", config.ClickHouse.Password)
	}
}

func TestConfigDefaults(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test timeout defaults
	expectedReadTimeout := 30 * time.Second
	if config.Server.ReadTimeout != expectedReadTimeout {
		t.Errorf("Expected read timeout %v, got %v", expectedReadTimeout, config.Server.ReadTimeout)
	}

	expectedWriteTimeout := 30 * time.Second
	if config.Server.WriteTimeout != expectedWriteTimeout {
		t.Errorf("Expected write timeout %v, got %v", expectedWriteTimeout, config.Server.WriteTimeout)
	}

	// Test ClickHouse defaults
	if config.ClickHouse.Port != 9000 {
		t.Errorf("Expected ClickHouse port 9000, got %d", config.ClickHouse.Port)
	}

	if config.ClickHouse.TLS != false {
		t.Errorf("Expected ClickHouse TLS false, got %t", config.ClickHouse.TLS)
	}
}