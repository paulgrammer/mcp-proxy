package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Data models
type User struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Email       string            `json:"email"`
	Phone       string            `json:"phone"`
	Address     Address           `json:"address"`
	Preferences UserPreferences   `json:"preferences"`
	CreatedAt   time.Time         `json:"created_at"`
	Metadata    map[string]string `json:"metadata"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

type UserPreferences struct {
	Language      string `json:"language"`
	Timezone      string `json:"timezone"`
	Notifications struct {
		Email bool `json:"email"`
		SMS   bool `json:"sms"`
		Push  bool `json:"push"`
	} `json:"notifications"`
	Privacy struct {
		DataSharing bool `json:"data_sharing"`
		Marketing   bool `json:"marketing"`
	} `json:"privacy"`
}

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	InStock     bool    `json:"in_stock"`
	Quantity    int     `json:"quantity"`
	ImageURL    string  `json:"image_url"`
}

type Order struct {
	ID                string      `json:"id"`
	CustomerID        string      `json:"customer_id"`
	CustomerName      string      `json:"customer_name"`
	CustomerEmail     string      `json:"customer_email"`
	Items             []OrderItem `json:"items"`
	ShippingAddress   Address     `json:"shipping_address"`
	Status            string      `json:"status"`
	Total             float64     `json:"total"`
	ExpressShipping   bool        `json:"express_shipping"`
	Currency          string      `json:"currency"`
	CreatedAt         time.Time   `json:"created_at"`
	EstimatedDelivery time.Time   `json:"estimated_delivery"`
}

type OrderItem struct {
	ProductID      string            `json:"product_id"`
	ProductName    string            `json:"product_name"`
	Quantity       int               `json:"quantity"`
	Price          float64           `json:"price"`
	Customizations map[string]string `json:"customizations,omitempty"`
}

type Notification struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"`
	UserID  string `json:"user_id,omitempty"`
}

type EmailTemplate struct {
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	TemplateType string            `json:"template_type"`
	Variables    map[string]string `json:"variables"`
}

// In-memory storage
var (
	users        = make(map[string]*User)
	products     = make(map[string]*Product)
	orders       = make(map[string]*Order)
	orderCounter = 1000
)

// Initialize demo data
func initDemoData() {
	// Sample users
	users["user123"] = &User{
		ID:    "user123",
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "+1-555-0123",
		Address: Address{
			Street:  "123 Main St",
			City:    "San Francisco",
			State:   "CA",
			Zip:     "94105",
			Country: "USA",
		},
		Preferences: UserPreferences{
			Language: "en",
			Timezone: "America/Los_Angeles",
		},
		CreatedAt: time.Now().AddDate(0, -3, 0),
		Metadata:  map[string]string{"tier": "premium", "source": "web"},
	}

	// Sample products
	products["prod001"] = &Product{
		ID:          "prod001",
		Name:        "Wireless Headphones",
		Description: "High-quality wireless headphones with noise cancellation",
		Price:       199.99,
		Category:    "electronics",
		InStock:     true,
		Quantity:    50,
		ImageURL:    "https://example.com/headphones.jpg",
	}

	products["prod002"] = &Product{
		ID:          "prod002",
		Name:        "Coffee Mug",
		Description: "Ceramic coffee mug with company logo",
		Price:       15.99,
		Category:    "accessories",
		InStock:     true,
		Quantity:    100,
		ImageURL:    "https://example.com/mug.jpg",
	}

	products["prod003"] = &Product{
		ID:          "prod003",
		Name:        "Laptop Stand",
		Description: "Adjustable aluminum laptop stand",
		Price:       89.99,
		Category:    "accessories",
		InStock:     false,
		Quantity:    0,
		ImageURL:    "https://example.com/stand.jpg",
	}

	// Sample order
	orders["ORD1001"] = &Order{
		ID:            "ORD1001",
		CustomerID:    "user123",
		CustomerName:  "John Doe",
		CustomerEmail: "john@example.com",
		Items: []OrderItem{
			{
				ProductID:   "prod001",
				ProductName: "Wireless Headphones",
				Quantity:    1,
				Price:       199.99,
			},
		},
		ShippingAddress: Address{
			Street:  "123 Main St",
			City:    "San Francisco",
			State:   "CA",
			Zip:     "94105",
			Country: "USA",
		},
		Status:            "shipped",
		Total:             199.99,
		ExpressShipping:   true,
		Currency:          "USD",
		CreatedAt:         time.Now().AddDate(0, 0, -2),
		EstimatedDelivery: time.Now().AddDate(0, 0, 1),
	}
}

