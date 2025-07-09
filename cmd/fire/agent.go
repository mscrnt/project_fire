package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mscrnt/project_fire/pkg/agent"
	"github.com/spf13/cobra"
)

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Remote diagnostic agent",
		Long:  "Manage the F.I.R.E. remote diagnostic agent for system monitoring",
	}

	cmd.AddCommand(agentServeCmd())
	cmd.AddCommand(agentConnectCmd())

	return cmd
}

func agentServeCmd() *cobra.Command {
	var (
		port     int
		certFile string
		keyFile  string
		caFile   string
		logFile  string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the diagnostic agent server",
		Long: `Start the F.I.R.E. diagnostic agent server with mTLS authentication.

The agent exposes the following endpoints:
  /sysinfo  - System information (CPU, memory, disk, network)
  /logs     - Application logs (with optional tail parameter)
  /sensors  - Hardware sensors (temperature, fans)
  /health   - Health check endpoint

Examples:
  # Start with default settings (requires cert files)
  bench agent serve --cert server.pem --key server.key --ca ca.pem

  # Start on custom port with logging
  bench agent serve --port 2223 --cert server.pem --key server.key --ca ca.pem --log agent.log

  # Using environment variables
  export FIRE_AGENT_PORT=2223
  export FIRE_AGENT_CERT=server.pem
  export FIRE_AGENT_KEY=server.key
  export FIRE_AGENT_CA=ca.pem
  bench agent serve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check environment variables for defaults
			if certFile == "" {
				certFile = os.Getenv("FIRE_AGENT_CERT")
			}
			if keyFile == "" {
				keyFile = os.Getenv("FIRE_AGENT_KEY")
			}
			if caFile == "" {
				caFile = os.Getenv("FIRE_AGENT_CA")
			}
			if envPort := os.Getenv("FIRE_AGENT_PORT"); envPort != "" && cmd.Flags().Changed("port") == false {
				fmt.Sscanf(envPort, "%d", &port)
			}

			// Create config
			config := agent.Config{
				Port:     port,
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   caFile,
				LogFile:  logFile,
			}

			// Create server
			server, err := agent.NewServer(config)
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}

			// Setup signal handling
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Start server in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Start()
			}()

			fmt.Printf("Agent server started on port %d with mTLS\n", port)
			fmt.Printf("Certificate: %s\n", certFile)
			fmt.Printf("CA: %s\n", caFile)
			fmt.Println("\nPress Ctrl+C to stop...")

			// Wait for signal or error
			select {
			case sig := <-sigChan:
				fmt.Printf("\nReceived signal: %v\n", sig)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := server.Shutdown(ctx); err != nil {
					return fmt.Errorf("shutdown error: %w", err)
				}
				fmt.Println("Server stopped gracefully")
				return nil

			case err := <-errChan:
				return fmt.Errorf("server error: %w", err)
			}
		},
	}

	cmd.Flags().IntVar(&port, "port", 2223, "Port to listen on")
	cmd.Flags().StringVar(&certFile, "cert", "", "Server certificate file (required)")
	cmd.Flags().StringVar(&keyFile, "key", "", "Server private key file (required)")
	cmd.Flags().StringVar(&caFile, "ca", "", "CA certificate file for client verification (required)")
	cmd.Flags().StringVar(&logFile, "log", "", "Log file path (optional)")

	return cmd
}

func agentConnectCmd() *cobra.Command {
	var (
		host     string
		port     int
		certFile string
		keyFile  string
		caFile   string
		endpoint string
		pretty   bool
	)

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a remote agent",
		Long: `Connect to a F.I.R.E. diagnostic agent and retrieve information.

Available endpoints:
  sysinfo  - System information
  logs     - Application logs
  sensors  - Hardware sensors
  health   - Health check

Examples:
  # Get system information
  bench agent connect --host 192.168.1.100 --endpoint sysinfo \
    --cert client.pem --key client.key --ca ca.pem

  # Get last 50 log lines
  bench agent connect --host server.local --endpoint "logs?tail=50" \
    --cert client.pem --key client.key --ca ca.pem

  # Pretty print JSON output
  bench agent connect --host 192.168.1.100 --endpoint sysinfo \
    --cert client.pem --key client.key --ca ca.pem --pretty`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check environment variables for defaults
			if certFile == "" {
				certFile = os.Getenv("FIRE_CLIENT_CERT")
			}
			if keyFile == "" {
				keyFile = os.Getenv("FIRE_CLIENT_KEY")
			}
			if caFile == "" {
				caFile = os.Getenv("FIRE_CLIENT_CA")
			}

			// Create config
			config := agent.ClientConfig{
				Host:     host,
				Port:     port,
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   caFile,
				Endpoint: endpoint,
			}

			// Create client
			client, err := agent.NewClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			// Connect and get data
			data, err := client.Connect()
			if err != nil {
				return fmt.Errorf("connection failed: %w", err)
			}

			// Output data
			if pretty && strings.HasPrefix(string(data), "{") {
				// Pretty print JSON
				var formatted interface{}
				if err := json.Unmarshal(data, &formatted); err == nil {
					prettyData, err := json.MarshalIndent(formatted, "", "  ")
					if err == nil {
						fmt.Println(string(prettyData))
						return nil
					}
				}
			}

			// Raw output
			fmt.Print(string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "localhost", "Target host")
	cmd.Flags().IntVar(&port, "port", 2223, "Target port")
	cmd.Flags().StringVar(&certFile, "cert", "", "Client certificate file (required)")
	cmd.Flags().StringVar(&keyFile, "key", "", "Client private key file (required)")
	cmd.Flags().StringVar(&caFile, "ca", "", "CA certificate file for server verification (required)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "sysinfo", "Endpoint to connect to")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Pretty print JSON output")

	return cmd
}

// generateAgentCerts generates client and server certificates for the agent
func generateAgentCerts(outputDir string) error {
	// Default CA path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	caPath := filepath.Join(homeDir, ".fire", "ca")

	fmt.Println("This command would generate agent certificates.")
	fmt.Println("For now, please use OpenSSL or another tool to generate certificates signed by the CA at:", caPath)
	fmt.Println("\nExample commands:")
	fmt.Println("  # Generate server certificate")
	fmt.Println("  openssl req -new -key server.key -out server.csr -subj \"/CN=fire-agent-server\"")
	fmt.Println("  openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.pem -days 365")
	fmt.Println("\n  # Generate client certificate")
	fmt.Println("  openssl req -new -key client.key -out client.csr -subj \"/CN=fire-agent-client\"")
	fmt.Println("  openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.pem -days 365")
	
	return nil
}