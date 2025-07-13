package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Create certs directory
	certDir := "certs"
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		log.Fatalf("Failed to create certs directory: %v", err)
	}

	fmt.Println("Generating mTLS certificates for F.I.R.E. agent...")

	// Generate CA
	fmt.Println("1. Generating CA certificate...")
	caKey, caCert := generateCA()
	saveCA(certDir, caKey, caCert)

	// Generate server certificate
	fmt.Println("2. Generating server certificate...")
	serverKey, serverCert := generateServerCert(caKey, caCert)
	saveServerCert(certDir, serverKey, serverCert)

	// Generate client certificate
	fmt.Println("3. Generating client certificate...")
	clientKey, clientCert := generateClientCert(caKey, caCert)
	saveClientCert(certDir, clientKey, clientCert)

	fmt.Println("\nCertificates generated successfully!")
	fmt.Printf("\nUsage:\n")
	fmt.Printf("  Server: bench agent serve --cert %s --key %s --ca %s\n",
		filepath.Join(certDir, "server.pem"),
		filepath.Join(certDir, "server-key.pem"),
		filepath.Join(certDir, "ca.pem"))
	fmt.Printf("  Client: bench agent connect --cert %s --key %s --ca %s\n",
		filepath.Join(certDir, "client.pem"),
		filepath.Join(certDir, "client-key.pem"),
		filepath.Join(certDir, "ca.pem"))
}

func generateCA() (*rsa.PrivateKey, *x509.Certificate) {
	// Generate RSA key
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatalf("Failed to generate CA key: %v", err)
	}

	// Create CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"F.I.R.E. Test Bench"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour * 10), // 10 years
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	// Create self-signed CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		log.Fatalf("Failed to create CA certificate: %v", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		log.Fatalf("Failed to parse CA certificate: %v", err)
	}

	return caKey, caCert
}

func generateServerCert(caKey *rsa.PrivateKey, caCert *x509.Certificate) (*rsa.PrivateKey, *x509.Certificate) {
	// Generate RSA key
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate server key: %v", err)
	}

	// Create server certificate template
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"F.I.R.E. Agent Server"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(0, 0, 0, 0)},
		DNSNames:    []string{"localhost", "*.local", "*"},
	}

	// Create server certificate signed by CA
	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		log.Fatalf("Failed to create server certificate: %v", err)
	}

	serverCert, err := x509.ParseCertificate(serverCertDER)
	if err != nil {
		log.Fatalf("Failed to parse server certificate: %v", err)
	}

	return serverKey, serverCert
}

func generateClientCert(caKey *rsa.PrivateKey, caCert *x509.Certificate) (*rsa.PrivateKey, *x509.Certificate) {
	// Generate RSA key
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate client key: %v", err)
	}

	// Create client certificate template
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization:  []string{"F.I.R.E. Agent Client"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "fire-agent-client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create client certificate signed by CA
	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		log.Fatalf("Failed to create client certificate: %v", err)
	}

	clientCert, err := x509.ParseCertificate(clientCertDER)
	if err != nil {
		log.Fatalf("Failed to parse client certificate: %v", err)
	}

	return clientKey, clientCert
}

func saveCA(dir string, key *rsa.PrivateKey, cert *x509.Certificate) {
	// Save CA certificate
	certPath := filepath.Join(dir, "ca.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		log.Fatalf("Failed to create CA cert file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
		certOut.Close()
		log.Fatalf("Failed to write CA cert: %v", err)
	}

	// Save CA private key
	keyPath := filepath.Join(dir, "ca-key.pem")
	keyOut, err := os.Create(keyPath)
	if err != nil {
		log.Fatalf("Failed to create CA key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		log.Fatalf("Failed to write CA key: %v", err)
	}

	if err := os.Chmod(keyPath, 0o600); err != nil {
		log.Fatalf("Failed to chmod CA key: %v", err)
	}
	fmt.Printf("  CA certificate: %s\n", certPath)
	fmt.Printf("  CA private key: %s (keep secure!)\n", keyPath)
}

func saveServerCert(dir string, key *rsa.PrivateKey, cert *x509.Certificate) {
	// Save server certificate
	certPath := filepath.Join(dir, "server.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		log.Fatalf("Failed to create server cert file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
		certOut.Close()
		log.Fatalf("Failed to write server cert: %v", err)
	}

	// Save server private key
	keyPath := filepath.Join(dir, "server-key.pem")
	keyOut, err := os.Create(keyPath)
	if err != nil {
		log.Fatalf("Failed to create server key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		log.Fatalf("Failed to write server key: %v", err)
	}

	if err := os.Chmod(keyPath, 0o600); err != nil {
		log.Fatalf("Failed to chmod server key: %v", err)
	}
	fmt.Printf("  Server certificate: %s\n", certPath)
	fmt.Printf("  Server private key: %s\n", keyPath)
}

func saveClientCert(dir string, key *rsa.PrivateKey, cert *x509.Certificate) {
	// Save client certificate
	certPath := filepath.Join(dir, "client.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		log.Fatalf("Failed to create client cert file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
		certOut.Close()
		log.Fatalf("Failed to write client cert: %v", err)
	}

	// Save client private key
	keyPath := filepath.Join(dir, "client-key.pem")
	keyOut, err := os.Create(keyPath)
	if err != nil {
		log.Fatalf("Failed to create client key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		log.Fatalf("Failed to write client key: %v", err)
	}

	if err := os.Chmod(keyPath, 0o600); err != nil {
		log.Fatalf("Failed to chmod client key: %v", err)
	}
	fmt.Printf("  Client certificate: %s\n", certPath)
	fmt.Printf("  Client private key: %s\n", keyPath)
}
