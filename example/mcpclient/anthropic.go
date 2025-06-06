package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// AnthropicProvider implements the LLMProvider interface for Anthropic's Claude
type AnthropicProvider struct {
	apiKey              string
	httpClient          *http.Client
	logger              *slog.Logger
	model               string
	systemPrompt        string
	conversationHistory []ConversationMessage
	conversationConfig  ConversationConfig
}

// Anthropic API structures
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []AnthropicMessage `json:"messages"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AnthropicTool struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	InputSchema AnthropicInputSchema `json:"input_schema"`
}

type AnthropicInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

type AnthropicResponse struct {
	ID      string                  `json:"id"`
	Content []AnthropicContentBlock `json:"content"`
	Usage   AnthropicUsage          `json:"usage"`
	Model   string                  `json:"model"`
	Role    string                  `json:"role"`
}

type AnthropicContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	Name  string                 `json:"name,omitempty"`
	ID    string                 `json:"id,omitempty"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string, logger *slog.Logger) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	return &AnthropicProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:             logger,
		model:              "claude-3-5-haiku-20241022", // Default model
		conversationConfig: DefaultConversationConfig(),
	}, nil
}

// GetProviderName returns the name of this provider
func (p *AnthropicProvider) GetProviderName() string {
	return "Anthropic Claude"
}

// SetModel allows changing the model
func (p *AnthropicProvider) SetModel(model string) {
	p.model = model
	p.logger.Info("Model changed", "new_model", model)
}

// SendMessage sends a message to Claude using function options
func (p *AnthropicProvider) SendMessage(ctx context.Context, options ...SendMessageOption) (*LLMResponse, error) {
	// Apply options
	opts := &SendMessageOptions{
		Role:         "user",
		MaxTokens:    500,
		Temperature:  0.7,
		SystemPrompt: p.systemPrompt,
	}
	for _, fn := range options {
		fn(opts)
	}

	// Validate that message is provided
	if opts.Message != nil {
		p.logger.Info("Sending message to Anthropic", "model", p.model, "message_type", opts.Message.Type, "tools_count", len(opts.Tools), "has_system", opts.SystemPrompt != "", "history_length", len(p.conversationHistory))

		// Convert message content and add to conversation history
		messageText := p.convertMessageContentToText(opts.Message)
		p.AddUserMessage(messageText)
	}

	// Convert MCP tools to Anthropic format
	anthropicTools := p.convertMCPToolsToAnthropic(opts.Tools)

	// Convert conversation history to Anthropic format
	messages := p.convertConversationToAnthropic()

	// Prepare request
	request := AnthropicRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   opts.MaxTokens,
		System:      opts.SystemPrompt,
		Tools:       anthropicTools,
		Temperature: opts.Temperature,
	}

	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		p.logger.Error("Failed to marshal request", "error", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		p.logger.Error("Failed to create HTTP request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	startTime := time.Now()
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.logger.Error("HTTP request failed", "error", err)
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)
	p.logger.Info("Anthropic API request completed", "status", resp.StatusCode, "duration", duration)

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.Error("API request failed", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		p.logger.Error("Failed to decode response", "error", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to unified response
	response := p.convertAnthropicResponse(&anthropicResp)

	// Add assistant response to conversation history
	p.AddAssistantMessage(response.TextContent, response.ToolCalls)

	return response, nil
}

// convertMCPToolsToAnthropic converts MCP tools to Anthropic format
func (p *AnthropicProvider) convertMCPToolsToAnthropic(mcpTools []mcp.Tool) []AnthropicTool {
	if len(mcpTools) == 0 {
		return nil
	}

	tools := make([]AnthropicTool, len(mcpTools))
	for i, mcpTool := range mcpTools {
		properties := make(map[string]interface{})
		required := make([]string, 0)

		// Convert MCP tool schema to Anthropic format
		if mcpTool.InputSchema.Properties != nil {
			properties = mcpTool.InputSchema.Properties
		}
		if len(mcpTool.InputSchema.Required) > 0 {
			required = append(required, mcpTool.InputSchema.Required...)
		}

		tools[i] = AnthropicTool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			InputSchema: AnthropicInputSchema{
				Type:       "object",
				Properties: properties,
				Required:   required,
			},
		}

		p.logger.Debug("Converted MCP tool", "name", mcpTool.Name, "required_params", len(required))
	}

	return tools
}

// convertAnthropicResponse converts Anthropic response to unified format
func (p *AnthropicProvider) convertAnthropicResponse(resp *AnthropicResponse) *LLMResponse {
	llmResp := &LLMResponse{
		Usage: TokenUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
		ToolCalls: make([]ToolCall, 0),
	}

	// Process content blocks
	textParts := make([]string, 0)
	for _, content := range resp.Content {
		switch content.Type {
		case "text":
			if content.Text != "" {
				textParts = append(textParts, content.Text)
			}
		case "tool_use":
			llmResp.ToolCalls = append(llmResp.ToolCalls, ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: content.Input,
			})
			p.logger.Info("Tool use detected", "name", content.Name, "id", content.ID)
		default:
			p.logger.Warn("Unknown content type", "type", content.Type)
		}
	}

	// Join text parts
	if len(textParts) > 0 {
		llmResp.TextContent = strings.Join(textParts, "\n")
	}

	p.logger.Info("Response converted",
		"text_length", len(llmResp.TextContent),
		"tool_calls", len(llmResp.ToolCalls),
		"input_tokens", llmResp.Usage.InputTokens,
		"output_tokens", llmResp.Usage.OutputTokens)

	return llmResp
}

