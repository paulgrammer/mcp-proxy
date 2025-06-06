package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Option is a function that configures the server
type Option func(*Proxy)

// WithName sets the server name
func WithName(name string) Option {
	return func(s *Proxy) {
		s.config.Name = name
	}
}

// WithAddr sets the server address
func WithAddr(addr string) Option {
	return func(s *Proxy) {
		s.config.Addr = addr
	}
}

// WithBaseURL sets the server base URL
func WithBaseURL(baseURL string) Option {
	return func(s *Proxy) {
		s.config.BaseURL = baseURL
	}
}

// WithLogger sets the server logger
func WithLogger(logger *slog.Logger) Option {
	return func(s *Proxy) {
		s.logger = logger
	}
}

// config holds server configuration
type config struct {
	Name    string
	Addr    string
	BaseURL string
}

// Proxy encapsulates an MCP server and manages resources like pipes and context.
type Proxy struct {
	config config
	logger *slog.Logger

	tools     []server.ServerTool
	prompts   []server.ServerPrompt
	resources []server.ServerResource

	ctx    context.Context
	cancel func()

	transport transport.Interface
	client    *client.Client

	wg sync.WaitGroup
}

// NewServer creates a new MCP server with the given options.
func NewServer(ctx context.Context, opts ...Option) (*Proxy, error) {
	server := &Proxy{
		config: config{
			Name:    "telephony-apps",
			Addr:    ":8888",
			BaseURL: "",
		},
		logger: slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(server)
	}

	// Set up context with cancellation, used to stop the server
	server.ctx, server.cancel = context.WithCancel(ctx)

	return server, nil
}

// AddTools adds multiple tools to an server.
func (s *Proxy) AddTools(tools ...server.ServerTool) {
	s.tools = append(s.tools, tools...)
}

// AddTool adds a tool to an server.
func (s *Proxy) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.tools = append(s.tools, server.ServerTool{
		Tool:    tool,
		Handler: handler,
	})
}

// AddPrompt adds a prompt to an server.
func (s *Proxy) AddPrompt(prompt mcp.Prompt, handler server.PromptHandlerFunc) {
	s.prompts = append(s.prompts, server.ServerPrompt{
		Prompt:  prompt,
		Handler: handler,
	})
}

// AddPrompts adds multiple prompts to an server.
func (s *Proxy) AddPrompts(prompts ...server.ServerPrompt) {
	s.prompts = append(s.prompts, prompts...)
}

// AddResource adds a resource to an server.
func (s *Proxy) AddResource(resource mcp.Resource, handler server.ResourceHandlerFunc) {
	s.resources = append(s.resources, server.ServerResource{
		Resource: resource,
		Handler:  handler,
	})
}

// AddResources adds multiple resources to an server.
func (s *Proxy) AddResources(resources ...server.ServerResource) {
	s.resources = append(s.resources, resources...)
}

// Start starts the server in a goroutine. Make sure to defer Close() after Start().
// When using NewServer(), the returned server is already started.
func (s *Proxy) Start() error {
	s.wg.Add(1)

	addr := s.config.Addr
	baseURL := s.config.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost%s", addr)
	}
	hooks := newServerHooks(s.logger)

	// Start the MCP server in a goroutine
	go func() {
		defer s.wg.Done()

		mcpServer := server.NewMCPServer(
			s.config.Name, "1.0.0",
			server.WithResourceCapabilities(true, true),
			server.WithPromptCapabilities(true),
			server.WithToolCapabilities(true),
			server.WithLogging(),
			server.WithHooks(hooks),
		)

		mcpServer.AddTools(s.tools...)
		mcpServer.AddPrompts(s.prompts...)
		mcpServer.AddResources(s.resources...)

		sseServer := server.NewSSEServer(mcpServer,
			server.WithBaseURL(baseURL),
			server.WithUseFullURLForMessageEndpoint(true),
		)

		mux := http.NewServeMux()
		mux.Handle("/sse", sseServer.SSEHandler())
		mux.Handle("/message", sseServer.MessageHandler())

		httpServer := &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		s.logger.Info("MCP SSE server listening", "addr", addr)

		// Start HTTP server in a goroutine
		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.logger.Error("MCP Proxy error", "error", err)
			}
		}()

		// Wait for context cancellation to shutdown server
		<-s.ctx.Done()
		s.logger.Info("Shutting down HTTP server...")

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Failed to shutdown HTTP server gracefully", "error", err)
		} else {
			s.logger.Info("HTTP server shutdown successfully")
		}
	}()

	transport, err := transport.NewSSE(fmt.Sprintf("%s/sse", baseURL))
	if err != nil {
		return fmt.Errorf("transport.NewSSE(): %w", err)
	}

	s.transport = transport

	if err := s.transport.Start(s.ctx); err != nil {
		return fmt.Errorf("transport.Start(): %w", err)
	}

	s.client = client.NewClient(s.transport)

	var initReq mcp.InitializeRequest
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	if _, err := s.client.Initialize(s.ctx, initReq); err != nil {
		return fmt.Errorf("client.Initialize(): %w", err)
	}

	return nil
}

// Close stops the server and cleans up resources like temporary directories.
func (s *Proxy) Close() {
	if s.transport != nil {
		s.transport.Close()
		s.transport = nil
		s.client = nil
	}

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	// Wait for server goroutine to finish
	s.wg.Wait()
}

// Client returns an MCP client connected to the server.
// The client is already initialized, i.e. you do _not_ need to call Client.Initialize().
func (s *Proxy) Client() *client.Client {
	return s.client
}
