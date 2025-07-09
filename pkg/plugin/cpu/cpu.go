package cpu

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mscrnt/project_fire/pkg/plugin"
)

func init() {
	// Register the CPU stress test plugin
	plugin.Register(&CPUPlugin{})
}

// CPUPlugin implements CPU stress testing
type CPUPlugin struct{}

// Name returns the plugin name
func (p *CPUPlugin) Name() string {
	return "cpu"
}

// Description returns the plugin description
func (p *CPUPlugin) Description() string {
	return "CPU stress test using stress-ng or native Go implementation"
}

// ValidateParams validates the parameters
func (p *CPUPlugin) ValidateParams(params plugin.Params) error {
	if params.Threads <= 0 {
		params.Threads = runtime.NumCPU()
	}

	if params.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}

	return nil
}

// DefaultParams returns default parameters
func (p *CPUPlugin) DefaultParams() plugin.Params {
	return plugin.Params{
		Duration: 60 * time.Second,
		Threads:  runtime.NumCPU(),
		Config: map[string]interface{}{
			"method": "auto", // auto, stress-ng, native
			"load":   100,    // target CPU load percentage
		},
	}
}

// Run executes the CPU stress test
func (p *CPUPlugin) Run(ctx context.Context, params plugin.Params) (plugin.Result, error) {
	result := plugin.Result{
		StartTime: time.Now(),
		Metrics:   make(map[string]float64),
		Details:   make(map[string]interface{}),
	}

	// Validate parameters
	if err := p.ValidateParams(params); err != nil {
		result.EndTime = time.Now()
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	// Get method from config
	method := "auto"
	if m, ok := params.Config["method"].(string); ok {
		method = m
	}

	// Try stress-ng first if available
	if method == "auto" || method == "stress-ng" {
		if err := p.runStressNG(ctx, params, &result); err == nil {
			return result, nil
		} else if method == "stress-ng" {
			// If specifically requested stress-ng and it failed, return error
			result.EndTime = time.Now()
			result.Success = false
			result.Error = fmt.Sprintf("stress-ng failed: %v", err)
			return result, err
		}
		// Fall back to native implementation
		result.Details["fallback"] = "stress-ng not available, using native implementation"
	}

	// Use native Go implementation
	return p.runNative(ctx, params, &result)
}

// runStressNG runs the stress-ng tool
func (p *CPUPlugin) runStressNG(ctx context.Context, params plugin.Params, result *plugin.Result) error {
	// Check if stress-ng is available
	if _, err := exec.LookPath("stress-ng"); err != nil {
		return fmt.Errorf("stress-ng not found in PATH")
	}

	// Build command
	args := []string{
		"--cpu", strconv.Itoa(params.Threads),
		"--timeout", fmt.Sprintf("%ds", int(params.Duration.Seconds())),
		"--metrics-brief",
	}

	// Add CPU method if specified
	if method, ok := params.Config["cpu-method"].(string); ok {
		args = append(args, "--cpu-method", method)
	}

	// Create command
	cmd := exec.CommandContext(ctx, "stress-ng", args...)

	// Run command and capture output
	output, err := cmd.CombinedOutput()
	result.Stdout = string(output)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return err
	}

	// Parse metrics from output
	p.parseStressNGMetrics(string(output), result)

	result.Success = true
	result.Details["method"] = "stress-ng"
	result.Details["command"] = strings.Join(append([]string{"stress-ng"}, args...), " ")

	return nil
}

// parseStressNGMetrics parses metrics from stress-ng output
func (p *CPUPlugin) parseStressNGMetrics(output string, result *plugin.Result) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for bogo ops
		if strings.Contains(line, "cpu") && strings.Contains(line, "bogo ops") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "ops" && i > 0 {
					if ops, err := strconv.ParseFloat(parts[i-1], 64); err == nil {
						result.Metrics["bogo_ops"] = ops
					}
				}
				if part == "ops/s" && i > 0 {
					if opsPerSec, err := strconv.ParseFloat(parts[i-1], 64); err == nil {
						result.Metrics["bogo_ops_per_second"] = opsPerSec
					}
				}
			}
		}
	}
}

// runNative runs a native Go CPU stress test
func (p *CPUPlugin) runNative(ctx context.Context, params plugin.Params, result *plugin.Result) (plugin.Result, error) {
	// Create done channel
	done := make(chan struct{})
	operations := make(chan int64, params.Threads)

	// Start worker goroutines
	for i := 0; i < params.Threads; i++ {
		go func() {
			ops := int64(0)
			for {
				select {
				case <-done:
					operations <- ops
					return
				default:
					// CPU-intensive operation
					for j := 0; j < 1000; j++ {
						_ = j * j * j
					}
					ops++
				}
			}
		}()
	}

	// Wait for duration or context cancellation
	select {
	case <-time.After(params.Duration):
	case <-ctx.Done():
	}

	// Stop workers
	close(done)

	// Collect operations count
	totalOps := int64(0)
	for i := 0; i < params.Threads; i++ {
		totalOps += <-operations
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Calculate metrics
	result.Metrics["operations"] = float64(totalOps)
	result.Metrics["operations_per_second"] = float64(totalOps) / result.Duration.Seconds()
	result.Metrics["operations_per_thread"] = float64(totalOps) / float64(params.Threads)

	result.Success = true
	result.Details["method"] = "native"
	result.Details["threads"] = params.Threads
	result.Details["runtime_cpu_count"] = runtime.NumCPU()

	return *result, nil
}

// Info returns detailed plugin information
func (p *CPUPlugin) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        p.Name(),
		Description: p.Description(),
		Category:    "stress",
		Metrics: []plugin.MetricInfo{
			{
				Name:        "bogo_ops",
				Type:        plugin.MetricTypeCounter,
				Unit:        "operations",
				Description: "Total bogus operations performed (stress-ng)",
			},
			{
				Name:        "bogo_ops_per_second",
				Type:        plugin.MetricTypeThroughput,
				Unit:        "ops/s",
				Description: "Bogus operations per second (stress-ng)",
			},
			{
				Name:        "operations",
				Type:        plugin.MetricTypeCounter,
				Unit:        "operations",
				Description: "Total operations performed (native)",
			},
			{
				Name:        "operations_per_second",
				Type:        plugin.MetricTypeThroughput,
				Unit:        "ops/s",
				Description: "Operations per second (native)",
			},
		},
		Parameters: []plugin.ParamInfo{
			{
				Name:        "duration",
				Type:        "duration",
				Default:     "60s",
				Description: "Test duration",
				Required:    true,
			},
			{
				Name:        "threads",
				Type:        "integer",
				Default:     runtime.NumCPU(),
				Description: "Number of CPU stress threads",
				Required:    false,
			},
			{
				Name:        "method",
				Type:        "string",
				Default:     "auto",
				Description: "Stress method: auto, stress-ng, or native",
				Required:    false,
			},
		},
	}
}