// GetAvailableModels returns available Anthropic models
func (p *AnthropicProvider) GetAvailableModels() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}

// GetCurrentModel returns the currently configured model
func (p *AnthropicProvider) GetCurrentModel() string {
	return p.model
}

// SetSystemPrompt sets the system prompt for this provider
func (p *AnthropicProvider) SetSystemPrompt(systemPrompt string) {
	p.systemPrompt = systemPrompt
	p.logger.Info("System prompt set", "length", len(systemPrompt))
}

// GetSystemPrompt returns the current system prompt
func (p *AnthropicProvider) GetSystemPrompt() string {
	return p.systemPrompt
}

// AddUserMessage adds a user message to the conversation history
func (p *AnthropicProvider) AddUserMessage(content string) {
	p.conversationHistory = append(p.conversationHistory, ConversationMessage{
		Role:    "user",
		Content: content,
	})
	p.optimizeConversationHistory()
}

// AddAssistantMessage adds an assistant message to the conversation history
func (p *AnthropicProvider) AddAssistantMessage(content string, toolCalls []ToolCall) {
	p.conversationHistory = append(p.conversationHistory, ConversationMessage{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	})
	p.optimizeConversationHistory()
}

// AddToolResponse adds a tool response to the conversation history
func (p *AnthropicProvider) AddToolResponse(toolCallID, toolName, content string) {
	p.conversationHistory = append(p.conversationHistory, ConversationMessage{
		Role:       "user",
		Content:    content,
		ToolCallID: toolCallID,
		Name:       toolName,
	})
	p.optimizeConversationHistory()
}

// GetConversationHistory returns the current conversation history
func (p *AnthropicProvider) GetConversationHistory() []ConversationMessage {
	return p.conversationHistory
}

// ClearConversationHistory clears the conversation history
func (p *AnthropicProvider) ClearConversationHistory() {
	p.conversationHistory = make([]ConversationMessage, 0)
	p.logger.Info("Conversation history cleared")
}

// SetConversationConfig sets the conversation optimization configuration
func (p *AnthropicProvider) SetConversationConfig(config ConversationConfig) {
	p.conversationConfig = config
	p.logger.Info("Conversation config updated", "max_messages", config.MaxMessages, "max_tokens", config.MaxTokens)
	p.optimizeConversationHistory()
}

// GetConversationConfig returns the current conversation configuration
func (p *AnthropicProvider) GetConversationConfig() ConversationConfig {
	return p.conversationConfig
}

// estimateTokens provides a rough estimate of tokens in text (4 chars ≈ 1 token)
func (p *AnthropicProvider) estimateTokens(text string) int {
	return len(text) / 4
}

