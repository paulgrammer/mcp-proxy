package proxy

import (
	"net/http"
	"time"
)

// Type aliases for better code readability and type safety
type Method string
type Data string
type Value string
type Mode string
type Capability string

// Capability constants define what kind of MCP Endpoint this proxy represents
const (
	// TOOL entities allow the LLM to perform actions by calling functions
	// Tools are active - they execute operations, send data, trigger workflows
	// Example: create_order, send_email, update_database, call_api
	TOOL Capability = "tool"

	// RESOURCE entities provide data that the LLM can read and reference
	// Resources are passive - they supply information for the LLM to use in responses
	// Example: user_profile, product_catalog, company_policies, documentation
	RESOURCE Capability = "resource"

	// PROMPT entities provide reusable prompt templates that can be invoked
	// Prompts define specific instructions or workflows for the LLM to follow
	// Example: email_template, code_review_checklist, customer_service_script
	PROMPT Capability = "prompt"
)

// ValueType constants define how parameter values are resolved
const (
	// DYNAMIC values are extracted by the LLM from the conversation context
	// Example: user's name, order details, extracted information
	DYNAMIC Value = "dynamic"

	// CONSTANT values are predefined static values
	// Example: API keys, fixed configuration values, service identifiers
	CONSTANT Value = "constant"
)

// HTTP method constants for the proxy requests
// These define what HTTP method will be used when calling the target service
const (
	POST   Method = http.MethodPost   // For creating resources or sending data
	GET    Method = http.MethodGet    // For retrieving information
	PUT    Method = http.MethodPut    // For updating entire resources
	PATCH  Method = http.MethodPatch  // For partial updates
	DELETE Method = http.MethodDelete // For removing resources
	UPDATE Method = "UPDATE"          // Custom method for specific update operations
)

// Mode constants define how webhook/client tools integrate with the MCP client
// Note: Only applies when Capability is TOOL
const (
	// WEBHOOK tool sends extracted data to your HTTP server endpoint
	// The LLM will make an HTTP request to your specified URL with the collected parameters
	// Use this when you want server-side processing of the extracted information
	WEBHOOK Mode = "webhook"

	// CLIENT tool triggers an event on the client side with extracted information
	// The data is sent to the MCP client application rather than an external server
	// Use this for client-side processing, UI updates, or local integrations
	CLIENT Mode = "client"
)

// Header represents HTTP headers that will be included in proxy requests
// These allow you to configure authentication, content types, and other HTTP metadata
type Header struct {
	// Type determines if this header value is dynamic (extracted by LLM) or constant (predefined)
	Type Value `json:"type" yaml:"type"`

	// Name is the HTTP header name (e.g., "Authorization", "Content-Type", "X-API-Key")
	Name string `json:"name" yaml:"name"`

	// Value is the header value - can be a constant string or a template for dynamic extraction
	// For dynamic headers, this might be a description of what to extract
	Value string `json:"value" yaml:"value"`
}

// Param defines a parameter that the LLM should extract from conversations
// These parameters become the data payload sent to your HTTP endpoint
type Param struct {
	// DataType specifies the expected data type (string, number, boolean, object, array, etc.)
	// Helps the LLM understand how to parse and format the extracted value
	DataType Data `json:"data_type" yaml:"data_type"`

	// ValueType indicates whether this parameter is dynamically extracted or a constant value
	ValueType Value `json:"value_type" yaml:"value_type"`

	// Description tells the LLM what information to extract for this parameter
	// Be specific: "customer's email address" vs "user's shipping address including street, city, zip"
	Description string `json:"description" yaml:"description"`

	// Identifier is the parameter name that will be used in the HTTP request
	// This becomes the JSON key name or query parameter name in the outgoing request
	Identifier string `json:"identifier" yaml:"identifier"`

	// Required indicates if the LLM must extract this parameter before making the request
	// If true and the parameter cannot be extracted, the tool execution will fail
	Required bool `json:"required" yaml:"required"`

	// Value is the static value for constant parameters
	// Only used when ValueType is CONSTANT or STATIC
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

// Endpoint defines a complete MCP Endpoint that proxies to an HTTP endpoint
// This can represent tools (actions), resources (data), or prompts (templates)
type Endpoint struct {
	// Capability specifies what kind of MCP Endpoint this represents
	// TOOL: executable actions, RESOURCE: readable data, PROMPT: reusable templates
	Capability Capability `json:"capability" yaml:"capability"`

	// Mode specifies webhook vs client integration (only used when Capability is TOOL)
	// For RESOURCE and PROMPT entities, this field is ignored
	Mode Mode `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Name is the Endpoint identifier that the LLM will use to reference this Endpoint
	// Tools: action names like "create_order", "send_email", "update_user_profile"
	// Resources: data sources like "user_profile", "product_catalog", "company_policies"
	// Prompts: template names like "email_template", "code_review", "customer_service"
	Name string `json:"name" yaml:"name"`

	// Method defines the HTTP method for the proxy request to your endpoint
	Method Method `json:"method" yaml:"method"`

	// Path is the endpoint path that will be appended to the backend's BaseURL
	// Supports path parameter templates using curly braces: "/users/{user_id}/orders/{order_id}"
	// Examples: "/orders", "/users/{user_id}", "/templates/generate"
	// The full URL becomes: Backend.BaseURL + Endpoint.Path
	Path string `json:"path" yaml:"path"`

	// Description explains the Endpoint's purpose to the LLM
	// Tools: when and how to use this action and any constraints or requirements
	// Resources: what data this resource contains and when to reference it
	// Prompts: what this template is for and when to invoke it
	// Example: "Creates a new customer order with the provided items and shipping details"
	Description string `json:"description" yaml:"description"`

	// Headers define HTTP headers to include in requests to your endpoint
	// Common uses: authentication tokens, content-type specifications, custom API headers
	Headers []*Header `json:"headers" yaml:"headers"`

	// WaitResponse determines conversation flow control
	// Tools: true = wait for action completion, false = fire-and-forget
	// Resources: typically true to wait for data retrieval
	// Prompts: typically true to wait for template processing
	WaitResponse bool `json:"wait_response" yaml:"wait_response"`

	// ResponseTimeout sets maximum wait time for your endpoint to respond
	// Only applies when WaitResponse is true. Default: 20 seconds
	// Consider your endpoint's typical response time when setting this value
	ResponseTimeout time.Duration `json:"response_timeout" yaml:"response_timeout"`

	// BodyParams define data that will be extracted and sent in the HTTP request body
	// Tools: parameters for the action to execute
	// Resources: filters or criteria for data retrieval
	// Prompts: variables to substitute into the template
	BodyParams []*Param `json:"body_params" yaml:"body_params"`

	// QueryParameters define data that will be extracted and sent as URL query parameters
	// Example: ?user_id=123&include_details=true
	// Commonly used for pagination, filtering, or simple parameter passing
	QueryParameters []*Param `json:"query_parameters" yaml:"query_parameters"`

	// PathParameters define variables that will be substituted into the URL path
	// Use curly braces in your URL template: "/users/{user_id}/orders/{order_id}"
	// The LLM will extract these values and substitute them into the path
	PathParameters []*Param `json:"path_parameters" yaml:"path_parameters"`
}
