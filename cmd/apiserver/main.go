package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/outpostos/edge-logs/pkg/apiserver"
	"github.com/outpostos/edge-logs/pkg/config"
)

// Build-time variables (set via ldflags)
var (
	Version   = "v0.1.0-dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "edge-logs-apiserver",
		Short: "Edge Logs API Server",
		Long:  `Edge Logs API Server provides log aggregation and query capabilities`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := run(); err != nil {
				klog.ErrorS(err, "Failed to start edge-logs-apiserver")
				os.Exit(1)
			}
		},
	}

	// Initialize klog flags
	klog.InitFlags(flag.CommandLine)
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	if err := rootCmd.Execute(); err != nil {
		klog.ErrorS(err, "Failed to execute command")
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create and start API server
	server, err := apiserver.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create API server: %w", err)
	}

	klog.InfoS("Starting edge-logs API server", "version", Version, "build_date", BuildDate, "git_commit", GitCommit, "port", cfg.Server.Port)

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	return nil
}