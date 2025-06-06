package client

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// ConversationMessage represents a single message in the conversation history
type ConversationMessage struct {
	Role       string     `json:"role"`                   // "user", "assistant", "system", "tool"
	Content    string     `json:"content"`                // Message content
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // Tool calls made by assistant
	ToolCallID string     `json:"tool_call_id,omitempty"` // ID for tool response messages
	Name       string     `json:"name,omitempty"`         // Tool name for tool response messages
}

// ConversationConfig holds configuration for conversation management
type ConversationConfig struct {
	MaxMessages      int  // Maximum number of messages to keep (0 = unlimited)
	MaxTokens        int  // Approximate max tokens to keep (0 = unlimited)
	KeepSystemMsg    bool // Always keep system message
	UseSlidingWindow bool // Use sliding window vs truncation
}

// DefaultConversationConfig returns sensible defaults
func DefaultConversationConfig() ConversationConfig {
	return ConversationConfig{
		MaxMessages:      20,   // Keep last 20 messages
		MaxTokens:        8000, // Rough estimate ~8k tokens
		KeepSystemMsg:    true,
		UseSlidingWindow: true,
	}
}

// MessageContent represents different types of content that can be sent to LLMs
type MessageContent struct {
	Type string      `json:"type"` // "text", "image", "multipart", etc.
	Data interface{} `json:"data"` // The actual content (string, []byte, complex structures, etc.)
}

// SendMessageOptions holds configuration for sending messages to LLMs
type SendMessageOptions struct {
	Message      *MessageContent // The message content and type
	Tools        []mcp.Tool
	Role         string
	SystemPrompt string
	MaxTokens    int
	Temperature  float64
}

// SendMessageOption is a function that configures SendMessageOptions
type SendMessageOption func(*SendMessageOptions)

// WithTextMessage sets a text message content
func WithTextMessage(text string) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.Message = &MessageContent{
			Type: "text",
			Data: text,
		}
	}
}

// WithMessage sets custom message content with specified type
func WithMessage(messageType string, data interface{}) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.Message = &MessageContent{
			Type: messageType,
			Data: data,
		}
	}
}

// WithImageMessage sets an image message content
func WithImageMessage(imageData []byte, mimeType string) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.Message = &MessageContent{
			Type: "image",
			Data: map[string]interface{}{
				"data":      imageData,
				"mime_type": mimeType,
			},
		}
	}
}

// WithMultipartMessage sets a multipart message content (text + images/files)
func WithMultipartMessage(parts []MessageContent) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.Message = &MessageContent{
			Type: "multipart",
			Data: parts,
		}
	}
}

// WithTools sets the available MCP tools for the message
func WithTools(tools []mcp.Tool) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.Tools = tools
	}
}

// WithRole sets the message role (defaults to "user")
func WithRole(role string) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.Role = role
	}
}

// WithSystemPrompt sets the system prompt for this specific request
func WithSystemPrompt(systemPrompt string) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.SystemPrompt = systemPrompt
	}
}

// WithMaxTokens sets the maximum tokens for the response
func WithMaxTokens(maxTokens int) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.MaxTokens = maxTokens
	}
}

// WithTemperature sets the temperature for the response (0.0-1.0)
func WithTemperature(temperature float64) SendMessageOption {
	return func(opts *SendMessageOptions) {
		opts.Temperature = temperature
	}
}

// WithOverride overrides message options
func WithOverride(overrides *SendMessageOptions) SendMessageOption {
	return func(opts *SendMessageOptions) {
		*opts = *overrides
	}
}

// LLMProvider interface for different LLM implementations
type LLMProvider interface {
	SendMessage(ctx context.Context, options ...SendMessageOption) (*LLMResponse, error)
	SetSystemPrompt(systemPrompt string)
	GetSystemPrompt() string
	GetProviderName() string

	// Conversation management
	AddUserMessage(content string)
	AddAssistantMessage(content string, toolCalls []ToolCall)
	AddToolResponse(toolCallID, toolName, content string)
	GetConversationHistory() []ConversationMessage
	ClearConversationHistory()

	// Conversation optimization
	SetConversationConfig(config ConversationConfig)
	GetConversationConfig() ConversationConfig
}

// LLMResponse represents a unified response from any LLM
type LLMResponse struct {
	TextContent string
	ToolCalls   []ToolCall
	Usage       TokenUsage
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]interface{}
}

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// MCPCapabilities holds all available MCP server capabilities
type MCPCapabilities struct {
	Tools     []mcp.Tool
	Resources []mcp.Resource
	Prompts   []mcp.Prompt
}

