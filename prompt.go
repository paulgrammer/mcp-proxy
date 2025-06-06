package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// HTTPPromptHandler handles prompt requests by making HTTP requests
type HTTPPromptHandler struct {
	endpoint *Endpoint
	backend  *Backend
	logger   *slog.Logger
	client   *http.Client
}

// NewHTTPPromptHandler creates a new HTTP prompt handler
func NewHTTPPromptHandler(endpoint *Endpoint, backend *Backend, logger *slog.Logger) *HTTPPromptHandler {
	return &HTTPPromptHandler{
		endpoint: endpoint,
		backend:  backend,
		logger:   logger,
		client: &http.Client{
			Timeout: endpoint.ResponseTimeout,
		},
	}
}

// CreateMCPPrompt creates an MCP prompt from endpoint configuration
func (h *HTTPPromptHandler) CreateMCPPrompt() mcp.Prompt {
	var promptOptions []mcp.PromptOption
	promptOptions = append(promptOptions, mcp.WithPromptDescription(h.endpoint.Description))

	// Add arguments based on endpoint configuration
	for _, param := range h.endpoint.BodyParams {
		promptOptions = append(promptOptions, h.createArgumentOption(param))
	}
	for _, param := range h.endpoint.QueryParameters {
		promptOptions = append(promptOptions, h.createArgumentOption(param))
	}
	for _, param := range h.endpoint.PathParameters {
		promptOptions = append(promptOptions, h.createArgumentOption(param))
	}

	return mcp.NewPrompt(h.endpoint.Name, promptOptions...)
}

// createArgumentOption creates an argument option for the MCP prompt
func (h *HTTPPromptHandler) createArgumentOption(param *Param) mcp.PromptOption {
	options := []mcp.ArgumentOption{
		mcp.ArgumentDescription(param.Description),
	}

	if param.Required {
		options = append(options, mcp.RequiredArgument())
	}

	return mcp.WithArgument(param.Identifier, options...)
}

// Handler handles prompt requests
func (h *HTTPPromptHandler) Handler(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Get arguments from the request - convert from map[string]string to map[string]any
	arguments := make(map[string]any)
	if req.Params.Arguments != nil {
		for k, v := range req.Params.Arguments {
			arguments[k] = v
		}
	}

	// Build the URL with path parameters
	url, err := h.buildURL(arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Build query parameters
	queryParams := h.buildQueryParams(arguments)
	if len(queryParams) > 0 {
		url += "?" + queryParams
	}

	// Build request body
	body, err := h.buildRequestBody(arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to build request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, string(h.endpoint.Method), url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	h.addHeaders(httpReq, arguments)

	h.logger.Debug("Making HTTP request for prompt",
		"prompt", h.endpoint.Name,
		"method", h.endpoint.Method,
		"url", url,
	)

	// Make the HTTP request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle response
	return h.handleResponse(resp)
}

// buildURL constructs the full URL with path parameters substituted
func (h *HTTPPromptHandler) buildURL(arguments map[string]any) (string, error) {
	url := h.backend.BaseURL + h.endpoint.Path

	// Replace path parameters
	for _, param := range h.endpoint.PathParameters {
		value, exists := arguments[param.Identifier]
		if !exists && param.Required {
			return "", fmt.Errorf("required path parameter '%s' not provided", param.Identifier)
		}
		if exists {
			placeholder := fmt.Sprintf("{%s}", param.Identifier)
			url = strings.ReplaceAll(url, placeholder, fmt.Sprintf("%v", value))
		}
	}

	return url, nil
}

// buildQueryParams constructs query parameters from arguments
func (h *HTTPPromptHandler) buildQueryParams(arguments map[string]any) string {
	var params []string

	for _, param := range h.endpoint.QueryParameters {
		value, exists := arguments[param.Identifier]
		if exists {
			params = append(params, fmt.Sprintf("%s=%v", param.Identifier, value))
		}
	}

	return strings.Join(params, "&")
}

// buildRequestBody constructs the JSON request body
func (h *HTTPPromptHandler) buildRequestBody(arguments map[string]any) ([]byte, error) {
	if len(h.endpoint.BodyParams) == 0 {
		return nil, nil
	}

	body := make(map[string]any)
	for _, param := range h.endpoint.BodyParams {
		value, exists := arguments[param.Identifier]
		if exists {
			body[param.Identifier] = value
		} else if param.Required {
			return nil, fmt.Errorf("required body parameter '%s' not provided", param.Identifier)
		}
	}

	if len(body) == 0 {
		return nil, nil
	}

	return json.Marshal(body)
}

// addHeaders adds headers to the HTTP request
func (h *HTTPPromptHandler) addHeaders(req *http.Request, arguments map[string]any) {
	// Add default headers from backend
	for _, header := range h.backend.DefaultHeaders {
		req.Header.Set(header.Name, header.Value)
	}

	// Add endpoint-specific headers
	for _, header := range h.endpoint.Headers {
		if header.Type == CONSTANT {
			req.Header.Set(header.Name, header.Value)
		} else if header.Type == DYNAMIC {
			// For dynamic headers, try to get value from arguments
			if value, exists := arguments[header.Name]; exists {
				req.Header.Set(header.Name, fmt.Sprintf("%v", value))
			}
		}
	}

	// Set content type for JSON if we have body parameters
	if len(h.endpoint.BodyParams) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
}

// handleResponse processes the HTTP response and returns MCP prompt result
func (h *HTTPPromptHandler) handleResponse(resp *http.Response) (*mcp.GetPromptResult, error) {
	// Read response body
	var responseBody bytes.Buffer
	if _, err := responseBody.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	responseText := responseBody.String()

	// Check if the request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.logger.Debug("Prompt request successful",
			"prompt", h.endpoint.Name,
			"status", resp.StatusCode,
		)

		// Try to parse the response as a structured prompt
		var promptData map[string]any
		if json.Unmarshal(responseBody.Bytes(), &promptData) == nil {
			// Response is JSON, try to extract prompt messages
			return h.parseStructuredPrompt(promptData)
		} else {
			// Response is plain text, create a simple prompt
			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Generated prompt from %s", h.endpoint.Name),
				Messages: []mcp.PromptMessage{
					{
						Role: mcp.RoleUser,
						Content: mcp.TextContent{
							Type: "text",
							Text: responseText,
						},
					},
				},
			}, nil
		}
	} else {
		h.logger.Error("Prompt request failed",
			"prompt", h.endpoint.Name,
			"status", resp.StatusCode,
			"response", responseText,
		)

		return nil, fmt.Errorf("prompt request failed with status %d: %s", resp.StatusCode, responseText)
	}
}

