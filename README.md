# MCP HTTP Proxy

A configurable proxy server that bridges Model Context Protocol (MCP) endpoints to HTTP APIs. Transform LLM tool calls, resource requests, and prompt templates into HTTP requests without writing code - just configure via YAML.

## 🚀 Features

- **Universal HTTP Integration** - Connect any REST API to MCP-compatible LLMs
- **Zero Code Required** - Pure YAML configuration
- **Dynamic Parameter Extraction** - LLM automatically extracts parameters from conversations
- **Multiple Capabilities** - Support for tools, resources, and prompts
- **Flexible Deployment** - Webhook (server-side) or client-side tool execution
- **RESTful API Support** - Path parameters, query strings, headers, and request bodies
- **Environment Variables** - Secure configuration with variable substitution

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Capabilities](#capabilities)
- [Parameter Types](#parameter-types)
- [Examples](#examples)
- [Environment Variables](#environment-variables)
- [API Reference](#api-reference)
- [Contributing](#contributing)

## 🏁 Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/paulgrammer/mcp-proxy
   cd mcp-proxy
   ```

2. **Run the proxy**
   ```bash
   go run main.go --config config/proxy.yaml
   ```

## ⚙️ Configuration

Configuration is done via YAML files. Each endpoint definition maps an MCP capability to an HTTP endpoint:

```yaml
- capability: tool
  mode: webhook
  name: create_order
  url: "${API_BASE_URL}/orders"
  method: POST
  description: "Creates a new customer order"
  wait_response: true
  response_timeout: 30s

  headers:
    - type: constant
      name: Authorization
      value: "Bearer ${API_TOKEN}"

  body_params:
    - data_type: string
      value_type: dynamic
      description: "customer's email address"
      identifier: customer_email
      required: true
```

## 🎯 Capabilities

### Tools
Execute actions and operations:
- **Webhook mode** - Send data to your HTTP server
- **Client mode** - Trigger client-side events

```yaml
capability: tool
mode: webhook  # or 'client'
```

### Resources
Retrieve and reference data:
```yaml
capability: resource
```

### Prompts
Generate templates and content:
```yaml
capability: prompt
```

## 📊 Parameter Types

### Value Types
- **`dynamic`** - Extracted by LLM from conversation
- **`constant`** - Predefined static values

### Data Types
- `string`, `number`, `boolean`, `object`, `array`

### Parameter Locations
- **Body Parameters** - JSON request payload
- **Query Parameters** - URL query string (`?param=value`)
- **Path Parameters** - URL path variables (`/users/{user_id}`)
- **Headers** - HTTP headers

## 💡 Examples

### E-commerce Order Tool
```yaml
- capability: tool
  mode: webhook
  name: create_order
  url: "https://api.shop.com/orders"
  method: POST
  description: "Creates a customer order with items and shipping"

  body_params:
    - data_type: array
      value_type: dynamic
      description: "list of items with product ID and quantity"
      identifier: items
      required: true
    - data_type: object
      value_type: dynamic
      description: "shipping address"
      identifier: shipping_address
      required: true
```

### User Profile Resource
```yaml
- capability: resource
  name: user_profile
  url: "https://api.users.com/users/{user_id}"
  method: GET
  description: "Retrieves user profile information"

  path_parameters:
    - data_type: string
      value_type: dynamic
      description: "user's unique identifier"
      identifier: user_id
      required: true
```

### Email Template Prompt
```yaml
- capability: prompt
  name: support_email
  url: "https://templates.com/generate"
  method: POST
  description: "Generates customer support email templates"

  body_params:
    - data_type: string
      value_type: dynamic
      description: "customer's name"
      identifier: customer_name
      required: true
    - data_type: string
      value_type: dynamic
      description: "support issue type"
      identifier: issue_type
      required: true
```

## 🔒 Environment Variables

Secure your configuration with environment variable substitution:

```yaml
url: "${API_BASE_URL}/api/v1/users"
headers:
  - type: constant
    name: Authorization
    value: "Bearer ${API_TOKEN}"
```

## 📖 Configuration Reference

### Endpoint Structure

| Field | Type | Description |
|-------|------|-------------|
| `capability` | string | Type of MCP endpoint: `tool`, `resource`, or `prompt` |
| `mode` | string | Tool execution mode: `webhook` or `client` (tools only) |
| `name` | string | Unique identifier for the endpoint |
| `url` | string | Target HTTP endpoint (supports templates and env vars) |
| `method` | string | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE` |
| `description` | string | Human-readable description for the LLM |
| `wait_response` | boolean | Whether to wait for HTTP response |
| `response_timeout` | duration | Maximum wait time (e.g., `30s`, `5m`) |

### Parameter Types

| Field | Type | Description |
|-------|------|-------------|
| `data_type` | string | Expected data type |
| `value_type` | string | `dynamic` (LLM-extracted) or `constant` |
| `description` | string | What the LLM should extract |
| `identifier` | string | Parameter name in HTTP request |
| `required` | boolean | Whether parameter is mandatory |

## 🔧 Advanced Configuration

### URL Templates
Support for dynamic URLs with path parameters:
```yaml
url: "/api/users/{user_id}/orders/{order_id}"
```

### Environment Variables
Reference environment variables in any string field:
```yaml
url: "${BASE_URL}/api/v1/endpoint"
headers:
  - name: Authorization
    value: "Bearer ${SECRET_TOKEN}"
```

### Response Handling
Configure how the proxy handles HTTP responses:
```yaml
wait_response: true        # Wait for response
response_timeout: 30s      # Timeout after 30 seconds
```

## 🚀 Getting Started

1. **Define your endpoints** in a YAML configuration file
2. **Set environment variables** for sensitive data
3. **Run the proxy server** pointing to your config
4. **Connect your MCP client** to start using the endpoints

The proxy automatically handles parameter extraction, HTTP request formatting, and response processing based on your YAML configuration.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- [Model Context Protocol Documentation](https://modelcontextprotocol.io/)
- [Examples Repository](examples/)
- [Configuration Samples](config/)
