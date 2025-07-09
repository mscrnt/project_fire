package cert

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// VerifyResult contains the result of certificate verification
type VerifyResult struct {
	Valid       bool
	RunID       string
	Plugin      string
	Status      string
	Duration    string
	Metrics     map[string]string
	Error       string
	Certificate *x509.Certificate
}

// VerifyCertificateFile verifies a certificate file and extracts test data
func VerifyCertificateFile(certPath, caCertPath string) (*VerifyResult, error) {
	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Load CA certificate
	caCertPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertBlock, _ := pem.Decode(caCertPEM)
	if caCertBlock == nil {
		return nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Create certificate pool with CA
	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	// Verify certificate
	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}

	result := &VerifyResult{
		Certificate: cert,
		Metrics:     make(map[string]string),
	}

	// Try to verify
	if _, err := cert.Verify(opts); err != nil {
		result.Valid = false
		result.Error = err.Error()
	} else {
		result.Valid = true
	}

	// Extract test data from certificate
	result.RunID = extractRunID(cert)

	// Extract data from extensions
	for _, ext := range cert.Extensions {
		oidString := ext.Id.String()
		value := string(ext.Value)

		switch oidString {
		case "1.3.6.1.4.1.99999.1.1": // Status
			result.Status = value
		case "1.3.6.1.4.1.99999.1.2": // Plugin
			result.Plugin = value
		case "1.3.6.1.4.1.99999.1.3": // Duration
			result.Duration = value + " seconds"
		default:
			// Check if it's a metric extension
			if strings.HasPrefix(oidString, "1.3.6.1.4.1.99999.2.") {
				parts := strings.SplitN(value, ":", 2)
				if len(parts) == 2 {
					result.Metrics[parts[0]] = parts[1]
				}
			}
		}
	}

	return result, nil
}

// extractRunID extracts the run ID from the certificate common name
func extractRunID(cert *x509.Certificate) string {
	// Common name format: "Test Run #123"
	cn := cert.Subject.CommonName
	if strings.HasPrefix(cn, "Test Run #") {
		return strings.TrimPrefix(cn, "Test Run #")
	}
	return ""
}

// FormatVerifyResult formats verification result for display
func FormatVerifyResult(result *VerifyResult) string {
	var sb strings.Builder

	sb.WriteString("Certificate Verification Result\n")
	sb.WriteString("===============================\n\n")

	if result.Valid {
		sb.WriteString("Status: VALID ✓\n")
	} else {
		sb.WriteString("Status: INVALID ✗\n")
		sb.WriteString(fmt.Sprintf("Error: %s\n", result.Error))
	}

	sb.WriteString("\nCertificate Details:\n")
	sb.WriteString(fmt.Sprintf("  Subject: %s\n", result.Certificate.Subject))
	sb.WriteString(fmt.Sprintf("  Issuer: %s\n", result.Certificate.Issuer))
	sb.WriteString(fmt.Sprintf("  Serial: %s\n", result.Certificate.SerialNumber))
	sb.WriteString(fmt.Sprintf("  Valid From: %s\n", result.Certificate.NotBefore))
	sb.WriteString(fmt.Sprintf("  Valid Until: %s\n", result.Certificate.NotAfter))

	if result.RunID != "" {
		sb.WriteString("\nTest Information:\n")
		sb.WriteString(fmt.Sprintf("  Run ID: %s\n", result.RunID))
		sb.WriteString(fmt.Sprintf("  Plugin: %s\n", result.Plugin))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", result.Status))
		sb.WriteString(fmt.Sprintf("  Duration: %s\n", result.Duration))

		if len(result.Metrics) > 0 {
			sb.WriteString("\nMetrics:\n")
			for metric, value := range result.Metrics {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", metric, value))
			}
		}
	}

	return sb.String()
}
