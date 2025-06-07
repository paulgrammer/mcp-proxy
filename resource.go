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

// HTTPResourceHandler handles resource requests by making HTTP requests
type HTTPResourceHandler struct {
	endpoint      *Endpoint
	backend       *Backend
	logger        *slog.Logger
	clientManager *ClientManager
}

// NewHTTPResourceHandler creates a new HTTP resource handler
func NewHTTPResourceHandler(endpoint *Endpoint, backend *Backend, logger *slog.Logger, clientManager *ClientManager) *HTTPResourceHandler {
	return &HTTPResourceHandler{
		endpoint:      endpoint,
		backend:       backend,
		logger:        logger,
		clientManager: clientManager,
	}
}

// CreateMCPResource creates an MCP resource from endpoint configuration
func (h *HTTPResourceHandler) CreateMCPResource() mcp.Resource {
	return mcp.NewResource(
		h.generateResourceURI(),
		h.endpoint.Name,
		mcp.WithMIMEType("application/json"),
		mcp.WithResourceDescription(h.endpoint.Description),
	)
}

// CreateMCPResourceTemplate creates an MCP resource template if the resource has path parameters
func (h *HTTPResourceHandler) CreateMCPResourceTemplate() *mcp.ResourceTemplate {
	if len(h.endpoint.PathParameters) == 0 {
		return nil // No template needed for static resources
	}

	template := mcp.NewResourceTemplate(
		h.generateResourceURITemplate(),
		h.endpoint.Name,
	)

	return &template
}

// generateResourceURI creates a URI for the resource
func (h *HTTPResourceHandler) generateResourceURI() string {
	return fmt.Sprintf("proxy://%s", h.endpoint.Name)
}

// generateResourceURITemplate creates a URI template for dynamic resources
func (h *HTTPResourceHandler) generateResourceURITemplate() string {
	uri := fmt.Sprintf("proxy://%s", h.endpoint.Name)

	// Add path parameters to the URI template
	if len(h.endpoint.PathParameters) > 0 {
		var params []string
		for _, param := range h.endpoint.PathParameters {
			params = append(params, fmt.Sprintf("{%s}", param.Identifier))
		}
		uri += "/" + strings.Join(params, "/")
	}

	return uri
}

// Handler handles resource read requests
func (h *HTTPResourceHandler) Handler(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract parameters from URI for dynamic resources
	arguments := h.extractArgumentsFromURI(req.Params.URI)

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

	h.logger.Debug("Making HTTP request for resource",
		"resource", h.endpoint.Name,
		"method", h.endpoint.Method,
		"url", url,
	)

	// Make the HTTP request using client manager
	resp, err := h.clientManager.DoRequest(ctx, httpReq, h.endpoint.Name)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle response
	return h.handleResponse(resp, req.Params.URI)
}

// extractArgumentsFromURI extracts parameters from the resource URI
func (h *HTTPResourceHandler) extractArgumentsFromURI(uri string) map[string]any {
	arguments := make(map[string]any)

	// Simple URI parsing - in a real implementation you might want more sophisticated parsing
	// For template URIs like "proxy://resource/{param1}/{param2}"
	// This is a simplified implementation
	parts := strings.Split(uri, "/")
	if len(parts) > 2 {
		// Extract path parameters based on template structure
		templateParts := strings.Split(h.generateResourceURITemplate(), "/")
		for i, part := range parts {
			if i < len(templateParts) && strings.HasPrefix(templateParts[i], "{") && strings.HasSuffix(templateParts[i], "}") {
				paramName := strings.Trim(templateParts[i], "{}")
				arguments[paramName] = part
			}
		}
	}

	return arguments
}

// buildURL constructs the full URL with path parameters substituted
func (h *HTTPResourceHandler) buildURL(arguments map[string]any) (string, error) {
	url := h.backend.BaseURL + h.endpoint.Path

	// Replace path parameters
	for _, param := range h.endpoint.PathParameters {
		var value any
		var exists bool

		if param.ValueType == CONSTANT {
			// Use the predefined value for constant parameters
			value = param.Value
			exists = param.Value != ""
		} else {
			// Use the value from arguments for dynamic parameters
			value, exists = arguments[param.Identifier]
		}

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
func (h *HTTPResourceHandler) buildQueryParams(arguments map[string]any) string {
	var params []string

	for _, param := range h.endpoint.QueryParameters {
		var value any
		var exists bool

		if param.ValueType == CONSTANT {
			// Use the predefined value for constant parameters
			value = param.Value
			exists = param.Value != ""
		} else {
			// Use the value from arguments for dynamic parameters
			value, exists = arguments[param.Identifier]
		}

		if exists {
			params = append(params, fmt.Sprintf("%s=%v", param.Identifier, value))
		}
	}

	return strings.Join(params, "&")
}

// buildRequestBody constructs the JSON request body
func (h *HTTPResourceHandler) buildRequestBody(arguments map[string]any) ([]byte, error) {
	if len(h.endpoint.BodyParams) == 0 {
		return nil, nil
	}

	body := make(map[string]any)
	for _, param := range h.endpoint.BodyParams {
		var value any
		var exists bool

		if param.ValueType == CONSTANT {
			// Use the predefined value for constant parameters
			value = param.Value
			exists = param.Value != ""
		} else {
			// Use the value from arguments for dynamic parameters
			value, exists = arguments[param.Identifier]
		}

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
func (h *HTTPResourceHandler) addHeaders(req *http.Request, arguments map[string]any) {
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

// handleResponse processes the HTTP response and returns MCP resource contents
func (h *HTTPResourceHandler) handleResponse(resp *http.Response, uri string) ([]mcp.ResourceContents, error) {
	// Read response body
	var responseBody bytes.Buffer
	if _, err := responseBody.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	responseText := responseBody.String()

	// Check if the request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.logger.Debug("Resource request successful",
			"resource", h.endpoint.Name,
			"status", resp.StatusCode,
		)

		// Try to determine if response is JSON
		var jsonData interface{}
		if json.Unmarshal(responseBody.Bytes(), &jsonData) == nil {
			// Response is valid JSON, return as JSON
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "application/json",
					Text:     responseText,
				},
			}, nil
		} else {
			// Response is not JSON, return as plain text
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/plain",
					Text:     responseText,
				},
			}, nil
		}
	} else {
		h.logger.Error("Resource request failed",
			"resource", h.endpoint.Name,
			"status", resp.StatusCode,
			"response", responseText,
		)

		return nil, fmt.Errorf("resource request failed with status %d: %s", resp.StatusCode, responseText)
	}
}
