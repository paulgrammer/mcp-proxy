package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
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
	config        config
	logger        *slog.Logger
	clientManager *ClientManager

	tools     []server.ServerTool
	prompts   []server.ServerPrompt
	resources []server.ServerResource

	transport transport.Interface
	client    *client.Client

	wg         sync.WaitGroup
	configFile string  // Path to the configuration file
	mcpConfig  *Config // Current configuration
}

// NewServer creates a new MCP server with the given options.
func NewServer(opts ...Option) (*Proxy, error) {
	server := &Proxy{
		config: config{
			Name:    "mpc-proxy",
			Addr:    ":8888",
			BaseURL: "",
		},
		logger:        slog.Default(),
		clientManager: NewClientManager(),
	}

	// Apply options
	for _, opt := range opts {
		opt(server)
	}

	return server, nil
}

// NewServerFromConfig creates a new MCP server from configuration
func NewServerFromConfig(cfg *Config, opts ...Option) (*Proxy, error) {
	server := &Proxy{
		config: config{
			Name:    cfg.MCP.ServerName,
			Addr:    ":8888",
			BaseURL: "",
		},
		logger:        slog.Default(),
		clientManager: NewClientManager(),
		mcpConfig:     cfg,
	}

	// Apply options
	for _, opt := range opts {
		opt(server)
	}

	// Setup endpoints from configuration
	if err := server.setupEndpointsFromConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to setup endpoints: %w", err)
	}

	return server, nil
}

// NewServerFromConfigFile creates a new MCP server from configuration file
func NewServerFromConfigFile(configFile string, opts ...Option) (*Proxy, error) {
	cfg, err := ParseConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	server := &Proxy{
		config: config{
			Name:    cfg.MCP.ServerName,
			Addr:    ":8888",
			BaseURL: "",
		},
		logger:        slog.Default(),
		clientManager: NewClientManager(),
		configFile:    configFile,
		mcpConfig:     cfg,
	}

	// Apply options
	for _, opt := range opts {
		opt(server)
	}

	// Setup endpoints from configuration
	if err := server.setupEndpointsFromConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to setup endpoints: %w", err)
	}

	return server, nil
}

// setupEndpointsFromConfig configures MCP endpoints from the config
func (s *Proxy) setupEndpointsFromConfig(cfg *Config) error {
	for _, backend := range cfg.Backends {
		if err := s.setupBackendEndpoints(backend); err != nil {
			return fmt.Errorf("failed to setup backend endpoints: %w", err)
		}
	}
	return nil
}

// setupBackendEndpoints sets up all endpoints for a backend
func (s *Proxy) setupBackendEndpoints(backend *Backend) error {
	for _, endpoint := range backend.Endpoints {
		switch endpoint.Capability {
		case TOOL:
			if err := s.setupToolEndpoint(&endpoint, backend); err != nil {
				return fmt.Errorf("failed to setup tool endpoint '%s': %w", endpoint.Name, err)
			}
		case RESOURCE:
			if err := s.setupResourceEndpoint(&endpoint, backend); err != nil {
				return fmt.Errorf("failed to setup resource endpoint '%s': %w", endpoint.Name, err)
			}
		case PROMPT:
			if err := s.setupPromptEndpoint(&endpoint, backend); err != nil {
				return fmt.Errorf("failed to setup prompt endpoint '%s': %w", endpoint.Name, err)
			}
		default:
			return fmt.Errorf("unknown capability '%s' for endpoint '%s'", endpoint.Capability, endpoint.Name)
		}
	}
	return nil
}

// setupToolEndpoint sets up a tool endpoint
func (s *Proxy) setupToolEndpoint(endpoint *Endpoint, backend *Backend) error {
	// Set default timeout if not specified
	if endpoint.ResponseTimeout == 0 {
		endpoint.ResponseTimeout = Duration(30 * time.Second)
	}

	handler := NewHTTPToolHandler(endpoint, backend, s.logger, s.clientManager)
	tool := handler.CreateMCPTool()

	s.AddTool(tool, handler.Handler)

	s.logger.Info("Added tool endpoint",
		"name", endpoint.Name,
		"mode", endpoint.Mode,
		"path", endpoint.Path,
		"method", endpoint.Method,
	)

	return nil
}

// setupResourceEndpoint sets up a resource endpoint
func (s *Proxy) setupResourceEndpoint(endpoint *Endpoint, backend *Backend) error {
	// Set default timeout if not specified
	if endpoint.ResponseTimeout == 0 {
		endpoint.ResponseTimeout = Duration(30 * time.Second)
	}

	handler := NewHTTPResourceHandler(endpoint, backend, s.logger, s.clientManager)

	// Check if this is a dynamic resource (has path parameters)
	if resourceTemplate := handler.CreateMCPResourceTemplate(); resourceTemplate != nil {
		// Add as resource template for dynamic resources
		s.AddResourceTemplate(*resourceTemplate, handler.Handler)
		s.logger.Info("Added resource template endpoint",
			"name", endpoint.Name,
			"template", resourceTemplate.URITemplate,
			"path", endpoint.Path,
			"method", endpoint.Method,
		)
	} else {
		// Add as static resource
		resource := handler.CreateMCPResource()
		s.AddResource(resource, handler.Handler)
		s.logger.Info("Added resource endpoint",
			"name", endpoint.Name,
			"uri", resource.URI,
			"path", endpoint.Path,
			"method", endpoint.Method,
		)
	}

	return nil
}

