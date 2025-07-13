package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mscrnt/project_fire/pkg/cert"
	"github.com/mscrnt/project_fire/pkg/db"
	"github.com/spf13/cobra"
)

func certCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "Certificate management",
		Long:  "Issue and verify certificates for test results",
	}

	cmd.AddCommand(certInitCmd())
	cmd.AddCommand(certIssueCmd())
	cmd.AddCommand(certVerifyCmd())

	return cmd
}

func certInitCmd() *cobra.Command {
	var (
		caPath string
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize certificate authority",
		Long: `Initialize a certificate authority (CA) for signing test certificates.

This command creates a self-signed CA certificate and private key that will be
used to sign individual test result certificates.

Examples:
  # Initialize CA in default location
  bench cert init

  # Initialize CA in custom location
  bench cert init --ca-path /path/to/ca

  # Force overwrite existing CA
  bench cert init --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default CA path
			if caPath == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}
				caPath = filepath.Join(homeDir, ".fire", "ca")
			}

			// Create directory
			if err := os.MkdirAll(caPath, 0o700); err != nil {
				return fmt.Errorf("failed to create CA directory: %w", err)
			}

			certPath := filepath.Join(caPath, "ca.crt")
			keyPath := filepath.Join(caPath, "ca.key")

			// Check if CA already exists
			if !force {
				if _, err := os.Stat(certPath); err == nil {
					return fmt.Errorf("CA certificate already exists at %s (use --force to overwrite)", certPath)
				}
			}

			// Create new CA
			issuer, err := cert.NewCertificateIssuer()
			if err != nil {
				return fmt.Errorf("failed to create CA: %w", err)
			}

			// Save CA files
			if err := issuer.SaveCA(certPath, keyPath); err != nil {
				return fmt.Errorf("failed to save CA: %w", err)
			}

			fmt.Println("Certificate Authority initialized successfully")
			fmt.Printf("CA Certificate: %s\n", certPath)
			fmt.Printf("CA Private Key: %s\n", keyPath)
			fmt.Println("\nIMPORTANT: Keep the private key secure and backed up!")

			return nil
		},
	}

	cmd.Flags().StringVar(&caPath, "ca-path", "", "Path to CA directory")
	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing CA")

	return cmd
}

func certIssueCmd() *cobra.Command {
	var (
		runID     int64
		latest    bool
		plugin    string
		output    string
		keyOutput string
		caPath    string
	)

	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue a certificate for test results",
		Long: `Issue a certificate attesting to test results.

The certificate contains cryptographically signed test data including:
- Test status (PASSED/FAILED)
- Plugin used
- Test duration
- Key metrics

Examples:
  # Issue certificate for latest run
  bench cert issue --latest

  # Issue certificate for specific run
  bench cert issue --run 42

  # Issue certificate with custom output
  bench cert issue --run 42 --output test-cert.pem --key test-key.pem`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if !latest && runID == 0 {
				return fmt.Errorf("either --latest or --run must be specified")
			}

			// Default CA path
			if caPath == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}
				caPath = filepath.Join(homeDir, ".fire", "ca")
			}

			// Load CA
			certPath := filepath.Join(caPath, "ca.crt")
			keyPath := filepath.Join(caPath, "ca.key")

			issuer, err := cert.LoadCA(certPath, keyPath)
			if err != nil {
				return fmt.Errorf("failed to load CA (run 'bench cert init' first): %w", err)
			}

			// Open database
			dbPath := getDBPath()
			database, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = database.Close() }()

			// Find run ID
			if latest {
				runs, err := database.ListRuns(db.RunFilter{
					Plugin: plugin,
					Limit:  1,
				})
				if err != nil {
					return fmt.Errorf("failed to list runs: %w", err)
				}
				if len(runs) == 0 {
					return fmt.Errorf("no runs found")
				}
				runID = runs[0].ID
			}

			// Get run and results
			run, err := database.GetRun(runID)
			if err != nil {
				return fmt.Errorf("run %d not found", runID)
			}

			results, err := database.GetResults(runID)
			if err != nil {
				return fmt.Errorf("failed to get results: %w", err)
			}

			// Issue certificate
			certificate, err := issuer.IssueCertificate(run, results)
			if err != nil {
				return fmt.Errorf("failed to issue certificate: %w", err)
			}

			// Generate output filenames if not specified
			if output == "" {
				timestamp := time.Now().Format("20060102_150405")
				output = fmt.Sprintf("fire_cert_%d_%s.pem", runID, timestamp)
			}

			// Save certificate
			if err := certificate.Save(output, keyOutput); err != nil {
				return fmt.Errorf("failed to save certificate: %w", err)
			}

			// Display information
			fmt.Printf("Certificate issued for run #%d\n", runID)
			fmt.Printf("Plugin: %s\n", run.Plugin)
			fmt.Printf("Status: %s\n", formatStatus(run.Success))
			fmt.Printf("Certificate: %s\n", output)
			if keyOutput != "" {
				fmt.Printf("Private Key: %s\n", keyOutput)
			}

			// Show certificate details
			fmt.Printf("\nCertificate Details:\n")
			fmt.Printf("  Subject: %s\n", certificate.Subject)
			fmt.Printf("  Serial: %s\n", certificate.SerialNumber)
			fmt.Printf("  Valid From: %s\n", certificate.NotBefore.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Valid Until: %s\n", certificate.NotAfter.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	cmd.Flags().Int64Var(&runID, "run", 0, "Run ID to issue certificate for")
	cmd.Flags().BoolVar(&latest, "latest", false, "Use latest run")
	cmd.Flags().StringVarP(&plugin, "plugin", "p", "", "Filter by plugin when using --latest")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output certificate file")
	cmd.Flags().StringVar(&keyOutput, "key", "", "Output private key file (optional)")
	cmd.Flags().StringVar(&caPath, "ca-path", "", "Path to CA directory")

	return cmd
}

func certVerifyCmd() *cobra.Command {
	var (
		caPath string
	)

	cmd := &cobra.Command{
		Use:   "verify [certificate]",
		Short: "Verify a test certificate",
		Long: `Verify a test certificate and display its contents.

This command verifies the certificate signature against the CA and extracts
the embedded test information.

Examples:
  # Verify a certificate
  bench cert verify test-cert.pem

  # Verify with custom CA path
  bench cert verify test-cert.pem --ca-path /path/to/ca`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			certFile := args[0]

			// Default CA path
			if caPath == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}
				caPath = filepath.Join(homeDir, ".fire", "ca")
			}

			// Verify certificate
			caCertPath := filepath.Join(caPath, "ca.crt")
			result, err := cert.VerifyCertificateFile(certFile, caCertPath)
			if err != nil {
				return fmt.Errorf("failed to verify certificate: %w", err)
			}

			// Display result
			fmt.Println(cert.FormatVerifyResult(result))

			// Exit with error code if invalid
			if !result.Valid {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&caPath, "ca-path", "", "Path to CA directory")

	return cmd
}
