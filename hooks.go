package proxy

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func newServerHooks(logger *slog.Logger) *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		logger.Debug("beforeAny", "method", method, "id", id, "message", message)
	})

	hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		logger.Debug("onSuccess", "method", method, "id", id, "message", message, "result", result)
	})

	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		logger.Error("onError", "method", method, "id", id, "message", message, "error", err)
	})

	hooks.AddBeforeInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest) {
		logger.Info("beforeInitialize", "id", id, "message", message)
	})

	hooks.AddOnRequestInitialization(func(ctx context.Context, id any, message any) error {
		logger.Info("onRequestInitialization", "id", id, "message", message)
		// authorization verification and other preprocessing tasks are performed.
		return nil
	})

	hooks.AddAfterInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		logger.Info("afterInitialize", "id", id, "message", message, "result", result)
	})

	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		logger.Debug("afterCallTool", "id", id, "message", message, "result", result)
	})

	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		logger.Debug("beforeCallTool", "id", id, "message", message)
	})

	return hooks
}