// API Handlers

// GET /api/users/{id}/profile - Get user profile
func getUserProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	user, exists := users[userID]
	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// PATCH /api/users/{id}/preferences - Update user preferences
func updateUserPreferences(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	user, exists := users[userID]
	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var updates struct {
		Notifications *struct {
			Email bool `json:"email"`
			SMS   bool `json:"sms"`
			Push  bool `json:"push"`
		} `json:"notifications,omitempty"`
		Privacy *struct {
			DataSharing bool `json:"data_sharing"`
			Marketing   bool `json:"marketing"`
		} `json:"privacy,omitempty"`
		Language *string `json:"language,omitempty"`
		Timezone *string `json:"timezone,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Apply updates
	if updates.Notifications != nil {
		user.Preferences.Notifications.Email = updates.Notifications.Email
		user.Preferences.Notifications.SMS = updates.Notifications.SMS
		user.Preferences.Notifications.Push = updates.Notifications.Push
	}
	if updates.Privacy != nil {
		user.Preferences.Privacy.DataSharing = updates.Privacy.DataSharing
		user.Preferences.Privacy.Marketing = updates.Privacy.Marketing
	}
	if updates.Language != nil {
		user.Preferences.Language = *updates.Language
	}
	if updates.Timezone != nil {
		user.Preferences.Timezone = *updates.Timezone
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// GET /api/products/search - Search products
func searchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	search := query.Get("search")
	category := query.Get("category")
	inStockOnly := query.Get("in_stock_only") == "true"

	minPrice, _ := strconv.ParseFloat(query.Get("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(query.Get("max_price"), 64)

	limit := 10
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	var results []*Product
	for _, product := range products {
		// Apply filters
		if search != "" && !strings.Contains(strings.ToLower(product.Name), strings.ToLower(search)) {
			continue
		}
		if category != "" && product.Category != category {
			continue
		}
		if inStockOnly && !product.InStock {
			continue
		}
		if minPrice > 0 && product.Price < minPrice {
			continue
		}
		if maxPrice > 0 && product.Price > maxPrice {
			continue
		}

		results = append(results, product)
		if len(results) >= limit {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"products": results,
		"total":    len(results),
		"filters": map[string]interface{}{
			"search":        search,
			"category":      category,
			"in_stock_only": inStockOnly,
			"min_price":     minPrice,
			"max_price":     maxPrice,
		},
	})
}

// POST /api/orders - Create order
func createOrder(w http.ResponseWriter, r *http.Request) {
	var orderReq struct {
		CustomerName    string      `json:"customer_name"`
		CustomerEmail   string      `json:"customer_email"`
		Items           []OrderItem `json:"items"`
		ShippingAddress Address     `json:"shipping_address"`
		ExpressShipping bool        `json:"express_shipping"`
		Source          string      `json:"source"`
	}

	if err := json.NewDecoder(r.Body).Decode(&orderReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Generate order ID
	orderCounter++
	orderID := fmt.Sprintf("ORD%d", orderCounter)

	// Calculate total
	var total float64
	for i, item := range orderReq.Items {
		if product, exists := products[item.ProductID]; exists {
			orderReq.Items[i].ProductName = product.Name
			orderReq.Items[i].Price = product.Price
			total += product.Price * float64(item.Quantity)
		}
	}

	// Create order
	order := &Order{
		ID:                orderID,
		CustomerName:      orderReq.CustomerName,
		CustomerEmail:     orderReq.CustomerEmail,
		Items:             orderReq.Items,
		ShippingAddress:   orderReq.ShippingAddress,
		Status:            "pending",
		Total:             total,
		ExpressShipping:   orderReq.ExpressShipping,
		Currency:          "USD",
		CreatedAt:         time.Now(),
		EstimatedDelivery: time.Now().AddDate(0, 0, 3), // 3 days default
	}

	if orderReq.ExpressShipping {
		order.EstimatedDelivery = time.Now().AddDate(0, 0, 1) // 1 day for express
	}

	orders[orderID] = order

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// GET /api/orders/{id}/status - Get order status
func getOrderStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	order, exists := orders[orderID]
	if !exists {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	status := map[string]interface{}{
		"order_id":           order.ID,
		"status":             order.Status,
		"created_at":         order.CreatedAt,
		"estimated_delivery": order.EstimatedDelivery,
		"total":              order.Total,
		"items_count":        len(order.Items),
		"express_shipping":   order.ExpressShipping,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// POST /api/notifications - Send notification (client tool)
func sendNotification(w http.ResponseWriter, r *http.Request) {
	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simulate notification processing
	log.Printf("📱 Notification: [%s] %s - %s", notification.Type, notification.Title, notification.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "sent",
		"timestamp":    time.Now(),
		"notification": notification,
	})
}

// POST /api/templates/email - Generate email template
func generateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerName    string                 `json:"customer_name"`
		Subject         string                 `json:"subject"`
		TemplateType    string                 `json:"template_type"`
		ContextData     map[string]interface{} `json:"context_data"`
		SenderSignature string                 `json:"sender_signature"`
		Tone            string                 `json:"tone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Generate email based on template type
	var body string
	switch req.TemplateType {
	case "order_confirmation":
		body = fmt.Sprintf("Dear %s,\n\nThank you for your order! We're excited to confirm that we've received your order and it's being processed.\n\nOrder Details:\n%v\n\nBest regards,\n%s",
			req.CustomerName, req.ContextData, req.SenderSignature)
	case "shipping_update":
		body = fmt.Sprintf("Dear %s,\n\nGreat news! Your order is on its way.\n\nShipping Details:\n%v\n\nBest regards,\n%s",
			req.CustomerName, req.ContextData, req.SenderSignature)
	default:
		body = fmt.Sprintf("Dear %s,\n\nThank you for contacting us.\n\nContext: %v\n\nBest regards,\n%s",
			req.CustomerName, req.ContextData, req.SenderSignature)
	}

	template := EmailTemplate{
		Subject:      req.Subject,
		Body:         body,
		TemplateType: req.TemplateType,
		Variables: map[string]string{
			"customer_name": req.CustomerName,
			"tone":          req.Tone,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// POST /api/templates/welcome - Generate welcome message
func generateWelcomeTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Language        string                 `json:"language"`
		CustomerName    string                 `json:"customer_name"`
		Context         string                 `json:"context"`
		Personalization map[string]interface{} `json:"personalization"`
		CulturalContext bool                   `json:"cultural_context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Generate localized welcome message
	greetings := map[string]string{
		"en": "Welcome",
		"es": "Bienvenido",
		"fr": "Bienvenue",
		"de": "Willkommen",
		"it": "Benvenuto",
	}

	greeting := greetings["en"] // default
	if g, exists := greetings[req.Language]; exists {
		greeting = g
	}

	message := fmt.Sprintf("%s %s! We're delighted to have you join us.", greeting, req.CustomerName)

	if req.Context == "premium_upgrade" {
		message += " Thank you for upgrading to premium!"
	}

	template := map[string]interface{}{
		"message":      message,
		"language":     req.Language,
		"context":      req.Context,
		"cultural":     req.CulturalContext,
		"generated_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func main() {
	initDemoData()

	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// User endpoints
	api.HandleFunc("/users/{id}/profile", getUserProfile).Methods("GET")
	api.HandleFunc("/users/{id}/preferences", updateUserPreferences).Methods("PATCH")

	// Product endpoints
	api.HandleFunc("/products/search", searchProducts).Methods("GET")

	// Order endpoints
	api.HandleFunc("/orders", createOrder).Methods("POST")
	api.HandleFunc("/orders/{id}/status", getOrderStatus).Methods("GET")

	// Notification endpoints
	api.HandleFunc("/notifications", sendNotification).Methods("POST")

	// Template endpoints
	api.HandleFunc("/templates/email", generateEmailTemplate).Methods("POST")
	api.HandleFunc("/templates/welcome", generateWelcomeTemplate).Methods("POST")

	// Add CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	fmt.Println("🚀 Demo REST API Server starting on :8181")
	fmt.Println("📚 Available endpoints:")
	fmt.Println("  GET  /api/users/{id}/profile")
	fmt.Println("  PATCH /api/users/{id}/preferences")
	fmt.Println("  GET  /api/products/search")
	fmt.Println("  POST /api/orders")
	fmt.Println("  GET  /api/orders/{id}/status")
	fmt.Println("  POST /api/notifications")
	fmt.Println("  POST /api/templates/email")
	fmt.Println("  POST /api/templates/welcome")
	fmt.Println("\n🔧 Test with: curl http://localhost:8181/api/users/user123/profile")

	log.Fatal(http.ListenAndServe(":8181", r))
}
