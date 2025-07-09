package plugin

import (
	"context"
	"encoding/json"
	"time"
)

// Params represents parameters passed to a test plugin
type Params struct {
	// Common parameters
	Duration time.Duration          `json:"duration"`
	Threads  int                    `json:"threads"`
	Config   map[string]interface{} `json:"config"`
}

// Result represents the output of a test plugin
type Result struct {
	// Timing information
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	
	// Test results
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Metrics map[string]float64     `json:"metrics"`
	Details map[string]interface{} `json:"details,omitempty"`
	
	// Raw output
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
}

// TestPlugin is the interface that all test plugins must implement
type TestPlugin interface {
	// Name returns the unique name of the plugin
	Name() string
	
	// Description returns a human-readable description
	Description() string
	
	// Run executes the test with the given parameters
	Run(ctx context.Context, params Params) (Result, error)
	
	// ValidateParams checks if the parameters are valid for this plugin
	ValidateParams(params Params) error
	
	// DefaultParams returns the default parameters for this plugin
	DefaultParams() Params
}

// MetricType represents the type of a metric
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"     // Point-in-time value
	MetricTypeCounter   MetricType = "counter"   // Cumulative value
	MetricTypeThroughput MetricType = "throughput" // Rate per second
	MetricTypeLatency   MetricType = "latency"   // Time measurement
)

// MetricInfo provides metadata about a metric
type MetricInfo struct {
	Name        string     `json:"name"`
	Type        MetricType `json:"type"`
	Unit        string     `json:"unit"`
	Description string     `json:"description"`
}

// PluginInfo provides metadata about a plugin
type PluginInfo struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Category    string       `json:"category"`
	Metrics     []MetricInfo `json:"metrics"`
	Parameters  []ParamInfo  `json:"parameters"`
}

// ParamInfo describes a parameter that a plugin accepts
type ParamInfo struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
}

// MarshalParams converts Params to JSON
func MarshalParams(p Params) ([]byte, error) {
	return json.Marshal(p)
}

// UnmarshalParams converts JSON to Params
func UnmarshalParams(data []byte) (Params, error) {
	var p Params
	err := json.Unmarshal(data, &p)
	return p, err
}