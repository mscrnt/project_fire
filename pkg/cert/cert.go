// Package cert provides certificate generation and management for mTLS communication.
package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/mscrnt/project_fire/pkg/db"
)

// CertificateIssuer handles certificate generation for test results
type CertificateIssuer struct {
	// CA certificate and key
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
}

// NewCertificateIssuer creates a new certificate issuer with a self-signed CA
func NewCertificateIssuer() (*CertificateIssuer, error) {
	// Generate RSA key pair for CA
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
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
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create self-signed CA certificate
	caCertDER, err := x509.CreateCertificate(
		rand.Reader,
		caTemplate,
		caTemplate,
		&caKey.PublicKey,
		caKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Parse the certificate
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return &CertificateIssuer{
		caCert: caCert,
		caKey:  caKey,
	}, nil
}

// SaveCA saves the CA certificate and key to files
func (i *CertificateIssuer) SaveCA(certPath, keyPath string) error {
	// Save CA certificate
	certOut, err := os.Create(certPath) // #nosec G304 -- certPath is provided by the user and validated by the caller
	if err != nil {
		return fmt.Errorf("failed to create CA cert file: %w", err)
	}
	defer func() { _ = certOut.Close() }()

	if err := pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: i.caCert.Raw,
	}); err != nil {
		return fmt.Errorf("failed to write CA cert: %w", err)
	}

	// Save CA private key
	keyOut, err := os.Create(keyPath) // #nosec G304 -- keyPath is provided by the user and validated by the caller
	if err != nil {
		return fmt.Errorf("failed to create CA key file: %w", err)
	}
	defer func() { _ = keyOut.Close() }()

	if err := pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(i.caKey),
	}); err != nil {
		return fmt.Errorf("failed to write CA key: %w", err)
	}

	// Set secure permissions on key file
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return fmt.Errorf("failed to set key permissions: %w", err)
	}

	return nil
}

// LoadCA loads CA certificate and key from files
func LoadCA(certPath, keyPath string) (*CertificateIssuer, error) {
	// Load CA certificate
	certPEM, err := os.ReadFile(certPath) // #nosec G304 -- certPath is a user-specified CA certificate file path
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode CA cert PEM")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA cert: %w", err)
	}

	// Load CA private key
	keyPEM, err := os.ReadFile(keyPath) // #nosec G304 -- keyPath is a user-specified CA key file path
	if err != nil {
		return nil, fmt.Errorf("failed to read CA key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode CA key PEM")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA key: %w", err)
	}

	return &CertificateIssuer{
		caCert: caCert,
		caKey:  caKey,
	}, nil
}

// IssueCertificate generates a certificate for a test run
func (i *CertificateIssuer) IssueCertificate(run *db.Run, results []*db.Result) (*Certificate, error) {
	// Generate key pair for the certificate
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"F.I.R.E. Test Result"},
			Country:      []string{"US"},
			CommonName:   fmt.Sprintf("Test Run #%d", run.ID),
		},
		NotBefore:    run.StartTime,
		NotAfter:     run.StartTime.Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
	}

	// Add custom extensions with test data
	extensions := i.buildExtensions(run, results)
	template.ExtraExtensions = extensions

	// Create certificate
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		i.caCert,
		&key.PublicKey,
		i.caKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &Certificate{
		Certificate: cert,
		PrivateKey:  key,
		RunID:       run.ID,
		IssuedAt:    time.Now(),
	}, nil
}

// buildExtensions creates X.509 extensions containing test data
func (i *CertificateIssuer) buildExtensions(run *db.Run, results []*db.Result) []pkix.Extension {
	var extensions []pkix.Extension

	// Add run status extension
	statusValue := "FAILED"
	if run.Success {
		statusValue = "PASSED"
	}
	extensions = append(extensions, pkix.Extension{
		Id:    []int{1, 3, 6, 1, 4, 1, 99999, 1, 1}, // Custom OID for status
		Value: []byte(statusValue),
	})

	// Add plugin extension
	extensions = append(extensions, pkix.Extension{
		Id:    []int{1, 3, 6, 1, 4, 1, 99999, 1, 2}, // Custom OID for plugin
		Value: []byte(run.Plugin),
	})

	// Add duration extension
	if run.EndTime != nil {
		duration := run.EndTime.Sub(run.StartTime)
		extensions = append(extensions, pkix.Extension{
			Id:    []int{1, 3, 6, 1, 4, 1, 99999, 1, 3}, // Custom OID for duration
			Value: []byte(fmt.Sprintf("%.2f", duration.Seconds())),
		})
	}

	// Add key metrics
	for i, result := range results {
		if i >= 5 { // Limit to 5 key metrics
			break
		}
		extensions = append(extensions, pkix.Extension{
			Id:    []int{1, 3, 6, 1, 4, 1, 99999, 2, i + 1}, // Custom OID for metrics
			Value: []byte(fmt.Sprintf("%s:%f %s", result.Metric, result.Value, result.Unit)),
		})
	}

	return extensions
}

// Certificate represents an issued certificate
type Certificate struct {
	*x509.Certificate
	PrivateKey *rsa.PrivateKey
	RunID      int64
	IssuedAt   time.Time
}

// Save saves the certificate and key to files
func (c *Certificate) Save(certPath, keyPath string) error {
	// Save certificate
	certOut, err := os.Create(certPath) // #nosec G304 -- certPath is provided by the user and validated by the caller
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer func() { _ = certOut.Close() }()

	if err := pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.Raw,
	}); err != nil {
		return fmt.Errorf("failed to write cert: %w", err)
	}

	// Save private key if path provided
	if keyPath != "" {
		keyOut, err := os.Create(keyPath) // #nosec G304 -- keyPath is provided by the user and validated by the caller
		if err != nil {
			return fmt.Errorf("failed to create key file: %w", err)
		}
		defer func() { _ = keyOut.Close() }()

		if err := pem.Encode(keyOut, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(c.PrivateKey),
		}); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}

		// Set secure permissions on key file
		if err := os.Chmod(keyPath, 0o600); err != nil {
			return fmt.Errorf("failed to set key permissions: %w", err)
		}
	}

	return nil
}

// SavePEM returns the certificate as PEM-encoded string
func (c *Certificate) SavePEM() string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.Raw,
	}))
}

// Verify verifies a certificate against the CA
func (i *CertificateIssuer) Verify(cert *x509.Certificate) error {
	// Create certificate pool with CA
	roots := x509.NewCertPool()
	roots.AddCert(i.caCert)

	// Verify certificate
	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}
