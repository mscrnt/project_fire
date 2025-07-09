package agent

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Config contains configuration for the agent server
type Config struct {
	Port     int    // Server port
	CertFile string // Server certificate file
	KeyFile  string // Server private key file
	CAFile   string // CA certificate file for client verification
	LogFile  string // Optional log file path
}

// DefaultConfig returns default agent configuration
func DefaultConfig() Config {
	return Config{
		Port: 2223,
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.CertFile == "" {
		return fmt.Errorf("server certificate file is required")
	}

	if c.KeyFile == "" {
		return fmt.Errorf("server key file is required")
	}

	if c.CAFile == "" {
		return fmt.Errorf("CA certificate file is required")
	}

	// Check if files exist
	if _, err := os.Stat(c.CertFile); err != nil {
		return fmt.Errorf("certificate file not found: %s", c.CertFile)
	}

	if _, err := os.Stat(c.KeyFile); err != nil {
		return fmt.Errorf("key file not found: %s", c.KeyFile)
	}

	if _, err := os.Stat(c.CAFile); err != nil {
		return fmt.Errorf("CA file not found: %s", c.CAFile)
	}

	return nil
}

// LoadTLSConfig creates TLS configuration from the agent config
func (c Config) LoadTLSConfig() (*tls.Config, error) {
	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Load CA certificate for client verification
	caCert, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config with mTLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS13,
	}

	return tlsConfig, nil
}

// ClientConfig contains configuration for the agent client
type ClientConfig struct {
	Host     string // Target host
	Port     int    // Target port
	CertFile string // Client certificate file
	KeyFile  string // Client private key file
	CAFile   string // CA certificate file for server verification
	Endpoint string // Endpoint to connect to
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Host: "localhost",
		Port: 2223,
	}
}

// Validate checks if the client configuration is valid
func (c ClientConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.CertFile == "" {
		return fmt.Errorf("client certificate file is required")
	}

	if c.KeyFile == "" {
		return fmt.Errorf("client key file is required")
	}

	if c.CAFile == "" {
		return fmt.Errorf("CA certificate file is required")
	}

	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	// Check if files exist
	if _, err := os.Stat(c.CertFile); err != nil {
		return fmt.Errorf("certificate file not found: %s", c.CertFile)
	}

	if _, err := os.Stat(c.KeyFile); err != nil {
		return fmt.Errorf("key file not found: %s", c.KeyFile)
	}

	if _, err := os.Stat(c.CAFile); err != nil {
		return fmt.Errorf("CA file not found: %s", c.CAFile)
	}

	return nil
}

// LoadClientTLSConfig creates TLS configuration for the client
func (c ClientConfig) LoadClientTLSConfig() (*tls.Config, error) {
	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate for server verification
	caCert, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
	}

	return tlsConfig, nil
}