// setupPromptEndpoint sets up a prompt endpoint
func (s *Proxy) setupPromptEndpoint(endpoint *Endpoint, backend *Backend) error {
	// Set default timeout if not specified
	if endpoint.ResponseTimeout == 0 {
		endpoint.ResponseTimeout = Duration(30 * time.Second)
	}

	handler := NewHTTPPromptHandler(endpoint, backend, s.logger, s.clientManager)
	prompt := handler.CreateMCPPrompt()

	s.AddPrompt(prompt, handler.Handler)

	s.logger.Info("Added prompt endpoint",
		"name", endpoint.Name,
		"path", endpoint.Path,
		"method", endpoint.Method,
	)

	return nil
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

// AddResourceTemplate adds a resource template to an server.
func (s *Proxy) AddResourceTemplate(template mcp.ResourceTemplate, handler server.ResourceHandlerFunc) {
	// Convert template to resource for the server
	// The MCP server handles templates internally
	var uriTemplate string
	if template.URITemplate != nil {
		uriTemplate = template.URITemplate.Template.Raw()
	}

	resource := mcp.Resource{
		URI:         uriTemplate,
		Name:        template.Name,
		Description: template.Description,
		MIMEType:    template.MIMEType,
	}
	s.resources = append(s.resources, server.ServerResource{
		Resource: resource,
		Handler:  handler,
	})
}

// configAPIHandler handles configuration API requests
func (s *Proxy) configAPIHandler() http.Handler {
	mux := http.NewServeMux()

	// Enable CORS for all config endpoints
	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			h(w, r)
		}
	}

	// /api/config - Handle GET and PUT requests for configuration
	mux.HandleFunc("/api/config", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if s.mcpConfig == nil {
				http.Error(w, "No configuration available", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(s.mcpConfig); err != nil {
				s.logger.Error("Failed to encode config", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			fmt.Println(">?>>>>>>>>>>>>>>>>>>", string(body))
			var newConfig Config
			if err := json.Unmarshal(body, &newConfig); err != nil {
				http.Error(w, fmt.Sprintf("Invalid JSON: %s", err.Error()), http.StatusBadRequest)
				return
			}

			// Validate the new configuration
			if err := validateParsedConfig(&newConfig); err != nil {
				http.Error(w, fmt.Sprintf("Configuration validation failed: %v", err), http.StatusBadRequest)
				return
			}

			// Set defaults
			if err := setConfigDefaults(&newConfig); err != nil {
				http.Error(w, fmt.Sprintf("Failed to set defaults: %v", err), http.StatusInternalServerError)
				return
			}

			// Post-process the configuration
			if err := postProcessParsedConfig(&newConfig); err != nil {
				http.Error(w, fmt.Sprintf("Failed to post-process config: %v", err), http.StatusInternalServerError)
				return
			}

			// Save to file if configFile is set
			if s.configFile != "" {
				yamlData, err := yaml.Marshal(&newConfig)
				if err != nil {
					http.Error(w, "Failed to marshal config to YAML", http.StatusInternalServerError)
					return
				}

				if err := os.WriteFile(s.configFile, yamlData, 0644); err != nil {
					s.logger.Error("Failed to write config file", "error", err, "file", s.configFile)
					http.Error(w, "Failed to save configuration file", http.StatusInternalServerError)
					return
				}
			}

			// Update the current configuration
			s.mcpConfig = &newConfig

			s.logger.Info("Configuration updated successfully")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Configuration updated successfully"})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	return mux
}

// Start starts the server in a goroutine. Make sure to defer Close() after Start().
// When using NewServer(), the returned server is already started.
func (s *Proxy) Start(ctx context.Context) error {
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
		webHandler := webHandler()
		configAPI := s.configAPIHandler()
		mux.Handle("/sse", sseServer.SSEHandler())
		mux.Handle("/message", sseServer.MessageHandler())
		mux.Handle("/api/", configAPI)
		mux.Handle("/config/", webHandler)
		mux.Handle("/assets/", webHandler)

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
		<-ctx.Done()
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

	if err := s.transport.Start(ctx); err != nil {
		return fmt.Errorf("transport.Start(): %w", err)
	}

	s.client = client.NewClient(s.transport)

	var initReq mcp.InitializeRequest
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	if _, err := s.client.Initialize(ctx, initReq); err != nil {
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

	// Wait for server goroutine to finish
	s.wg.Wait()
}

// Client returns an MCP client connected to the server.
// The client is already initialized, i.e. you do _not_ need to call Client.Initialize().
func (s *Proxy) Client() *client.Client {
	return s.client
}
