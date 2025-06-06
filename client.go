package proxy

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ClientConfig struct {
	Timeout         time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      1 * time.Second,
		MaxIdleConns:    100,
		MaxConnsPerHost: 10,
	}
}

type HTTPClient struct {
	client *http.Client
	config *ClientConfig
}

func NewHTTPClient(config *ClientConfig) *HTTPClient {
	if config == nil {
		config = DefaultClientConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxConnsPerHost,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	return &HTTPClient{
		client: client,
		config: config,
	}
}

func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.DoWithCircuitBreaker(ctx, req, nil)
}

func (c *HTTPClient) DoWithCircuitBreaker(ctx context.Context, req *http.Request, cb *CircuitBreaker) (*http.Response, error) {
	if cb != nil && !cb.CanExecute() {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	req = req.WithContext(ctx)

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		resp, err = c.client.Do(req)

		if err == nil && resp.StatusCode < 500 {
			if cb != nil {
				cb.RecordSuccess()
			}
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < c.config.MaxRetries {
			time.Sleep(c.config.RetryDelay * time.Duration(attempt+1))
		}
	}

	if cb != nil {
		cb.RecordFailure()
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, err)
	}

	return resp, nil
}

func (c *HTTPClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

type CircuitBreaker struct {
	mu            sync.RWMutex
	failureCount  int
	lastFailTime  time.Time
	maxFailures   int
	resetTimeout  time.Duration
	state         string
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == "closed" {
		return true
	}

	if cb.state == "open" && time.Since(cb.lastFailTime) > cb.resetTimeout {
		return true
	}

	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailTime = time.Now()

	if cb.failureCount >= cb.maxFailures {
		cb.state = "open"
	}
}

type ClientManager struct {
	clients        map[string]*HTTPClient
	defaultClient  *HTTPClient
	circuitBreaker *CircuitBreaker
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:        make(map[string]*HTTPClient),
		defaultClient:  NewHTTPClient(DefaultClientConfig()),
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

func (cm *ClientManager) GetClient(name string) *HTTPClient {
	if client, exists := cm.clients[name]; exists {
		return client
	}
	return cm.defaultClient
}

func (cm *ClientManager) SetClient(name string, config *ClientConfig) {
	cm.clients[name] = NewHTTPClient(config)
}

func (cm *ClientManager) DoRequest(ctx context.Context, req *http.Request, clientName string) (*http.Response, error) {
	client := cm.GetClient(clientName)
	return client.DoWithCircuitBreaker(ctx, req, cm.circuitBreaker)
}

func (cm *ClientManager) Close() error {
	for _, client := range cm.clients {
		client.Close()
	}
	cm.defaultClient.Close()
	return nil
}
