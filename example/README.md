# Demo Setup Guide

This guide helps you run the complete MCP HTTP Proxy demo with a real REST API backend.

## 🚀 Complete Setup Commands

Here's the complete sequence to get everything running:

```bash
# Terminal 1: Start Demo API
mkdir mcp-demo && cd mcp-demo
go mod init mcp-demo
go get github.com/gorilla/mux

# Save demo-api.go and run
go run demo-api.go

# Terminal 2: Start MCP Proxy (from your project root)
export DEMO_API_URL=http://localhost:8080
export API_TOKEN=demo-token-123

# Ensure config.yml is in ./example/ directory
go run cmd/proxy/main.go --config ./example/config.yml
```

## 🚀 Quick Start

### 1. Start the Demo REST API

First, save the demo API code and install dependencies:

```bash
# Create demo directory
mkdir mcp-demo && cd mcp-demo

# Save the demo API code as demo-api.go
# (copy the Go code from the artifact)

# Install dependencies
go mod init mcp-demo
go get github.com/gorilla/mux

# Start the demo API server
go run demo-api.go
```

The API will start on `http://localhost:8080` with these endpoints:

- **GET** `/api/users/{id}/profile` - User profile data
- **PATCH** `/api/users/{id}/preferences` - Update user settings
- **GET** `/api/products/search` - Search product catalog
- **POST** `/api/orders` - Create new orders
- **GET** `/api/orders/{id}/status` - Check order status
- **POST** `/api/notifications` - Send notifications
- **POST** `/api/templates/email` - Generate email templates
- **POST** `/api/templates/welcome` - Generate welcome messages

### 2. Set Environment Variables

```bash
export DEMO_API_URL=http://localhost:8080
export API_TOKEN=demo-token-123
```

### 3. Configure and Start MCP Proxy

Save the configuration as `config.yml` in the `example/` directory, then start the MCP proxy:

```bash
# Start the MCP HTTP Proxy with configuration
go run cmd/proxy/main.go --config ./example/config.yml
```

The proxy will start and connect to your demo API backend, creating MCP endpoints for all configured tools, resources, and prompts.

## 🧪 Test the Demo

With both the demo API and MCP proxy running:

1. **Demo API** → `http://localhost:8080` (REST endpoints)
2. **MCP Proxy** → Converts MCP calls to HTTP requests

### Test Individual Endpoints

```bash
# Test user profile
curl http://localhost:8080/api/users/user123/profile

# Test product search
curl "http://localhost:8080/api/products/search?search=headphones&in_stock_only=true"

# Test order status
curl http://localhost:8080/api/orders/ORD1001/status

# Test order creation
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_name": "Jane Smith",
    "customer_email": "jane@example.com",
    "items": [{"product_id": "prod001", "quantity": 1}],
    "shipping_address": {
      "street": "456 Oak Ave",
      "city": "Portland",
      "state": "OR",
      "zip": "97201",
      "country": "USA"
    },
    "express_shipping": true
  }'

# Test notification
curl -X POST http://localhost:8080/api/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Order Confirmed",
    "message": "Your order has been successfully placed!",
    "type": "success"
  }'

# Test email template
curl -X POST http://localhost:8080/api/templates/email \
  -H "Content-Type: application/json" \
  -d '{
    "customer_name": "John",
    "subject": "Order Confirmation",
    "template_type": "order_confirmation",
    "context_data": {"order_id": "ORD1002"},
    "tone": "friendly"
  }'
```

## 📊 Demo Data

The API includes sample data:

### Users
- **user123**: John Doe with premium account and full profile

### Products
- **prod001**: Wireless Headphones ($199.99) - In Stock
- **prod002**: Coffee Mug ($15.99) - In Stock
- **prod003**: Laptop Stand ($89.99) - Out of Stock

### Orders
- **ORD1001**: Sample shipped order for user123

## 🎯 MCP Integration Examples

With both servers running, these are example interactions through the MCP proxy:

### Customer Service Scenarios

**"I want to order wireless headphones"**
→ Uses `product_search` to find headphones
→ Uses `create_order` to place the order
→ Uses `show_notification` to confirm success

**"What's the status of order ORD1001?"**
→ Uses `order_status` to check delivery info

**"Change my notification preferences"**
→ Uses `user_profile` to get current settings
→ Uses `update_user_preferences` to save changes

**"Send a shipping update email"**
→ Uses `customer_service_email` template with shipping context

### E-commerce Scenarios

**"Show me electronics under $200"**
→ Uses `product_search` with category and price filters

**"Create a welcome email for new Spanish users"**
→ Uses `welcome_message` with Spanish language setting

## 🔧 Customization

### Adding New Endpoints

1. Add handler function to `demo-api.go`
2. Register route in `main()` function
3. Add corresponding configuration in `config.yaml`
4. Restart both servers

### Modifying Data

Edit the `initDemoData()` function to change:
- User profiles and preferences
- Product catalog and pricing
- Sample orders and statuses

### Environment Variables

Set these for different environments:

```bash
# Development
export DEMO_API_URL=http://localhost:8080
export API_TOKEN=dev-token

# Production
export DEMO_API_URL=https://api.yourcompany.com
export API_TOKEN=prod-secure-token
```

## 🐛 Troubleshooting

### Common Issues

**"Connection refused"**
- Make sure demo API is running on port 8080: `curl http://localhost:8080/api/users/user123/profile`
- Check if port is already in use: `lsof -i :8080`
- Verify MCP proxy is running with correct config

**"Config file not found"**
- Ensure `config.yml` exists in `./example/` directory
- Check the file path: `ls -la ./example/config.yml`
- Verify current working directory when running proxy

**"Failed to load configuration"**
- Check YAML syntax in `config.yml`
- Verify environment variables are set: `echo $DEMO_API_URL`
- Run with verbose logging: `go run cmd/proxy/main.go --config ./example/config.yml --verbose`

**"User not found"**
- Use `user123` as the user ID in tests
- Check the `initDemoData()` function for available users

**"Product not found"**
- Use `prod001`, `prod002`, or `prod003` as product IDs
- Check product availability with `/api/products/search`

**"Order not found"**
- Use `ORD1001` for the sample order
- Create new orders via `/api/orders` endpoint

### Debug Mode

Add logging to see requests:

```go
// Add this middleware to see all requests
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
})
```

## 🎉 Next Steps

1. **Try real conversations** with your MCP client using the configured endpoints
2. **Monitor the API logs** to see how the proxy translates LLM requests
3. **Experiment with different prompts** to test parameter extraction
4. **Add your own endpoints** to integrate with existing services
5. **Deploy to production** with proper authentication and error handling

The demo provides a foundation for understanding how MCP proxies work with real HTTP APIs!
