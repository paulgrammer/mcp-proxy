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

// OpenAIProvider implements the LLMProvider interface for OpenAI's GPT models
type OpenAIProvider struct {
	apiKey              string
	httpClient          *http.Client
	logger              *slog.Logger
	model               string
	baseURL             string
	systemPrompt        string
	conversationHistory []ConversationMessage
	conversationConfig  ConversationConfig
}

// OpenAI API structures
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Tools       []OpenAITool    `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type OpenAIMessage struct {
	Role      string           `json:"role"`
	Content   interface{}      `json:"content"`
	Name      string           `json:"name,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

type OpenAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, logger *slog.Logger) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	return &OpenAIProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:             logger,
		model:              "gpt-4o", // Default model
		baseURL:            "https://api.openai.com/v1",
		conversationConfig: DefaultConversationConfig(),
	}, nil
}

// GetProviderName returns the name of this provider
func (p *OpenAIProvider) GetProviderName() string {
	return "OpenAI GPT"
}

// SetModel allows changing the model
func (p *OpenAIProvider) SetModel(model string) {
	p.model = model
	p.logger.Info("Model changed", "new_model", model)
}

// SetBaseURL allows changing the base URL (useful for Azure OpenAI or other compatible APIs)
func (p *OpenAIProvider) SetBaseURL(baseURL string) {
	p.baseURL = strings.TrimSuffix(baseURL, "/")
	p.logger.Info("Base URL changed", "new_url", p.baseURL)
}

// SendMessage sends a message to OpenAI using function options
func (p *OpenAIProvider) SendMessage(ctx context.Context, options ...SendMessageOption) (*LLMResponse, error) {
	// Apply options
	opts := &SendMessageOptions{
		Role:         "user",
		Temperature:  0.7,
		MaxTokens:    4000,
		SystemPrompt: p.systemPrompt,
	}
	for _, option := range options {
		option(opts)
	}

	// Validate that message is provided
	if opts.Message == nil {
		return nil, fmt.Errorf("message content is required - use WithTextMessage() or other message options")
	}

	p.logger.Info("Sending message to OpenAI", "model", p.model, "message_type", opts.Message.Type, "tools_count", len(opts.Tools), "has_system", opts.SystemPrompt != "", "history_length", len(p.conversationHistory))

	// Convert message content and add to conversation history
	messageText := p.convertMessageContentToText(opts.Message)
	p.AddUserMessage(messageText)

	// Convert MCP tools to OpenAI format
	openaiTools := p.convertMCPToolsToOpenAI(opts.Tools)

	// Convert conversation history to OpenAI format
	messages := p.convertConversationToOpenAI(opts.SystemPrompt)

	// Prepare request
	request := OpenAIRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
	}

	// Add tools if available
	if len(openaiTools) > 0 {
		request.Tools = openaiTools
		request.ToolChoice = "auto"
	}

	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		p.logger.Error("Failed to marshal request", "error", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		p.logger.Error("Failed to create HTTP request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Make request
	startTime := time.Now()
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.logger.Error("HTTP request failed", "error", err)
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)
	p.logger.Info("OpenAI API request completed", "status", resp.StatusCode, "duration", duration)

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.Error("API request failed", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		p.logger.Error("Failed to decode response", "error", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to unified response
	response := p.convertOpenAIResponse(&openaiResp)

	// Add assistant response to conversation history
	p.AddAssistantMessage(response.TextContent, response.ToolCalls)

	return response, nil
}

// convertMCPToolsToOpenAI converts MCP tools to OpenAI format
func (p *OpenAIProvider) convertMCPToolsToOpenAI(mcpTools []mcp.Tool) []OpenAITool {
	if len(mcpTools) == 0 {
		return nil
	}

	tools := make([]OpenAITool, len(mcpTools))
	for i, mcpTool := range mcpTools {
		// Convert schema - OpenAI expects the full JSON schema
		parameters := map[string]interface{}{
			"type":       "object",
			"properties": mcpTool.InputSchema.Properties,
		}

		if len(mcpTool.InputSchema.Required) > 0 {
			parameters["required"] = mcpTool.InputSchema.Required
		}

		tools[i] = OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        mcpTool.Name,
				Description: mcpTool.Description,
				Parameters:  parameters,
			},
		}

		p.logger.Debug("Converted MCP tool", "name", mcpTool.Name, "required_params", len(mcpTool.InputSchema.Required))
	}

	return tools
}

// convertOpenAIResponse converts OpenAI response to unified format
func (p *OpenAIProvider) convertOpenAIResponse(resp *OpenAIResponse) *LLMResponse {
	llmResp := &LLMResponse{
		Usage: TokenUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
		ToolCalls: make([]ToolCall, 0),
	}

	// Process choices (typically just one)
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]

		// Handle text content
		if content, ok := choice.Message.Content.(string); ok && content != "" {
			llmResp.TextContent = content
		}

		// Handle tool calls
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCall.Type == "function" {
				// Parse arguments from JSON string
				var arguments map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
					p.logger.Error("Failed to parse tool arguments", "error", err, "arguments", toolCall.Function.Arguments)
					continue
				}

				llmResp.ToolCalls = append(llmResp.ToolCalls, ToolCall{
					ID:        toolCall.ID,
					Name:      toolCall.Function.Name,
					Arguments: arguments,
				})

				p.logger.Info("Tool use detected", "name", toolCall.Function.Name, "id", toolCall.ID)
			}
		}
	}

	p.logger.Info("Response converted",
		"text_length", len(llmResp.TextContent),
		"tool_calls", len(llmResp.ToolCalls),
		"input_tokens", llmResp.Usage.InputTokens,
		"output_tokens", llmResp.Usage.OutputTokens)

	return llmResp
}

