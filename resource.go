package proxy

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
)

type Resource struct{}

func (r *Resource) ResourceTemplate() mcp.ResourceTemplate {
	panic("not implemented")
}

func (r *Resource) Handler(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{}, errors.New("not implemented")
}
