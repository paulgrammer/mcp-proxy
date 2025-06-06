# Universal MCP Client

A modular, extensible MCP (Model Context Protocol) client that can work with multiple LLM providers including Anthropic Claude, OpenAI GPT, and more.

## Features

- 🔄 **Multi-LLM Support**: Easy integration with different LLM providers
- 🛠️ **Complete MCP Support**: Tools, Resources, and Prompts
- 📊 **Structured Logging**: Uses Go's slog for comprehensive logging
- 🔍 **Debug Capabilities**: List and inspect all MCP server capabilities
- 🏗️ **Modular Architecture**: Easy to extend with new providers

## Supported LLM Providers

- ✅ **Anthropic Claude** (Claude 3.5 Sonnet, Haiku, etc.)
- ✅ **OpenAI GPT** (GPT-4o, GPT-4 Turbo, etc.)
- 🚧 **Ollama** (Coming soon)
- 🚧 **Local Models** (Coming soon)

## Installation

```bash
go mod init mcp-client
go get github.com/mark3labs/mcp-go/client
go get github.com/mark3labs/mcp-go/client/transport
go get github.com/mark3labs/mcp-go/mcp
go get github.com/joho/godotenv
```

## Configuration

### Environment Variables

#### LLM Provider Configuration

You can configure the LLM provider in two ways:

**Option 1: Explicit Provider Selection (Recommended)**
```bash
# Set the provider explicitly
export LLM_PROVIDER="anthropic"  # or "openai", "ollama", "local"

# Then set the required credentials for your chosen provider
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_MODEL="claude-3-5-sonnet-20241022"  # Optional, defaults to Sonnet
```

**Option 2: Auto-Detection**
The client automatically detects which LLM provider to use based on available environment variables:

#### Anthropic Claude
```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_MODEL="claude-3-5-sonnet-20241022"  # Optional, defaults to Sonnet
```

#### OpenAI GPT
```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_MODEL="gpt-4o"  # Optional, defaults to GPT-4o
export OPENAI_BASE_URL="https://api.openai.com/v1"  # Optional, for Azure OpenAI
```

#### Ollama
```bash
export OLLAMA_BASE_URL="http://localhost:11434"
export OLLAMA_MODEL="llama2"  # Optional, defaults to llama2
```

#### Local Models
```bash
export LOCAL_LLM_URL="http://localhost:8080"
export LOCAL_LLM_MODEL="local-model"  # Optional, defaults to local-model
```

#### MCP Server
```bash
export MCP_SERVER_URL="http://localhost:8888/sse"  # Optional, defaults to localhost:8888
```

### .env File Support

Create a `.env` file in your project directory:

```env
# Explicit provider selection (recommended)
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your-anthropic-key
ANTHROPIC_MODEL=claude-3-5-sonnet-20241022

# MCP Server configuration
MCP_SERVER_URL=http://localhost:8888/sse
```

Or for OpenAI:

```env
LLM_PROVIDER=openai
OPENAI_API_KEY=your-openai-key
OPENAI_MODEL=gpt-4o
```

## Usage

### Basic Usage

```bash
go run *.go
```

The client will:
1. Connect to the MCP server
2. Fetch all available tools, resources, and prompts
3. Auto-detect and initialize the appropriate LLM provider
4. Start an interactive chat session

### Interactive Commands

- **Regular chat**: Just type your message
- **`capabilities`**: List all MCP server capabilities (tools, resources, prompts)
- **`provider`**: Show current LLM provider information
- **`exit`**: Quit the application

### Example Session

```
🎉 Universal MCP Client Ready!
🤖 Using LLM Provider: Anthropic Claude

=== 🛠️ MCP Server Capabilities ===

📧 Tools (3):
  1. read_file
     Description: Read contents of a file
     Required params: path

  2. write_file
     Description: Write content to a file
     Required params: path, content

  3. search_web
     Description: Search the web for information
     Required params: query

💬 Start chatting! Commands:
  - Type your message to chat with the LLM
  - Type 'capabilities' to list MCP server capabilities
  - Type 'provider' to show current LLM provider info
  - Type 'exit' to quit

> Can you read the file README.md?

🔧 Executing tool: read_file
✅ Tool result: # My Project...

🤖 Anthropic Claude: I've read your README.md file. It contains information about your project...

📊 Tokens: 150 input, 75 output

>
```

## Architecture

### Core Components

1. **LLMProvider Interface**: Unified interface for all LLM providers
2. **MCPClient**: Handles all MCP server communication
3. **UniversalMCPClient**: Orchestrates LLM and MCP interactions
4. **ProviderFactory**: Creates and configures LLM providers