// optimizeConversationHistory trims conversation based on configured limits
func (p *AnthropicProvider) optimizeConversationHistory() {
	if len(p.conversationHistory) == 0 {
		return
	}

	originalLength := len(p.conversationHistory)

	// Apply message count limit
	if p.conversationConfig.MaxMessages > 0 && len(p.conversationHistory) > p.conversationConfig.MaxMessages {
		if p.conversationConfig.UseSlidingWindow {
			// Keep the most recent messages
			startIdx := len(p.conversationHistory) - p.conversationConfig.MaxMessages
			p.conversationHistory = p.conversationHistory[startIdx:]
		} else {
			// Truncate to max
			p.conversationHistory = p.conversationHistory[:p.conversationConfig.MaxMessages]
		}
	}

	// Apply token count limit (approximate)
	if p.conversationConfig.MaxTokens > 0 {
		totalTokens := 0
		for i := len(p.conversationHistory) - 1; i >= 0; i-- {
			msgTokens := p.estimateTokens(p.conversationHistory[i].Content)
			if totalTokens+msgTokens > p.conversationConfig.MaxTokens {
				// Remove older messages
				p.conversationHistory = p.conversationHistory[i+1:]
				break
			}
			totalTokens += msgTokens
		}
	}

	if len(p.conversationHistory) != originalLength {
		p.logger.Info("Conversation history optimized",
			"original_length", originalLength,
			"new_length", len(p.conversationHistory),
			"messages_removed", originalLength-len(p.conversationHistory))
	}
}

// convertConversationToAnthropic converts conversation history to Anthropic format
func (p *AnthropicProvider) convertConversationToAnthropic() []AnthropicMessage {
	messages := make([]AnthropicMessage, 0, len(p.conversationHistory))

	for _, msg := range p.conversationHistory {
		switch msg.Role {
		case "user":
			var content any

			if msg.ToolCallID != "" {
				content = []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": msg.ToolCallID,
						"content":     msg.Content,
					},
				}
			} else {
				content = msg.Content
			}

			messages = append(messages, AnthropicMessage{
				Role:    "user",
				Content: content,
			})
		case "assistant":
			// Handle assistant messages with potential tool calls
			if len(msg.ToolCalls) > 0 {
				// Create content blocks for text and tool calls
				content := make([]interface{}, 0)

				// Add text content if present
				if msg.Content != "" {
					content = append(content, map[string]interface{}{
						"type": "text",
						"text": msg.Content,
					})
				}

				// Add tool calls
				for _, toolCall := range msg.ToolCalls {
					content = append(content, map[string]interface{}{
						"type":  "tool_use",
						"id":    toolCall.ID,
						"name":  toolCall.Name,
						"input": toolCall.Arguments,
					})
				}

				messages = append(messages, AnthropicMessage{
					Role:    "assistant",
					Content: content,
				})
			} else {
				// Simple text message
				messages = append(messages, AnthropicMessage{
					Role:    "assistant",
					Content: msg.Content,
				})
			}
		}
	}

	return messages
}

// convertMessageContentToText converts MessageContent to text for conversation history
func (p *AnthropicProvider) convertMessageContentToText(content *MessageContent) string {
	switch content.Type {
	case "text":
		if text, ok := content.Data.(string); ok {
			return text
		}
		return fmt.Sprintf("%v", content.Data)
	case "image":
		return "[Image content]"
	case "multipart":
		if parts, ok := content.Data.([]MessageContent); ok {
			var textParts []string
			for _, part := range parts {
				textParts = append(textParts, p.convertMessageContentToText(&part))
			}
			return strings.Join(textParts, " ")
		}
		return "[Multipart content]"
	default:
		return fmt.Sprintf("[%s content]", content.Type)
	}
}

// convertMessageContentToAnthropic converts MessageContent to Anthropic-compatible format
func (p *AnthropicProvider) convertMessageContentToAnthropic(content *MessageContent) interface{} {
	switch content.Type {
	case "text":
		return content.Data
	case "image":
		if imageData, ok := content.Data.(map[string]interface{}); ok {
			return map[string]interface{}{
				"type": "image",
				"source": map[string]interface{}{
					"type":       "base64",
					"media_type": imageData["mime_type"],
					"data":       imageData["data"],
				},
			}
		}
		return content.Data
	case "multipart":
		if parts, ok := content.Data.([]MessageContent); ok {
			var convertedParts []interface{}
			for _, part := range parts {
				convertedParts = append(convertedParts, p.convertMessageContentToAnthropic(&part))
			}
			return convertedParts
		}
		return content.Data
	default:
		return content.Data
	}
}
