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

// HTTPToolHandler handles tool execution by making HTTP requests
type HTTPToolHandler struct {
	endpoint *Endpoint
	backend  *Backend
	logger   *slog.Logger
	client   *http.Client
}

// NewHTTPToolHandler creates a new HTTP tool handler
func NewHTTPToolHandler(endpoint *Endpoint, backend *Backend, logger *slog.Logger) *HTTPToolHandler {
	return &HTTPToolHandler{
		endpoint: endpoint,
		backend:  backend,
		logger:   logger,
		client: &http.Client{
			Timeout: endpoint.ResponseTimeout,
		},
	}
}

// CreateMCPTool creates an MCP tool from endpoint configuration
func (h *HTTPToolHandler) CreateMCPTool() mcp.Tool {
	var toolOptions []mcp.ToolOption
	toolOptions = append(toolOptions, mcp.WithDescription(h.endpoint.Description))

	// Add parameters based on endpoint configuration
	for _, param := range h.endpoint.BodyParams {
		toolOptions = append(toolOptions, h.createParameterOption(param))
	}
	for _, param := range h.endpoint.QueryParameters {
		toolOptions = append(toolOptions, h.createParameterOption(param))
	}
	for _, param := range h.endpoint.PathParameters {
		toolOptions = append(toolOptions, h.createParameterOption(param))
	}

	return mcp.NewTool(h.endpoint.Name, toolOptions...)
}

// createParameterOption creates a parameter option for the MCP tool based on data type
func (h *HTTPToolHandler) createParameterOption(param *Param) mcp.ToolOption {
	var propertyOptions []mcp.PropertyOption
	propertyOptions = append(propertyOptions, mcp.Description(param.Description))
	if param.Required {
		propertyOptions = append(propertyOptions, mcp.Required())
	}

	switch strings.ToLower(string(param.DataType)) {
	case "string":
		return mcp.WithString(param.Identifier, propertyOptions...)
	case "number":
		return mcp.WithNumber(param.Identifier, propertyOptions...)
	case "boolean":
		return mcp.WithBoolean(param.Identifier, propertyOptions...)
	case "object":
		return mcp.WithObject(param.Identifier, propertyOptions...)
	case "array":
		return mcp.WithArray(param.Identifier, propertyOptions...)
	default:
		// Default to string if type is unknown
		return mcp.WithString(param.Identifier, propertyOptions...)
	}
}

// Handler executes the tool by making an HTTP request
func (h *HTTPToolHandler) Handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := req.GetArguments()

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

	h.logger.Debug("Making HTTP request for tool",
		"tool", h.endpoint.Name,
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
func (h *HTTPToolHandler) buildURL(arguments map[string]any) (string, error) {
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
func (h *HTTPToolHandler) buildQueryParams(arguments map[string]any) string {
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
func (h *HTTPToolHandler) buildRequestBody(arguments map[string]any) ([]byte, error) {
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
func (h *HTTPToolHandler) addHeaders(req *http.Request, arguments map[string]any) {
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
			// This is a simplified implementation - in practice you might want more sophisticated mapping
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

// handleResponse processes the HTTP response and returns MCP result
func (h *HTTPToolHandler) handleResponse(resp *http.Response) (*mcp.CallToolResult, error) {
	// Read response body
	var responseBody bytes.Buffer
	if _, err := responseBody.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	responseText := responseBody.String()

	// Check if the request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.logger.Debug("Tool execution successful",
			"tool", h.endpoint.Name,
			"status", resp.StatusCode,
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Tool '%s' executed successfully. Response: %s", h.endpoint.Name, responseText),
				},
			},
		}, nil
	} else {
		h.logger.Error("Tool execution failed",
			"tool", h.endpoint.Name,
			"status", resp.StatusCode,
			"response", responseText,
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Tool '%s' failed with status %d: %s", h.endpoint.Name, resp.StatusCode, responseText),
				},
			},
			IsError: true,
		}, nil
	}
}
