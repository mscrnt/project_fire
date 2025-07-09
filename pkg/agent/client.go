package agent

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an agent client
type Client struct {
	config     ClientConfig
	httpClient *http.Client
}

// NewClient creates a new agent client
func NewClient(config ClientConfig) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Load TLS config
	tlsConfig, err := config.LoadClientTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Create HTTP client with TLS
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 30 * time.Second,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// Connect connects to the specified endpoint and returns the response
func (c *Client) Connect() ([]byte, error) {
	// Build URL
	url := fmt.Sprintf("https://%s:%d/%s", c.config.Host, c.config.Port, c.config.Endpoint)

	// Make request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// CheckHealth checks if the agent is healthy
func (c *Client) CheckHealth() error {
	// Override endpoint temporarily
	originalEndpoint := c.config.Endpoint
	c.config.Endpoint = "health"
	defer func() {
		c.config.Endpoint = originalEndpoint
	}()

	body, err := c.Connect()
	if err != nil {
		return err
	}

	if string(body) != "OK\n" {
		return fmt.Errorf("unexpected health response: %s", string(body))
	}

	return nil
}