// parseStructuredPrompt attempts to parse a structured JSON response into prompt messages
func (h *HTTPPromptHandler) parseStructuredPrompt(data map[string]any) (*mcp.GetPromptResult, error) {
	result := &mcp.GetPromptResult{
		Description: fmt.Sprintf("Generated prompt from %s", h.endpoint.Name),
		Messages:    []mcp.PromptMessage{},
	}

	// Check if there's a description field
	if desc, exists := data["description"]; exists {
		if descStr, ok := desc.(string); ok {
			result.Description = descStr
		}
	}

	// Check if there are messages
	if messages, exists := data["messages"]; exists {
		if msgArray, ok := messages.([]interface{}); ok {
			for _, msg := range msgArray {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					promptMsg := h.parsePromptMessage(msgMap)
					if promptMsg != nil {
						result.Messages = append(result.Messages, *promptMsg)
					}
				}
			}
		}
	}

	// If no messages were found, create a simple text message with the entire response
	if len(result.Messages) == 0 {
		responseText, _ := json.Marshal(data)
		result.Messages = []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: string(responseText),
				},
			},
		}
	}

	return result, nil
}

// parsePromptMessage parses a single message from JSON data
func (h *HTTPPromptHandler) parsePromptMessage(data map[string]interface{}) *mcp.PromptMessage {
	msg := &mcp.PromptMessage{}

	// Parse role
	if role, exists := data["role"]; exists {
		if roleStr, ok := role.(string); ok {
			switch strings.ToLower(roleStr) {
			case "user":
				msg.Role = mcp.RoleUser
			case "assistant":
				msg.Role = mcp.RoleAssistant
			default:
				msg.Role = mcp.RoleUser
			}
		}
	} else {
		msg.Role = mcp.RoleUser
	}

	// Parse content
	if content, exists := data["content"]; exists {
		if contentStr, ok := content.(string); ok {
			msg.Content = mcp.TextContent{
				Type: "text",
				Text: contentStr,
			}
		} else if contentMap, ok := content.(map[string]interface{}); ok {
			// Handle structured content
			if contentType, exists := contentMap["type"]; exists {
				if typeStr, ok := contentType.(string); ok && typeStr == "text" {
					if text, exists := contentMap["text"]; exists {
						if textStr, ok := text.(string); ok {
							msg.Content = mcp.TextContent{
								Type: "text",
								Text: textStr,
							}
						}
					}
				}
			}
		}
	}

	// If no content was parsed, return nil
	if msg.Content == nil {
		return nil
	}

	return msg
}
