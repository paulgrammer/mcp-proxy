package proxy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolResponse represents the  response
type ToolResponse struct{}

// Tool provides functionality
type Tool struct {
	name        string
	description string
}

// NewTool creates a new instance of Tool
func NewTool(name string) *Tool {
	return &Tool{
		name:        "",
		description: "Description here",
	}
}

// Name returns the tool's unique identifier
func (t *Tool) Name() string {
	if t.name != "" {
		return t.name
	}

	return "undefined_tool"
}

// Description returns the tool's description
func (t *Tool) Description() string {
	return t.description
}

// Tool creates and returns the MCP tool configuration
func (t *Tool) Tool() mcp.Tool {
	return mcp.NewTool(
		t.Name(),
		mcp.WithDescription(t.description),
	)
}

// Handler returns the tool's execution handler
func (t *Tool) Handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	response := &ToolResponse{}

	result, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool response: %w", err)
	}

	return mcp.NewToolResultText(string(result)), nil
}