// MCPClient handles MCP server communication
type MCPClient struct {
	client       *client.Client
	capabilities MCPCapabilities
	logger       *slog.Logger
}

// UniversalMCPClient integrates MCP with any LLM provider
type UniversalMCPClient struct {
	mcpClient   *MCPClient
	llmProvider LLMProvider
	logger      *slog.Logger
}

// NewMCPClient creates a new MCP client
func NewMCPClient(transport transport.Interface, logger *slog.Logger) *MCPClient {
	return &MCPClient{
		client: client.NewClient(transport),
		logger: logger,
	}
}

// Initialize connects to MCP server and fetches all capabilities
func (c *MCPClient) Initialize(ctx context.Context) error {
	c.logger.Info("Starting MCP client initialization")

	// Start MCP client
	if err := c.client.Start(ctx); err != nil {
		c.logger.Error("Failed to start MCP client", "error", err)
		return fmt.Errorf("failed to start MCP client: %w", err)
	}

	// Initialize MCP session
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "universal-mcp-client",
		Version: "2.0.0",
	}

	initResp, err := c.client.Initialize(ctx, initReq)
	if err != nil {
		c.logger.Error("Failed to initialize MCP client", "error", err)
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	c.logger.Info("MCP client initialized",
		"server_name", initResp.ServerInfo.Name,
		"server_version", initResp.ServerInfo.Version,
		"protocol_version", initResp.ProtocolVersion)

	// Fetch all capabilities
	if err := c.fetchCapabilities(ctx); err != nil {
		return fmt.Errorf("failed to fetch capabilities: %w", err)
	}

	return nil
}

// fetchCapabilities retrieves tools, resources, and prompts from MCP server
func (c *MCPClient) fetchCapabilities(ctx context.Context) error {
	c.logger.Info("Fetching MCP server capabilities")

	// Fetch tools
	if err := c.fetchTools(ctx); err != nil {
		c.logger.Warn("Failed to fetch tools", "error", err)
	}

	// Fetch resources
	if err := c.fetchResources(ctx); err != nil {
		c.logger.Warn("Failed to fetch resources", "error", err)
	}

	// Fetch prompts
	if err := c.fetchPrompts(ctx); err != nil {
		c.logger.Warn("Failed to fetch prompts", "error", err)
	}

	c.logCapabilities()
	return nil
}

func (c *MCPClient) fetchTools(ctx context.Context) error {
	toolsResp, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return err
	}

	c.capabilities.Tools = toolsResp.Tools
	c.logger.Info("Fetched tools", "count", len(c.capabilities.Tools))
	return nil
}

func (c *MCPClient) fetchResources(ctx context.Context) error {
	resourcesResp, err := c.client.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return err
	}

	c.capabilities.Resources = resourcesResp.Resources
	c.logger.Info("Fetched resources", "count", len(c.capabilities.Resources))
	return nil
}

func (c *MCPClient) fetchPrompts(ctx context.Context) error {
	promptsResp, err := c.client.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return err
	}

	c.capabilities.Prompts = promptsResp.Prompts
	c.logger.Info("Fetched prompts", "count", len(c.capabilities.Prompts))
	return nil
}

func (c *MCPClient) logCapabilities() {
	c.logger.Info("=== MCP Server Capabilities ===")

	// Log tools
	if len(c.capabilities.Tools) > 0 {
		c.logger.Info("📧 Available Tools:")
		for _, tool := range c.capabilities.Tools {
			c.logger.Info("  Tool",
				"name", tool.Name,
				"description", tool.Description)
		}
	}

	// Log resources
	if len(c.capabilities.Resources) > 0 {
		c.logger.Info("📄 Available Resources:")
		for _, resource := range c.capabilities.Resources {
			c.logger.Info("  Resource",
				"uri", resource.URI,
				"name", resource.Name,
				"description", resource.Description,
				"mime_type", resource.MIMEType)
		}
	}

	// Log prompts
	if len(c.capabilities.Prompts) > 0 {
		c.logger.Info("💭 Available Prompts:")
		for _, prompt := range c.capabilities.Prompts {
			c.logger.Info("  Prompt",
				"name", prompt.Name,
				"description", prompt.Description)
		}
	}

	c.logger.Info("=== End Capabilities ===")
}

// CallTool executes an MCP tool
func (c *MCPClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	c.logger.Info("Calling MCP tool", "name", name, "arguments", arguments)

	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = name
	callReq.Params.Arguments = arguments

	result, err := c.client.CallTool(ctx, callReq)
	if err != nil {
		c.logger.Error("Tool call failed", "name", name, "error", err)
		return nil, err
	}

	c.logger.Info("Tool call successful", "name", name, "content_count", len(result.Content))
	return result, nil
}

