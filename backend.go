package proxy

// Backend defines the target HTTP backend configuration
type Backend struct {
	// BaseURL is the base URL for all endpoints in this backend
	BaseURL string `json:"base_url" yaml:"base_url"`

	// DefaultHeaders are headers that will be included in all requests to this backend
	// Individual endpoint headers will be merged with these defaults
	// Common uses: authentication tokens, API keys, content-type specifications
	DefaultHeaders []*Header `json:"default_headers" yaml:"default_headers"`

	// Endpoints defines all the MCP endpoints for this backend
	// Each endpoint will use this backend's BaseURL and DefaultHeaders
	Endpoints []Endpoint `json:"endpoints" yaml:"endpoints"`
}
