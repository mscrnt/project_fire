//go:build integration
// +build integration

package agent

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAgentIntegration tests the full agent server and client flow
func TestAgentIntegration(t *testing.T) {
	// Create temporary directory for certificates
	tempDir, err := os.MkdirTemp("", "fire-agent-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	caFile, serverCertFile, serverKeyFile, clientCertFile, clientKeyFile := generateTestCertificates(t, tempDir)

	// Find available port
	port := findAvailablePort(t)

	// Create server config
	serverConfig := Config{
		Port:     port,
		CertFile: serverCertFile,
		KeyFile:  serverKeyFile,
		CAFile:   caFile,
	}

	// Create and start server
	server, err := NewServer(serverConfig)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client config
	clientConfig := ClientConfig{
		Host:     "localhost",
		Port:     port,
		CertFile: clientCertFile,
		KeyFile:  clientKeyFile,
		CAFile:   caFile,
	}

	// Test each endpoint
	endpoints := []string{"sysinfo", "sensors", "health"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			clientConfig.Endpoint = endpoint

			client, err := NewClient(&clientConfig)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			data, err := client.Connect()
			if err != nil {
				t.Fatalf("failed to connect: %v", err)
			}

			if len(data) == 0 {
				t.Error("received empty response")
			}

			t.Logf("Received %d bytes from %s endpoint", len(data), endpoint)
		})
	}

	// Test health check
	t.Run("health_check", func(t *testing.T) {
		clientConfig.Endpoint = "health"
		client, err := NewClient(&clientConfig)
		if err != nil {
			t.Fatal(err)
		}

		if err := client.CheckHealth(); err != nil {
			t.Errorf("health check failed: %v", err)
		}
	})

	// Shutdown server
	if err := server.Shutdown(context.TODO()); err != nil {
		t.Errorf("failed to shutdown server: %v", err)
	}
}

// findAvailablePort finds an available port for testing
func findAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return port
}

// generateTestCertificates generates a CA and certificates for testing
func generateTestCertificates(t *testing.T, dir string) (caFile, serverCertFile, serverKeyFile, clientCertFile, clientKeyFile string) {
	// Generate CA
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatal(err)
	}

	// Save CA
	caFile = filepath.Join(dir, "ca.crt")
	caKeyFile := filepath.Join(dir, "ca.key")

	saveCertificate(t, caFile, caCertDER)
	savePrivateKey(t, caKeyFile, caKey)

	// Generate server certificate
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:    []string{"localhost"},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	serverCertFile = filepath.Join(dir, "server.crt")
	serverKeyFile = filepath.Join(dir, "server.key")

	saveCertificate(t, serverCertFile, serverCertDER)
	savePrivateKey(t, serverKeyFile, serverKey)

	// Generate client certificate
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	clientCertFile = filepath.Join(dir, "client.crt")
	clientKeyFile = filepath.Join(dir, "client.key")

	saveCertificate(t, clientCertFile, clientCertDER)
	savePrivateKey(t, clientKeyFile, clientKey)

	return
}

// saveCertificate saves a certificate to a file
func saveCertificate(t *testing.T, filename string, derBytes []byte) {
	certOut, err := os.Create(filename) // #nosec G304 -- filename is from controlled test input
	if err != nil {
		t.Fatal(err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatal(err)
	}
}

// savePrivateKey saves a private key to a file
func savePrivateKey(t *testing.T, filename string, key *rsa.PrivateKey) {
	keyOut, err := os.Create(filename) // #nosec G304 -- filename is from controlled test input
	if err != nil {
		t.Fatal(err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		t.Fatal(err)
	}
}
