package apiserver

import (
	"testing"

	"github.com/outpostos/edge-logs/pkg/config"
)

func TestNewServer(t *testing.T) {
	// Test creating a new server with default config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			Host:         "localhost",
		},
		ClickHouse: config.ClickHouseConfig{
			Host:     "localhost",
			Port:     9000,
			Database: "edge_logs",
		},
		Kubernetes: config.KubernetesConfig{
			InCluster: false,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("Expected no error creating server, got %v", err)
	}

	if server == nil {
		t.Fatal("Expected server instance, got nil")
	}

	if server.config != cfg {
		t.Error("Expected server config to match provided config")
	}

	if server.container == nil {
		t.Error("Expected container to be initialized")
	}

	if server.httpServer == nil {
		t.Error("Expected HTTP server to be initialized")
	}

	// Verify server address
	expectedAddr := "localhost:8080"
	if server.httpServer.Addr != expectedAddr {
		t.Errorf("Expected server address %s, got %s", expectedAddr, server.httpServer.Addr)
	}
}

func TestServerConfiguration(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 9999,
			Host: "0.0.0.0",
		},
		ClickHouse: config.ClickHouseConfig{
			Host:     "remote-db",
			Port:     9000,
			Database: "test_logs",
		},
		Kubernetes: config.KubernetesConfig{
			InCluster: false,
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedAddr := "0.0.0.0:9999"
	if server.httpServer.Addr != expectedAddr {
		t.Errorf("Expected address %s, got %s", expectedAddr, server.httpServer.Addr)
	}
}