package memory

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mscrnt/project_fire/pkg/plugin"
)

func init() {
	// Register the memory test plugin
	if err := plugin.Register(&Plugin{}); err != nil {
		// Since init() can't return an error, we panic on registration failure
		// This is acceptable because plugin registration is a critical startup operation
		panic(fmt.Sprintf("failed to register memory plugin: %v", err))
	}
}

// Plugin implements memory stress testing
type Plugin struct{}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "memory"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Memory stress test using memtester or native Go implementation"
}

// ValidateParams validates the parameters
func (p *Plugin) ValidateParams(params plugin.Params) error {
	if params.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}

	// Check memory size parameter
	if _, ok := params.Config["size_mb"]; !ok {
		// Default to 1GB if not specified
		params.Config["size_mb"] = 1024
	}

	return nil
}

// DefaultParams returns default parameters
func (p *Plugin) DefaultParams() plugin.Params {
	return plugin.Params{
		Duration: 60 * time.Second,
		Threads:  1,
		Config: map[string]interface{}{
			"method":  "auto",   // auto, memtester, native
			"size_mb": 1024,     // memory size in MB
			"pattern": "random", // fill pattern: zero, random, sequential
		},
	}
}

// Run executes the memory stress test
func (p *Plugin) Run(ctx context.Context, params plugin.Params) (plugin.Result, error) {
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

	// Try memtester first if available
	if method == "auto" || method == "memtester" {
		if err := p.runMemtester(ctx, params, &result); err == nil {
			return result, nil
		} else if method == "memtester" {
			// If specifically requested memtester and it failed, return error
			result.EndTime = time.Now()
			result.Success = false
			result.Error = fmt.Sprintf("memtester failed: %v", err)
			return result, err
		}
		// Fall back to native implementation
		result.Details["fallback"] = "memtester not available, using native implementation"
	}

	// Use native Go implementation
	return p.runNative(ctx, params, &result)
}

// runMemtester runs the memtester tool
func (p *Plugin) runMemtester(ctx context.Context, params plugin.Params, result *plugin.Result) error {
	// Check if memtester is available
	if _, err := exec.LookPath("memtester"); err != nil {
		return fmt.Errorf("memtester not found in PATH")
	}

	// Get memory size
	sizeMB := 1024
	if s, ok := params.Config["size_mb"].(int); ok {
		sizeMB = s
	} else if s, ok := params.Config["size_mb"].(float64); ok {
		sizeMB = int(s)
	}

	// Calculate iterations based on duration
	// Memtester takes about 1 minute per iteration for 1GB
	iterations := int(params.Duration.Minutes())
	if iterations < 1 {
		iterations = 1
	}

	// Build command
	args := []string{
		fmt.Sprintf("%dM", sizeMB),
		strconv.Itoa(iterations),
	}

	// Create command with timeout
	ctx, cancel := context.WithTimeout(ctx, params.Duration+30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "memtester", args...)

	// Run command and capture output
	output, err := cmd.CombinedOutput()
	result.Stdout = string(output)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil && ctx.Err() != context.DeadlineExceeded {
		result.Success = false
		result.Error = err.Error()
		return err
	}

	// Parse metrics from output
	p.parseMemtesterMetrics(string(output), result)

	result.Success = true
	result.Details["method"] = "memtester"
	result.Details["command"] = strings.Join(append([]string{"memtester"}, args...), " ")
	result.Details["size_mb"] = sizeMB

	return nil
}

// parseMemtesterMetrics parses metrics from memtester output
func (p *Plugin) parseMemtesterMetrics(output string, result *plugin.Result) {
	lines := strings.Split(output, "\n")
	testsRun := 0
	testsPassed := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for test results
		if strings.Contains(line, "ok") {
			testsRun++
			testsPassed++
		} else if strings.Contains(line, "FAILURE") {
			testsRun++
		}
	}

	result.Metrics["tests_run"] = float64(testsRun)
	result.Metrics["tests_passed"] = float64(testsPassed)
	if testsRun > 0 {
		result.Metrics["pass_rate"] = float64(testsPassed) / float64(testsRun) * 100
	}
}

