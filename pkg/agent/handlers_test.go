package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSysinfoHandler(t *testing.T) {
	// Create request
	req, err := http.NewRequest("GET", "/sysinfo", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := http.HandlerFunc(sysinfoHandler)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check content type
	expected := "application/json"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("handler returned wrong content type: got %v want %v",
			ct, expected)
	}

	// Parse response
	var info SysInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &info); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Validate response
	if info.Timestamp.IsZero() {
		t.Error("timestamp is zero")
	}

	if info.Host.Hostname == "" {
		t.Error("hostname is empty")
	}

	if info.CPU.LogicalCores == 0 {
		t.Error("CPU logical cores is 0")
	}

	if info.Memory.Total == 0 {
		t.Error("memory total is 0")
	}
}

func TestSysinfoHandlerMethods(t *testing.T) {
	tests := []struct {
		method     string
		wantStatus int
	}{
		{"GET", http.StatusOK},
		{"POST", http.StatusMethodNotAllowed},
		{"PUT", http.StatusMethodNotAllowed},
		{"DELETE", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/sysinfo", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(sysinfoHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}
		})
	}
}

func TestLogsHandler(t *testing.T) {
	// Create request
	req, err := http.NewRequest("GET", "/logs", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := http.HandlerFunc(logsHandler)
	handler.ServeHTTP(rr, req)

	// For this test, we expect either OK or NotFound (if log file doesn't exist)
	if status := rr.Code; status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("handler returned unexpected status code: got %v", status)
	}

	// If OK, check response format
	if rr.Code == http.StatusOK {
		var logs LogsResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &logs); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if logs.File == "" {
			t.Error("log file name is empty")
		}

		if logs.Timestamp.IsZero() {
			t.Error("timestamp is zero")
		}
	}
}

func TestLogsHandlerQueryParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"default", "", http.StatusNotFound}, // Default file likely doesn't exist
		{"with tail", "?tail=50", http.StatusNotFound},
		{"invalid file", "?file=../etc/passwd", http.StatusBadRequest},
		{"directory traversal", "?file=../../secret", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/logs"+tt.query, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(logsHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}
		})
	}
}

func TestSensorsHandler(t *testing.T) {
	// Create request
	req, err := http.NewRequest("GET", "/sensors", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := http.HandlerFunc(sensorsHandler)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Parse response
	var sensors SensorsInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &sensors); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Validate response
	if sensors.Timestamp.IsZero() {
		t.Error("timestamp is zero")
	}

	// Note: sensor data might be empty on test systems, so we just check structure
	if sensors.Temperature == nil {
		t.Error("temperature array is nil")
	}

	if sensors.Fans == nil {
		t.Error("fans array is nil")
	}
}

func TestHealthHandler(t *testing.T) {
	// Create request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check content type
	expected := "text/plain"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("handler returned wrong content type: got %v want %v",
			ct, expected)
	}

	// Check body
	expectedBody := "OK\n"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v",
			body, expectedBody)
	}
}

func TestResponseWriter(t *testing.T) {
	// Test the responseWriter wrapper
	original := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: original, statusCode: http.StatusOK}

	// Test default status code
	if wrapped.statusCode != http.StatusOK {
		t.Errorf("default status code wrong: got %v want %v",
			wrapped.statusCode, http.StatusOK)
	}

	// Test WriteHeader
	wrapped.WriteHeader(http.StatusNotFound)
	if wrapped.statusCode != http.StatusNotFound {
		t.Errorf("status code not updated: got %v want %v",
			wrapped.statusCode, http.StatusNotFound)
	}

	// Verify it was passed through
	if original.Code != http.StatusNotFound {
		t.Errorf("status code not passed through: got %v want %v",
			original.Code, http.StatusNotFound)
	}
}

// BenchmarkSysinfoHandler benchmarks the sysinfo handler
func BenchmarkSysinfoHandler(b *testing.B) {
	req, err := http.NewRequest("GET", "/sysinfo", nil)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(sysinfoHandler)
		handler.ServeHTTP(rr, req)
	}
}
