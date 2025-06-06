package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/client/transport"
	client "github.com/paulgrammer/mcp-proxy/example/mcpclient"
)

func main() {
	var (
		mcpURL = flag.String("mcp-url", "http://localhost:8888/sse", "MCP server URL")
		help   = flag.Bool("help", false, "Show help message")
	)

	if *help {
		flag.Usage()
		return
	}

	flag.Parse()

	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting Universal MCP Client")

	// Get MCP server URL from environment or use default
	logger.Info("Connecting to MCP server", "url", mcpURL)

	// Initialize MCP transport
	transport, err := transport.NewSSE(*mcpURL)
	if err != nil {
		logger.Error("Failed to create transport", "error", err)
		os.Exit(1)
	}

	// Create MCP client
	mcpClient := client.NewMCPClient(transport, logger)

	// Initialize MCP client
	if err := mcpClient.Initialize(context.Background()); err != nil {
		logger.Error("Failed to initialize MCP client", "error", err)
		os.Exit(1)
	}

	// Create provider factory
	factory := client.NewProviderFactory(logger)

	// Create LLM provider from environment
	llmProvider, err := factory.CreateProviderFromEnv()
	if err != nil {
		logger.Error("Failed to create LLM provider", "error", err)
		fmt.Println("❌ No LLM provider configured!")
		fmt.Println("   Configuration options:")
		fmt.Println("   1. Set LLM_PROVIDER environment variable to choose provider:")
		fmt.Println("      - LLM_PROVIDER=anthropic (requires ANTHROPIC_API_KEY)")
		fmt.Println("      - LLM_PROVIDER=openai (requires OPENAI_API_KEY)")
		fmt.Println("      - LLM_PROVIDER=ollama (requires OLLAMA_BASE_URL)")
		fmt.Println("      - LLM_PROVIDER=local (requires LOCAL_LLM_URL)")
		fmt.Println("   2. Or let it auto-detect by setting any of:")
		fmt.Println("      - ANTHROPIC_API_KEY (for Claude)")
		fmt.Println("      - OPENAI_API_KEY (for GPT)")
		fmt.Println("      - OLLAMA_BASE_URL (for Ollama)")
		fmt.Println("      - LOCAL_LLM_URL (for local models)")
		os.Exit(1)
	}

	// Set Prompt
	llmProvider.SetSystemPrompt("You're an helpful assistant")

	// Create universal client
	universalClient := client.NewUniversalMCPClient(mcpClient, llmProvider, logger)

	logger.Info("Universal MCP Client initialized successfully")
	fmt.Println("🎉 Universal MCP Client Ready!")
	fmt.Printf("🤖 Using LLM Provider: %s\n", llmProvider.GetProviderName())

	// Show available capabilities
	universalClient.ListCapabilities()

	fmt.Println("💬 Start chatting! Commands:")
	fmt.Println("  - Type your message to chat with the LLM")
	fmt.Println("  - Type 'capabilities' to list MCP server capabilities")
	fmt.Println("  - Type 'provider' to show current LLM provider info")
	fmt.Println("  - Type 'exit' to quit")
	fmt.Print("\n> ")

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			fmt.Print("> ")
			continue
		}

		switch input {
		case "exit":
			fmt.Println("Goodbye! 👋")
			return
		case "capabilities":
			universalClient.ListCapabilities()
		case "provider":
			universalClient.ShowProviderInfo()
		default:
			if err := universalClient.ProcessMessage(context.Background(), client.WithTextMessage(input)); err != nil {
				logger.Error("Failed to process message", "error", err)
				fmt.Printf("❌ Error: %v\n", err)
			}
		}

		fmt.Print("\n> ")
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading input", "error", err)
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
