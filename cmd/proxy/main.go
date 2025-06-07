package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	proxy "github.com/paulgrammer/mcp-proxy"
)

// A version string that can be set with
//
//	-ldflags "-X main.Build=SOMEVERSION"
//
// at compile-time.
var Build string

func main() {
	// Define command-line flags
	configPath := flag.String("config", "./config.yml", "Path to the configuration file")
	version := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// Handle version flag
	if *version {
		if Build != "" {
			fmt.Printf("mcp-proxy version %s\n", Build)
		} else {
			fmt.Println("mcp-proxy version unknown (development build)")
		}
		os.Exit(0)
	}

	// Set up structured logging first
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Parse the configuration
	cfg, err := proxy.ParseConfig(*configPath)
	if err != nil {
		logger.Error("Failed to parse configuration", "error", err, "config_path", *configPath)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully",
		"server_name", cfg.MCP.ServerName,
		"version", cfg.MCP.Version,
		"backends", len(cfg.Backends),
	)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Create proxy from configuration
	srv, err := proxy.NewServerFromConfig(cfg,
		proxy.WithAddr(getEnvOrDefault("SERVER_ADDR", ":8888")),
		proxy.WithBaseURL(getEnvOrDefault("SERVER_BASE_URL", "http://localhost:8888")),
		proxy.WithLogger(logger),
	)
	if err != nil {
		logger.Error("Failed to create proxy from config", "error", err)
		os.Exit(1)
	}
	defer srv.Close()

	logger.Info("Server created successfully with endpoints configured")

	// Start proxy
	if err := srv.Start(ctx); err != nil {
		logger.Error("Failed to start proxy", "error", err)
		os.Exit(1)
	}
}

// getEnvOrDefault returns the value of the environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