### Adding New LLM Providers

To add a new LLM provider, implement the `LLMProvider` interface:

```go
type LLMProvider interface {
    SendMessage(ctx context.Context, message string, tools []mcp.Tool) (*LLMResponse, error)
    GetProviderName() string
}
```

Example implementation:

```go
type MyCustomProvider struct {
    apiKey string
    logger *slog.Logger
}

func (p *MyCustomProvider) SendMessage(ctx context.Context, message string, tools []mcp.Tool) (*LLMResponse, error) {
    // Your implementation here
    return &LLMResponse{
        TextContent: "Response from custom provider",
        ToolCalls:   []ToolCall{},
        Usage:       TokenUsage{InputTokens: 100, OutputTokens: 50},
    }, nil
}

func (p *MyCustomProvider) GetProviderName() string {
    return "My Custom Provider"
}
```

Then add it to the `ProviderFactory`:

```go
func (f *ProviderFactory) createCustomProvider(config ProviderConfig) (LLMProvider, error) {
    return &MyCustomProvider{
        apiKey: config.APIKey,
        logger: f.logger,
    }, nil
}
```

## MCP Server Capabilities

The client automatically discovers and logs all MCP server capabilities:

### Tools
- **Discovered automatically** from the MCP server
- **Converted to LLM format** (Anthropic/OpenAI tool schemas)
- **Executed when requested** by the LLM

### Resources
- **Listed and accessible** via URI
- **Read on demand** when needed
- **Supports all MIME types**

### Prompts
- **Template-based prompts** with arguments
- **Retrieved and processed** dynamically
- **Integrated into conversations**

## Logging

The client uses structured logging with different levels:

- **INFO**: General operation logs
- **DEBUG**: Detailed tool execution and API calls
- **ERROR**: Error conditions and failures
- **WARN**: Non-fatal issues

Enable debug logging:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

## Error Handling

The client includes comprehensive error handling:

- **MCP connection failures**: Graceful retry and logging
- **LLM API errors**: Detailed error messages with status codes
- **Tool execution failures**: Continue conversation with error context
- **Invalid configurations**: Clear validation messages

## Performance Features

- **Concurrent tool execution**: Multiple tools can run simultaneously
- **Connection pooling**: Efficient HTTP client reuse
- **Request timeouts**: Configurable timeouts for all operations
- **Token tracking**: Monitor usage across all providers

## Examples

### Using with Different Providers

#### Anthropic Claude
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
go run *.go
```

#### OpenAI GPT
```bash
export OPENAI_API_KEY="sk-..."
go run *.go
```

#### Azure OpenAI
```bash
export OPENAI_API_KEY="your-azure-key"
export OPENAI_BASE_URL="https://your-resource.openai.azure.com/openai/deployments/your-deployment"
export OPENAI_MODEL="gpt-4"
go run *.go
```

### Programmatic Usage

```go
package client

import (
    "context"
    "log/slog"
    "os"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    // Create MCP client
    transport, _ := transport.NewSSE("http://localhost:8888/sse")
    mcpClient := NewMCPClient(transport, logger)
    mcpClient.Initialize(context.Background())

    // Create LLM provider
    factory := NewProviderFactory(logger)
    provider, _ := factory.CreateProviderFromEnv()

    // Create universal client
    client := NewUniversalMCPClient(mcpClient, provider, logger)

    // Process message
    client.ProcessMessage(context.Background(), "Hello, can you help me?")
}
```

## Troubleshooting

### Common Issues

1. **"No LLM provider configured"**
   - Ensure you have set at least one API key environment variable
   - Check that the API key is valid and has sufficient credits

2. **"Failed to connect to MCP server"**
   - Verify the MCP server is running on the specified URL
   - Check firewall and network connectivity

3. **"Tool execution failed"**
   - Review MCP server logs for detailed error information
   - Ensure the tool parameters match the expected schema

### Debug Mode

Enable verbose logging to troubleshoot issues:

```bash
export LOG_LEVEL=debug
go run *.go
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new providers
4. Submit a pull request

### Adding Provider Support

When adding a new provider:

1. Implement the `LLMProvider` interface
2. Add configuration to `ProviderFactory`
3. Add environment variable detection
4. Update documentation and examples
5. Add integration tests

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- [MCP Go SDK](https://github.com/mark3labs/mcp-go) for MCP protocol implementation
- [Anthropic](https://www.anthropic.com/) for Claude API
- [OpenAI](https://openai.com/) for GPT API