// GetAvailableModels returns available OpenAI models
func (p *OpenAIProvider) GetAvailableModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
	}
}

// GetCurrentModel returns the currently configured model
func (p *OpenAIProvider) GetCurrentModel() string {
	return p.model
}

// SetSystemPrompt sets the system prompt for this provider
func (p *OpenAIProvider) SetSystemPrompt(systemPrompt string) {
	p.systemPrompt = systemPrompt
	p.logger.Info("System prompt set", "length", len(systemPrompt))
}

// GetSystemPrompt returns the current system prompt
func (p *OpenAIProvider) GetSystemPrompt() string {
	return p.systemPrompt
}

// AddUserMessage adds a user message to the conversation history
func (p *OpenAIProvider) AddUserMessage(content string) {
	p.conversationHistory = append(p.conversationHistory, ConversationMessage{
		Role:    "user",
		Content: content,
	})
	p.optimizeConversationHistory()
}

// AddAssistantMessage adds an assistant message to the conversation history
func (p *OpenAIProvider) AddAssistantMessage(content string, toolCalls []ToolCall) {
	p.conversationHistory = append(p.conversationHistory, ConversationMessage{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	})
	p.optimizeConversationHistory()
}

// AddToolResponse adds a tool response to the conversation history
func (p *OpenAIProvider) AddToolResponse(toolCallID, toolName, content string) {
	p.conversationHistory = append(p.conversationHistory, ConversationMessage{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		Name:       toolName,
	})
	p.optimizeConversationHistory()
}

// GetConversationHistory returns the current conversation history
func (p *OpenAIProvider) GetConversationHistory() []ConversationMessage {
	return p.conversationHistory
}

// ClearConversationHistory clears the conversation history
func (p *OpenAIProvider) ClearConversationHistory() {
	p.conversationHistory = make([]ConversationMessage, 0)
	p.logger.Info("Conversation history cleared")
}

// SetConversationConfig sets the conversation optimization configuration
func (p *OpenAIProvider) SetConversationConfig(config ConversationConfig) {
	p.conversationConfig = config
	p.logger.Info("Conversation config updated", "max_messages", config.MaxMessages, "max_tokens", config.MaxTokens)
	p.optimizeConversationHistory()
}

// GetConversationConfig returns the current conversation configuration
func (p *OpenAIProvider) GetConversationConfig() ConversationConfig {
	return p.conversationConfig
}

// estimateTokens provides a rough estimate of tokens in text (4 chars ≈ 1 token)
func (p *OpenAIProvider) estimateTokens(text string) int {
	return len(text) / 4
}

// optimizeConversationHistory trims conversation based on configured limits
func (p *OpenAIProvider) optimizeConversationHistory() {
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

// convertConversationToOpenAI converts conversation history to OpenAI format
func (p *OpenAIProvider) convertConversationToOpenAI(systemPrompt string) []OpenAIMessage {
	messages := make([]OpenAIMessage, 0, len(p.conversationHistory)+1)

	// Add system message if provided
	if systemPrompt != "" {
		messages = append(messages, OpenAIMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Convert conversation history
	for _, msg := range p.conversationHistory {
		switch msg.Role {
		case "user":
			messages = append(messages, OpenAIMessage{
				Role:    "user",
				Content: msg.Content,
			})
		case "assistant":
			// Handle assistant messages with potential tool calls
			openaiMsg := OpenAIMessage{
				Role:    "assistant",
				Content: msg.Content,
			}

			// Add tool calls if present
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]OpenAIToolCall, len(msg.ToolCalls))
				for i, toolCall := range msg.ToolCalls {
					// Marshal arguments to JSON string
					argBytes, _ := json.Marshal(toolCall.Arguments)
					toolCalls[i] = OpenAIToolCall{
						ID:   fmt.Sprintf("call_%d", i),
						Type: "function",
						Function: OpenAIFunctionCall{
							Name:      toolCall.Name,
							Arguments: string(argBytes),
						},
					}
				}
				openaiMsg.ToolCalls = toolCalls
			}

			messages = append(messages, openaiMsg)
		case "tool":
			// Tool response message
			messages = append(messages, OpenAIMessage{
				Role:    "tool",
				Content: msg.Content,
				Name:    msg.Name,
			})
		}
	}

	return messages
}

// convertMessageContentToText converts MessageContent to text for conversation history
func (p *OpenAIProvider) convertMessageContentToText(content *MessageContent) string {
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

// convertMessageContentToOpenAI converts MessageContent to OpenAI-compatible format
func (p *OpenAIProvider) convertMessageContentToOpenAI(content *MessageContent) interface{} {
	switch content.Type {
	case "text":
		return content.Data
	case "image":
		if imageData, ok := content.Data.(map[string]interface{}); ok {
			return []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "[Image content]", // OpenAI vision API format would be different
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": fmt.Sprintf("data:%s;base64,%s", imageData["mime_type"], imageData["data"]),
					},
				},
			}
		}
		return content.Data
	case "multipart":
		if parts, ok := content.Data.([]MessageContent); ok {
			var convertedParts []interface{}
			for _, part := range parts {
				if partContent := p.convertMessageContentToOpenAI(&part); partContent != nil {
					if partArray, ok := partContent.([]interface{}); ok {
						convertedParts = append(convertedParts, partArray...)
					} else {
						convertedParts = append(convertedParts, map[string]interface{}{
							"type": "text",
							"text": fmt.Sprintf("%v", partContent),
						})
					}
				}
			}
			return convertedParts
		}
		return content.Data
	default:
		return content.Data
	}
}