// runNative runs a native Go memory stress test
func (p *Plugin) runNative(ctx context.Context, params plugin.Params, result *plugin.Result) (plugin.Result, error) {
	// Get memory size
	sizeMB := 1024
	if s, ok := params.Config["size_mb"].(int); ok {
		sizeMB = s
	} else if s, ok := params.Config["size_mb"].(float64); ok {
		sizeMB = int(s)
	}

	// Get pattern
	pattern := "random"
	if p, ok := params.Config["pattern"].(string); ok {
		pattern = p
	}

	// Allocate memory blocks
	blockSize := 1024 * 1024 // 1MB blocks
	numBlocks := sizeMB
	blocks := make([][]byte, numBlocks)

	// Allocate and fill memory
	var wg sync.WaitGroup
	errors := make(chan error, numBlocks)
	allocStart := time.Now()

	for i := 0; i < numBlocks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Allocate block
			blocks[idx] = make([]byte, blockSize)

			// Fill block based on pattern
			switch pattern {
			case "zero":
				// Already zero-filled
			case "sequential":
				for j := 0; j < blockSize; j++ {
					blocks[idx][j] = byte(j % 256)
				}
			case "random":
				// Fill with pseudo-random data
				for j := 0; j < blockSize; j++ {
					blocks[idx][j] = byte((idx*blockSize + j) * 2654435761 % 256)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for allocation errors
	for err := range errors {
		if err != nil {
			result.EndTime = time.Now()
			result.Success = false
			result.Error = fmt.Sprintf("memory allocation failed: %v", err)
			return *result, err
		}
	}

	allocDuration := time.Since(allocStart)
	result.Metrics["allocation_time_ms"] = float64(allocDuration.Milliseconds())
	result.Metrics["allocated_mb"] = float64(sizeMB)

	// Perform memory access patterns for remaining duration
	accessStart := time.Now()
	accessCount := int64(0)
	done := make(chan struct{})

	// Start workers to access memory
	numWorkers := runtime.NumCPU()
	if params.Threads > 0 {
		numWorkers = params.Threads
	}

	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					return
				default:
					// Access random blocks
					blockIdx := int(time.Now().UnixNano()) % numBlocks
					if blockIdx < len(blocks) && blocks[blockIdx] != nil {
						// Read and write to trigger memory access
						sum := 0
						for i := 0; i < 1024; i++ {
							idx := (i * 1024) % blockSize
							sum += int(blocks[blockIdx][idx])
							blocks[blockIdx][idx] = byte((sum + workerID) % 256)
						}
						accessCount++
					}
				}
			}
		}(w)
	}

	// Wait for duration or context cancellation
	remaining := params.Duration - allocDuration
	if remaining > 0 {
		select {
		case <-time.After(remaining):
		case <-ctx.Done():
		}
	}

	close(done)
	time.Sleep(100 * time.Millisecond) // Allow workers to finish

	accessDuration := time.Since(accessStart)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Calculate metrics
	result.Metrics["access_operations"] = float64(accessCount)
	result.Metrics["access_rate_ops_per_sec"] = float64(accessCount) / accessDuration.Seconds()
	result.Metrics["bandwidth_mb_per_sec"] = (float64(accessCount) * 1024 * 1024) / (accessDuration.Seconds() * 1024 * 1024)

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	result.Metrics["heap_alloc_mb"] = float64(m.HeapAlloc) / 1024 / 1024
	result.Metrics["sys_memory_mb"] = float64(m.Sys) / 1024 / 1024

	result.Success = true
	result.Details["method"] = "native"
	result.Details["pattern"] = pattern
	result.Details["workers"] = numWorkers
	result.Details["blocks"] = numBlocks

	return *result, nil
}

// Info returns detailed plugin information
func (p *Plugin) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        p.Name(),
		Description: p.Description(),
		Category:    "stress",
		Metrics: []plugin.MetricInfo{
			{
				Name:        "tests_run",
				Type:        plugin.MetricTypeCounter,
				Unit:        "tests",
				Description: "Number of memory tests run (memtester)",
			},
			{
				Name:        "tests_passed",
				Type:        plugin.MetricTypeCounter,
				Unit:        "tests",
				Description: "Number of memory tests passed (memtester)",
			},
			{
				Name:        "pass_rate",
				Type:        plugin.MetricTypeGauge,
				Unit:        "%",
				Description: "Percentage of tests passed (memtester)",
			},
			{
				Name:        "allocated_mb",
				Type:        plugin.MetricTypeGauge,
				Unit:        "MB",
				Description: "Amount of memory allocated",
			},
			{
				Name:        "allocation_time_ms",
				Type:        plugin.MetricTypeLatency,
				Unit:        "ms",
				Description: "Time taken to allocate memory",
			},
			{
				Name:        "access_operations",
				Type:        plugin.MetricTypeCounter,
				Unit:        "operations",
				Description: "Number of memory access operations",
			},
			{
				Name:        "access_rate_ops_per_sec",
				Type:        plugin.MetricTypeThroughput,
				Unit:        "ops/s",
				Description: "Memory access operations per second",
			},
			{
				Name:        "bandwidth_mb_per_sec",
				Type:        plugin.MetricTypeThroughput,
				Unit:        "MB/s",
				Description: "Estimated memory bandwidth",
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
				Name:        "size_mb",
				Type:        "integer",
				Default:     1024,
				Description: "Amount of memory to test in MB",
				Required:    false,
			},
			{
				Name:        "pattern",
				Type:        "string",
				Default:     "random",
				Description: "Memory fill pattern: zero, random, or sequential",
				Required:    false,
			},
			{
				Name:        "method",
				Type:        "string",
				Default:     "auto",
				Description: "Test method: auto, memtester, or native",
				Required:    false,
			},
		},
	}
}
