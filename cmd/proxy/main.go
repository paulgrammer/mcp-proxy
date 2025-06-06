package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	proxy "github.com/paulgrammer/mcp-proxy"
)

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

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

	// Create proxy with options
	srv, err := proxy.NewServer(ctx,
		proxy.WithName(getEnvOrDefault("SERVER_NAME", "mcp-proxy")),
		proxy.WithAddr(getEnvOrDefault("SERVER_ADDR", ":8888")),
		proxy.WithBaseURL(getEnvOrDefault("SERVER_BASE_URL", "http://localhost:8888")),
		proxy.WithLogger(logger),
	)
	if err != nil {
		logger.Error("Failed to create proxy", "error", err)
		os.Exit(1)
	}
	defer srv.Close()

	// Start proxy
	if err := srv.Start(); err != nil {
		logger.Error("Failed to start proxy", "error", err)
		os.Exit(1)
	}

	logger.Info("Server started successfully")

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Shutting down proxy...")
}

// getEnvOrDefault returns the value of the environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
