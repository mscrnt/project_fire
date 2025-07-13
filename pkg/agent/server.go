package agent

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Server represents the agent server
type Server struct {
	config     Config
	httpServer *http.Server
	logger     *log.Logger
}

// NewServer creates a new agent server
func NewServer(config Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Setup logger
	logger := log.New(os.Stdout, "[agent] ", log.LstdFlags)
	if config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger = log.New(logFile, "[agent] ", log.LstdFlags)
	}

	// Create server
	server := &Server{
		config: config,
		logger: logger,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/sysinfo", server.loggingMiddleware(sysinfoHandler))
	mux.HandleFunc("/logs", server.loggingMiddleware(logsHandler))
	mux.HandleFunc("/sensors", server.loggingMiddleware(sensorsHandler))
	mux.HandleFunc("/health", server.loggingMiddleware(healthHandler))

	// Load TLS config
	tlsConfig, err := config.LoadTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ErrorLog:     logger,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server, nil
}

// Start starts the agent server
func (s *Server) Start() error {
	s.logger.Printf("Starting agent server on port %d with mTLS", s.config.Port)

	// Note: We use ListenAndServeTLS with empty cert/key paths because
	// the certificates are already loaded in the TLS config
	err := s.httpServer.ListenAndServeTLS("", "")
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Println("Shutting down agent server...")
	return s.httpServer.Shutdown(ctx)
}

// loggingMiddleware logs incoming requests
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log client certificate info
		clientCert := "none"
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			clientCert = r.TLS.PeerCertificates[0].Subject.CommonName
		}

		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the handler
		next(wrapped, r)

		// Log the request
		s.logger.Printf("%s %s %d %s client=%s duration=%s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			r.RemoteAddr,
			clientCert,
			time.Since(start),
		)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// healthHandler returns server health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "OK\n")
}
