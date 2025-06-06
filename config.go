package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete MCP HTTP proxy configuration
type Config struct {
	// MCP configuration
	MCP *MCPConfig `json:"mcp" yaml:"mcp"`

	// Backends configuration (multiple backends for multi-backend mode)
	Backends []*Backend `json:"backends,omitempty" yaml:"backends,omitempty"`
}

// MCPConfig defines MCP-specific settings
type MCPConfig struct {
	// ServerName for MCP identification
	ServerName string `json:"server_name" yaml:"server_name" default:"MCP HTTP Proxy"`

	// Version of the MCP server
	Version string `json:"version" yaml:"version" default:"1.0.0"`
}

func ParseConfig(filename string) (*Config, error) {
	// Expand path to handle environment variables and home directory
	expandedPath := expandPath(filename)

	// Read the YAML file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", expandedPath, err)
	}

	// Unmarshal YAML into Config struct
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Set defaults if needed
	if err := setConfigDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("failed to set config defaults: %w", err)
	}

	// Validate the configuration
	if err := validateParsedConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Post-process the configuration
	if err := postProcessParsedConfig(&cfg); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	return &cfg, nil
}

// ParseConfigFromBytes parses configuration from byte data
func ParseConfigFromBytes(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Set defaults if needed
	if err := setConfigDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("failed to set config defaults: %w", err)
	}

	// Validate the configuration
	if err := validateParsedConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Post-process the configuration
	if err := postProcessParsedConfig(&cfg); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	return &cfg, nil
}

// ParseConfigWithValidation parses config with optional validation
func ParseConfigWithValidation(filename string, validate bool) (*Config, error) {
	// Expand path to handle environment variables and home directory
	expandedPath := expandPath(filename)

	// Read the YAML file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", expandedPath, err)
	}

	// Unmarshal YAML into Config struct
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Set defaults
	if err := setConfigDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("failed to set config defaults: %w", err)
	}

	// Optional validation
	if validate {
		if err := validateParsedConfig(&cfg); err != nil {
			return nil, fmt.Errorf("config validation failed: %w", err)
		}
	}

	// Post-process the configuration
	if err := postProcessParsedConfig(&cfg); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	return &cfg, nil
}

// setConfigDefaults sets default values for the configuration
func setConfigDefaults(cfg *Config) error {
	// Set MCP defaults if MCP config is nil
	if cfg.MCP == nil {
		cfg.MCP = &MCPConfig{
			ServerName: "MCP HTTP Proxy",
			Version:    "1.0.0",
		}
	} else {
		// Set individual MCP defaults
		if cfg.MCP.ServerName == "" {
			cfg.MCP.ServerName = "MCP HTTP Proxy"
		}
		if cfg.MCP.Version == "" {
			cfg.MCP.Version = "1.0.0"
		}
	}

	return nil
}

