# F.I.R.E. Phase 1: Core CLI & Engine

## Overview

Phase 1 implements the foundational command-line interface and plugin engine for F.I.R.E. This phase establishes:

- A robust plugin architecture for extensible test implementations
- SQLite-based persistence for test runs and results
- CLI commands for running tests and exporting data
- Built-in CPU and memory stress test plugins

## Architecture

### Plugin System

The plugin system is built around the `TestPlugin` interface:

```go
type TestPlugin interface {
    Name() string
    Description() string
    Run(ctx context.Context, params Params) (Result, error)
    ValidateParams(params Params) error
    DefaultParams() Params
}
```

Plugins are registered at startup using Go's `init()` function:

```go
func init() {
    plugin.Register(&CPUPlugin{})
}
```

### Database Schema

The SQLite database uses two main tables:

**runs**
- `id`: Primary key
- `plugin`: Plugin name
- `params`: JSON parameters
- `start_time`, `end_time`: Timestamps
- `exit_code`, `success`: Status
- `error`, `stdout`, `stderr`: Output

**results**
- `id`: Primary key
- `run_id`: Foreign key to runs
- `metric`: Metric name
- `value`: Numeric value
- `unit`: Unit of measurement

### CLI Commands

#### test
Run a system test with a specific plugin:
```bash
bench test cpu --duration 60s --threads 4
bench test memory --config size_mb=2048
bench test --list  # List available plugins
```

#### export
Export test results in various formats:
```bash
bench export csv --run 42 --out results.csv
bench export json --run 42
bench export csv --all --out all-results.csv
```

#### list
List test runs from the database:
```bash
bench list
bench list --plugin cpu
bench list --failed --limit 10
```

#### show
Show detailed information about a run:
```bash
bench show 42
bench show 42 -v  # Include stdout/stderr
```

## Built-in Plugins

### CPU Stress Test
- **Name**: `cpu`
- **Methods**: stress-ng (if available) or native Go implementation
- **Metrics**: 
  - `bogo_ops`: Total operations (stress-ng)
  - `operations_per_second`: Throughput
  - `operations_per_thread`: Per-thread performance

### Memory Test
- **Name**: `memory`
- **Methods**: memtester (if available) or native Go implementation
- **Metrics**:
  - `allocated_mb`: Memory allocated
  - `allocation_time_ms`: Allocation latency
  - `access_operations`: Total memory accesses
  - `bandwidth_mb_per_sec`: Estimated bandwidth

## Usage Examples

### Running a CPU Stress Test
```bash
# Run CPU test for 5 minutes with 8 threads
./bench test cpu --duration 5m --threads 8

# Output:
Starting test: cpu (run ID: 1)
Duration: 5m0s, Threads: 8

Test completed in 300.05s
Success: true

Metrics:
  operations: 1234567890.00
  operations_per_second: 4115226.30 ops/s
  operations_per_thread: 154320986.25

Details:
  method: native
  threads: 8
  runtime_cpu_count: 16
```

### Exporting Results
```bash
# Export single run to CSV
./bench export csv --run 1 --out cpu-test.csv

# CSV format:
Run ID,Plugin,Start Time,End Time,Duration (s),Success,Exit Code,Metric,Value,Unit
1,cpu,2024-01-20 10:30:00,2024-01-20 10:35:00,300.050,true,0,operations,1234567890.000000,
1,cpu,2024-01-20 10:30:00,2024-01-20 10:35:00,300.050,true,0,operations_per_second,4115226.300000,ops/s
```

### Viewing Run History
```bash
# List recent runs
./bench list --limit 5

ID     Plugin          Start Time           End Time             Duration   Status
3      memory          2024-01-20 11:00:00  2024-01-20 11:01:00  60.0s      success
2      cpu             2024-01-20 10:45:00  2024-01-20 10:46:00  60.0s      success
1      cpu             2024-01-20 10:30:00  2024-01-20 10:35:00  300.1s     success
```

## Configuration

### Database Location
By default, the database is stored at `~/.fire/fire.db`. This can be overridden with the `FIRE_DB_PATH` environment variable:

```bash
export FIRE_DB_PATH=/path/to/custom/fire.db
```

### Plugin Parameters
Plugin parameters can be specified via command-line flags:

```bash
# Common parameters
--duration    Test duration (e.g., 60s, 5m, 1h)
--threads     Number of threads/workers

# Plugin-specific config
--config key=value
```

## Development Guide

### Creating a New Plugin

1. Create a new package under `pkg/plugin/`:
```go
package mytest

import (
    "context"
    "github.com/mscrnt/project_fire/pkg/plugin"
)

type MyTestPlugin struct{}

func init() {
    plugin.Register(&MyTestPlugin{})
}

func (p *MyTestPlugin) Name() string {
    return "mytest"
}

func (p *MyTestPlugin) Run(ctx context.Context, params plugin.Params) (plugin.Result, error) {
    result := plugin.Result{
        StartTime: time.Now(),
        Metrics:   make(map[string]float64),
    }
    
    // Implement test logic here
    
    result.EndTime = time.Now()
    result.Success = true
    return result, nil
}
```

2. Import the plugin in `cmd/fire/test.go`:
```go
import _ "github.com/mscrnt/project_fire/pkg/plugin/mytest"
```

3. Build and test:
```bash
go build -o bench ./cmd/fire
./bench test mytest
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/plugin

# Run with coverage
go test -v -race -coverprofile=coverage.txt ./...
```

## Next Steps

Phase 2 will add:
- Scheduled test execution with cron-style timing
- HTML/PDF report generation
- X.509 certificate issuance for pass/fail attestation
- Advanced metrics aggregation and trending