// ReadResource reads a resource from MCP server
func (c *MCPClient) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	c.logger.Info("Reading MCP resource", "uri", uri)

	readReq := mcp.ReadResourceRequest{}
	readReq.Params.URI = uri

	result, err := c.client.ReadResource(ctx, readReq)
	if err != nil {
		c.logger.Error("Resource read failed", "uri", uri, "error", err)
		return nil, err
	}

	c.logger.Info("Resource read successful", "uri", uri, "content_count", len(result.Contents))
	return result, nil
}

// GetPrompt retrieves a prompt from MCP server
func (c *MCPClient) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.GetPromptResult, error) {
	c.logger.Info("Getting MCP prompt", "name", name, "arguments", arguments)

	// Convert map[string]interface{} to map[string]string for prompts
	stringArgs := make(map[string]string)
	for k, v := range arguments {
		if s, ok := v.(string); ok {
			stringArgs[k] = s
		} else {
			stringArgs[k] = fmt.Sprintf("%v", v)
		}
	}

	promptReq := mcp.GetPromptRequest{}
	promptReq.Params.Name = name
	promptReq.Params.Arguments = stringArgs

	result, err := c.client.GetPrompt(ctx, promptReq)
	if err != nil {
		c.logger.Error("Prompt retrieval failed", "name", name, "error", err)
		return nil, err
	}

	c.logger.Info("Prompt retrieved successfully", "name", name, "message_count", len(result.Messages))
	return result, nil
}

// NewUniversalMCPClient creates a new universal MCP client
func NewUniversalMCPClient(mcpClient *MCPClient, llmProvider LLMProvider, logger *slog.Logger) *UniversalMCPClient {
	return &UniversalMCPClient{
		mcpClient:   mcpClient,
		llmProvider: llmProvider,
		logger:      logger,
	}
}