// validateParsedConfig validates the parsed configuration
func validateParsedConfig(cfg *Config) error {
	// Validate MCP configuration
	if cfg.MCP == nil {
		return fmt.Errorf("MCP configuration is required")
	}

	// Validate backends
	if len(cfg.Backends) == 0 {
		return fmt.Errorf("at least one backend must be configured")
	}

	// Validate each backend
	for i, backend := range cfg.Backends {
		if err := validateBackend(backend, i); err != nil {
			return fmt.Errorf("backend %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateBackend validates a single backend configuration
func validateBackend(backend *Backend, index int) error {
	// Validate base URL
	if backend.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	// Validate endpoints
	if len(backend.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint must be configured")
	}

	// Validate each endpoint
	endpointNames := make(map[string]bool)
	for j, endpoint := range backend.Endpoints {
		if err := validateEndpoint(endpoint, j); err != nil {
			return fmt.Errorf("endpoint %d validation failed: %w", j, err)
		}

		// Check for duplicate endpoint names
		if endpointNames[endpoint.Name] {
			return fmt.Errorf("duplicate endpoint name '%s'", endpoint.Name)
		}
		endpointNames[endpoint.Name] = true
	}

	return nil
}

// validateEndpoint validates a single endpoint configuration
func validateEndpoint(endpoint Endpoint, index int) error {
	// Validate required fields
	if endpoint.Name == "" {
		return fmt.Errorf("name is required")
	}

	if endpoint.Path == "" {
		return fmt.Errorf("path is required")
	}

	// Validate capability
	validCapabilities := []string{string(TOOL), string(RESOURCE), string(PROMPT)}
	if !slices.Contains(validCapabilities, string(endpoint.Capability)) {
		return fmt.Errorf("invalid capability '%s', must be one of: %s",
			endpoint.Capability, strings.Join(validCapabilities, ", "))
	}

	// Validate mode for TOOL capability
	if endpoint.Capability == TOOL {
		validModes := []string{string(WEBHOOK), string(CLIENT)}
		if !slices.Contains(validModes, string(endpoint.Mode)) {
			return fmt.Errorf("invalid mode '%s' for tool, must be one of: %s",
				endpoint.Mode, strings.Join(validModes, ", "))
		}
	}

	// Validate HTTP method
	validMethods := []string{string(GET), string(POST), string(PUT), string(PATCH), string(DELETE)}
	if !slices.Contains(validMethods, string(endpoint.Method)) {
		return fmt.Errorf("invalid HTTP method '%s'", endpoint.Method)
	}

	return nil
}

// postProcessParsedConfig performs post-processing on the parsed configuration
func postProcessParsedConfig(cfg *Config) error {
	// Process environment variable substitution for all backends
	for _, backend := range cfg.Backends {
		if err := processBackendEnvironmentVars(backend); err != nil {
			return fmt.Errorf("failed to process environment variables: %w", err)
		}
	}

	return nil
}

// processBackendEnvironmentVars processes environment variables in backend configuration
func processBackendEnvironmentVars(backend *Backend) error {
	// Expand environment variables in base URL
	backend.BaseURL = os.ExpandEnv(backend.BaseURL)

	// Expand environment variables in default headers
	for _, header := range backend.DefaultHeaders {
		header.Name = os.ExpandEnv(header.Name)
		header.Value = os.ExpandEnv(header.Value)
	}

	// Process environment variables in endpoints
	for i := range backend.Endpoints {
		processEndpointEnvironmentVars(&backend.Endpoints[i])
	}

	return nil
}

// processEndpointEnvironmentVars processes environment variables in endpoint configuration
func processEndpointEnvironmentVars(endpoint *Endpoint) {
	// Expand environment variables in path
	endpoint.Path = os.ExpandEnv(endpoint.Path)

	// Expand environment variables in description
	endpoint.Description = os.ExpandEnv(endpoint.Description)

	// Process headers
	for _, header := range endpoint.Headers {
		header.Name = os.ExpandEnv(header.Name)
		header.Value = os.ExpandEnv(header.Value)
	}

	// Process parameters (body, query, path)
	for _, param := range endpoint.BodyParams {
		processParamEnvironmentVars(param)
	}
	for _, param := range endpoint.QueryParameters {
		processParamEnvironmentVars(param)
	}
	for _, param := range endpoint.PathParameters {
		processParamEnvironmentVars(param)
	}
}

// processParamEnvironmentVars processes environment variables in parameter configuration
func processParamEnvironmentVars(param *Param) {
	param.Description = os.ExpandEnv(param.Description)
	param.Identifier = os.ExpandEnv(param.Identifier)
	// Note: We don't expand Value field as it's used by the LLM for dynamic extraction
}

// expandPath expands environment variables and home directory in paths
func expandPath(path string) string {
	// Expand environment variables
	expanded := os.ExpandEnv(path)

	// Expand home directory
	if strings.HasPrefix(expanded, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			expanded = filepath.Join(home, expanded[2:])
		}
	}

	return expanded
}
