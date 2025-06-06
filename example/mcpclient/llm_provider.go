package client

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// ProviderType represents different LLM providers
type ProviderType string

const (
	ProviderAnthropic ProviderType = "anthropic"
	ProviderOpenAI    ProviderType = "openai"
	ProviderOllama    ProviderType = "ollama"
	ProviderLocal     ProviderType = "local"
)

// ProviderConfig holds configuration for LLM providers
type ProviderConfig struct {
	Type         ProviderType
	APIKey       string
	BaseURL      string
	Model        string
	SystemPrompt string
}

// ProviderFactory creates LLM providers based on configuration
type ProviderFactory struct {
	logger *slog.Logger
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(logger *slog.Logger) *ProviderFactory {
	return &ProviderFactory{
		logger: logger,
	}
}

// CreateProvider creates an LLM provider based on the configuration
func (f *ProviderFactory) CreateProvider(config ProviderConfig) (LLMProvider, error) {
	f.logger.Info("Creating LLM provider", "type", config.Type, "model", config.Model)

	switch config.Type {
	case ProviderAnthropic:
		return f.createAnthropicProvider(config)
	case ProviderOpenAI:
		return f.createOpenAIProvider(config)
	case ProviderOllama:
		return f.createOllamaProvider(config)
	case ProviderLocal:
		return f.createLocalProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// CreateProviderFromEnv creates a provider based on environment variables
func (f *ProviderFactory) CreateProviderFromEnv() (LLMProvider, error) {
	// Check for explicit provider configuration first
	providerEnv := os.Getenv("LLM_PROVIDER")
	if providerEnv != "" {
		providerType, err := GetProviderFromString(providerEnv)
		if err != nil {
			return nil, fmt.Errorf("invalid LLM_PROVIDER value '%s': %w", providerEnv, err)
		}
		
		config, err := f.getConfigForProvider(providerType)
		if err != nil {
			return nil, err
		}
		
		return f.CreateProvider(config)
	}

	// Fallback: auto-detect based on available environment variables
	f.logger.Info("LLM_PROVIDER not set, auto-detecting from available environment variables")
	
	// Check which provider is configured via environment variables
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		config := ProviderConfig{
			Type:   ProviderAnthropic,
			APIKey: apiKey,
			Model:  getEnvOrDefault("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),
		}
		return f.CreateProvider(config)
	}

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config := ProviderConfig{
			Type:    ProviderOpenAI,
			APIKey:  apiKey,
			BaseURL: getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			Model:   getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
		}
		return f.CreateProvider(config)
	}

	if baseURL := os.Getenv("OLLAMA_BASE_URL"); baseURL != "" {
		config := ProviderConfig{
			Type:    ProviderOllama,
			BaseURL: baseURL,
			Model:   getEnvOrDefault("OLLAMA_MODEL", "llama2"),
		}
		return f.CreateProvider(config)
	}

	if baseURL := os.Getenv("LOCAL_LLM_URL"); baseURL != "" {
		config := ProviderConfig{
			Type:    ProviderLocal,
			BaseURL: baseURL,
			Model:   getEnvOrDefault("LOCAL_LLM_MODEL", "local-model"),
		}
		return f.CreateProvider(config)
	}

	return nil, fmt.Errorf("no LLM provider configured. Please set LLM_PROVIDER environment variable or one of: ANTHROPIC_API_KEY, OPENAI_API_KEY, OLLAMA_BASE_URL, or LOCAL_LLM_URL")
}

// getConfigForProvider creates a config for the specified provider type using environment variables
func (f *ProviderFactory) getConfigForProvider(providerType ProviderType) (ProviderConfig, error) {
	config := ProviderConfig{Type: providerType}
	
	switch providerType {
	case ProviderAnthropic:
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		config.Model = getEnvOrDefault("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022")
		
		if config.APIKey == "" {
			return config, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required for Anthropic provider")
		}
		
	case ProviderOpenAI:
		config.APIKey = os.Getenv("OPENAI_API_KEY")
		config.BaseURL = getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
		config.Model = getEnvOrDefault("OPENAI_MODEL", "gpt-4o")
		
		if config.APIKey == "" {
			return config, fmt.Errorf("OPENAI_API_KEY environment variable is required for OpenAI provider")
		}
		
	case ProviderOllama:
		config.BaseURL = os.Getenv("OLLAMA_BASE_URL")
		config.Model = getEnvOrDefault("OLLAMA_MODEL", "llama2")
		
		if config.BaseURL == "" {
			return config, fmt.Errorf("OLLAMA_BASE_URL environment variable is required for Ollama provider")
		}
		
	case ProviderLocal:
		config.BaseURL = os.Getenv("LOCAL_LLM_URL")
		config.Model = getEnvOrDefault("LOCAL_LLM_MODEL", "local-model")
		
		if config.BaseURL == "" {
			return config, fmt.Errorf("LOCAL_LLM_URL environment variable is required for local provider")
		}
		
	default:
		return config, fmt.Errorf("unsupported provider type: %s", providerType)
	}
	
	return config, nil
}

func (f *ProviderFactory) createAnthropicProvider(config ProviderConfig) (LLMProvider, error) {
	provider, err := NewAnthropicProvider(config.APIKey, f.logger)
	if err != nil {
		return nil, err
	}

	if config.Model != "" {
		provider.SetModel(config.Model)
	}

	if config.SystemPrompt != "" {
		provider.SetSystemPrompt(config.SystemPrompt)
	}

	return provider, nil
}

func (f *ProviderFactory) createOpenAIProvider(config ProviderConfig) (LLMProvider, error) {
	provider, err := NewOpenAIProvider(config.APIKey, f.logger)
	if err != nil {
		return nil, err
	}

	if config.Model != "" {
		provider.SetModel(config.Model)
	}

	if config.BaseURL != "" {
		provider.SetBaseURL(config.BaseURL)
	}

	if config.SystemPrompt != "" {
		provider.SetSystemPrompt(config.SystemPrompt)
	}

	return provider, nil
}

func (f *ProviderFactory) createOllamaProvider(config ProviderConfig) (LLMProvider, error) {
	// Placeholder for Ollama provider
	// This would implement the LLMProvider interface for Ollama
	return nil, fmt.Errorf("Ollama provider not yet implemented")
}

func (f *ProviderFactory) createLocalProvider(config ProviderConfig) (LLMProvider, error) {
	// Placeholder for local/custom LLM provider
	// This would implement the LLMProvider interface for local models
	return nil, fmt.Errorf("Local provider not yet implemented")
}

// GetAvailableProviders returns a list of available provider types
func (f *ProviderFactory) GetAvailableProviders() []ProviderType {
	return []ProviderType{
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderOllama,
		ProviderLocal,
	}
}

// GetProviderFromString converts a string to ProviderType
func GetProviderFromString(provider string) (ProviderType, error) {
	switch strings.ToLower(provider) {
	case "anthropic", "claude":
		return ProviderAnthropic, nil
	case "openai", "gpt":
		return ProviderOpenAI, nil
	case "ollama":
		return ProviderOllama, nil
	case "local":
		return ProviderLocal, nil
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}
}

// GetConfigFromFlags creates a provider config from command line flags or prompts
func (f *ProviderFactory) GetConfigFromFlags(providerStr, model, apiKey, baseURL string) (ProviderConfig, error) {
	return f.GetConfigFromFlagsWithSystem(providerStr, model, apiKey, baseURL, "")
}

// GetConfigFromFlagsWithSystem creates a provider config from command line flags with system prompt
func (f *ProviderFactory) GetConfigFromFlagsWithSystem(providerStr, model, apiKey, baseURL, systemPrompt string) (ProviderConfig, error) {
	providerType, err := GetProviderFromString(providerStr)
	if err != nil {
		return ProviderConfig{}, err
	}

	config := ProviderConfig{
		Type:         providerType,
		Model:        model,
		APIKey:       apiKey,
		BaseURL:      baseURL,
		SystemPrompt: systemPrompt,
	}

	// Set defaults based on provider type
	switch providerType {
	case ProviderAnthropic:
		if config.APIKey == "" {
			config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if config.Model == "" {
			config.Model = "claude-3-5-sonnet-20241022"
		}

	case ProviderOpenAI:
		if config.APIKey == "" {
			config.APIKey = os.Getenv("OPENAI_API_KEY")
		}
		if config.Model == "" {
			config.Model = "gpt-4o"
		}
		if config.BaseURL == "" {
			config.BaseURL = "https://api.openai.com/v1"
		}

	case ProviderOllama:
		if config.BaseURL == "" {
			config.BaseURL = "http://localhost:11434"
		}
		if config.Model == "" {
			config.Model = "llama2"
		}

	case ProviderLocal:
		if config.BaseURL == "" {
			config.BaseURL = "http://localhost:8080"
		}
		if config.Model == "" {
			config.Model = "local-model"
		}
	}

	return config, nil
}

// ValidateConfig validates a provider configuration
func (f *ProviderFactory) ValidateConfig(config ProviderConfig) error {
	switch config.Type {
	case ProviderAnthropic, ProviderOpenAI:
		if config.APIKey == "" {
			return fmt.Errorf("%s provider requires an API key", config.Type)
		}
	case ProviderOllama, ProviderLocal:
		if config.BaseURL == "" {
			return fmt.Errorf("%s provider requires a base URL", config.Type)
		}
	}

	if config.Model == "" {
		return fmt.Errorf("model is required for all providers")
	}

	return nil
}