// ProcessMessage handles a user message and coordinates LLM and MCP interactions
func (c *UniversalMCPClient) ProcessMessage(ctx context.Context, options ...SendMessageOption) error {
	c.logger.Info("Processing user message", "provider", c.llmProvider.GetProviderName())

	opts := &SendMessageOptions{
		Role:        "user",
		MaxTokens:   4000,
		Temperature: 0.7,
		Tools:       c.mcpClient.capabilities.Tools,
	}
	for _, fn := range options {
		fn(opts)
	}

	// Send to LLM with available tools
	response, err := c.llmProvider.SendMessage(ctx, WithOverride(opts))
	if err != nil {
		c.logger.Error("LLM request failed", "error", err)
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Process LLM response
	if response.TextContent != "" {
		fmt.Printf("🤖 %s: %s\n", c.llmProvider.GetProviderName(), response.TextContent)
	}

	// Execute any tool calls
	for _, toolCall := range response.ToolCalls {
		if err := c.executeToolCall(ctx, toolCall); err != nil {
			c.logger.Error("Tool execution failed", "tool", toolCall.Name, "error", err)
			fmt.Printf("❌ Failed to execute tool %s: %v\n", toolCall.Name, err)
			continue
		}
	}

	// If tool calls were executed, send tool responses back to LLM
	if len(response.ToolCalls) > 0 {
		c.logger.Info("Sending tool responses back to LLM")

		// Send empty message to continue conversation with tool results
		toolResponse, err := c.llmProvider.SendMessage(ctx, WithOverride(&SendMessageOptions{
			Tools:        opts.Tools,
			MaxTokens:    opts.MaxTokens,
			Temperature:  opts.Temperature,
			SystemPrompt: opts.SystemPrompt,
		}))

		if err != nil {
			c.logger.Error("Failed to send tool responses to LLM", "error", err)
			return fmt.Errorf("failed to send tool responses to LLM: %w", err)
		}

		// Display LLM response to tool results
		if toolResponse.TextContent != "" {
			fmt.Printf("🤖 %s: %s\n", c.llmProvider.GetProviderName(), toolResponse.TextContent)
		}

		// Handle any additional tool calls (recursive)
		for _, toolCall := range toolResponse.ToolCalls {
			if err := c.executeToolCall(ctx, toolCall); err != nil {
				c.logger.Error("Tool execution failed", "tool", toolCall.Name, "error", err)
				fmt.Printf("❌ Failed to execute tool %s: %v\n", toolCall.Name, err)
				continue
			}
		}
	}

	// Log token usage
	c.logger.Info("Token usage",
		"input_tokens", response.Usage.InputTokens,
		"output_tokens", response.Usage.OutputTokens)

	fmt.Printf("📊 Tokens: %d input, %d output\n", response.Usage.InputTokens, response.Usage.OutputTokens)

	return nil
}

func (c *UniversalMCPClient) executeToolCall(ctx context.Context, toolCall ToolCall) error {
	c.logger.Info("Executing tool call", "name", toolCall.Name)
	fmt.Printf("🔧 Executing tool: %s\n", toolCall.Name)

	result, err := c.mcpClient.CallTool(ctx, toolCall.Name, toolCall.Arguments)
	if err != nil {
		return err
	}

	// Display tool result
	for _, content := range result.Content {
		// Handle different content types using type assertion
		if textContent, ok := content.(mcp.TextContent); ok {
			fmt.Printf("✅ Tool result: %s\n", textContent.Text)

			// Add tool response to conversation history
			c.llmProvider.AddToolResponse(toolCall.ID, toolCall.Name, textContent.Text)
		} else {
			// Generic content handling
			fmt.Printf("✅ Tool result: %+v\n", content)

			// Add tool response to conversation history
			c.llmProvider.AddToolResponse(toolCall.ID, toolCall.Name, fmt.Sprintf("%+v", content))
		}
	}

	return nil
}

// ListCapabilities displays all available MCP capabilities
func (c *UniversalMCPClient) ListCapabilities() {
	fmt.Println("\n=== 🛠️ MCP Server Capabilities ===")

	if len(c.mcpClient.capabilities.Tools) > 0 {
		fmt.Printf("\n📧 Tools (%d):\n", len(c.mcpClient.capabilities.Tools))
		for i, tool := range c.mcpClient.capabilities.Tools {
			fmt.Printf("  %d. %s\n", i+1, tool.Name)
			fmt.Printf("     Description: %s\n", tool.Description)
			if len(tool.InputSchema.Required) > 0 {
				fmt.Printf("     Required params: %s\n", strings.Join(tool.InputSchema.Required, ", "))
			}
			fmt.Println()
		}
	}

	if len(c.mcpClient.capabilities.Resources) > 0 {
		fmt.Printf("\n📄 Resources (%d):\n", len(c.mcpClient.capabilities.Resources))
		for i, resource := range c.mcpClient.capabilities.Resources {
			fmt.Printf("  %d. %s\n", i+1, resource.URI)
			if resource.Name != "" {
				fmt.Printf("     Name: %s\n", resource.Name)
			}
			if resource.Description != "" {
				fmt.Printf("     Description: %s\n", resource.Description)
			}
			if resource.MIMEType != "" {
				fmt.Printf("     MIME Type: %s\n", resource.MIMEType)
			}
			fmt.Println()
		}
	}

	if len(c.mcpClient.capabilities.Prompts) > 0 {
		fmt.Printf("\n💭 Prompts (%d):\n", len(c.mcpClient.capabilities.Prompts))
		for i, prompt := range c.mcpClient.capabilities.Prompts {
			fmt.Printf("  %d. %s\n", i+1, prompt.Name)
			fmt.Printf("     Description: %s\n", prompt.Description)
			if len(prompt.Arguments) > 0 {
				fmt.Printf("     Arguments: %d defined\n", len(prompt.Arguments))
			}
			fmt.Println()
		}
	}

	fmt.Println("=== End Capabilities ===\n")
}

// ShowProviderInfo displays current LLM provider information
func (c *UniversalMCPClient) ShowProviderInfo() {
	fmt.Printf("\n=== 🤖 Current LLM Provider ===\n")
	fmt.Printf("Provider: %s\n", c.llmProvider.GetProviderName())

	// Try to get model info if provider supports it
	if modelProvider, ok := c.llmProvider.(interface{ GetCurrentModel() string }); ok {
		fmt.Printf("Model: %s\n", modelProvider.GetCurrentModel())
	}

	if modelsProvider, ok := c.llmProvider.(interface{ GetAvailableModels() []string }); ok {
		models := modelsProvider.GetAvailableModels()
		if len(models) > 0 {
			fmt.Printf("Available models: %s\n", strings.Join(models, ", "))
		}
	}

	if modelsProvider, ok := c.llmProvider.(interface{ GetConversationHistory() []ConversationMessage }); ok {
		conversation := modelsProvider.GetConversationHistory()
		fmt.Printf("Messages in conversation: %d\n", len(conversation))
	}

	fmt.Println("=== End Provider Info ===\n")
}
