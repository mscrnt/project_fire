# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

F.I.R.E. (Full Intensity Rigorous Evaluation) is a Go-powered PC test bench designed for burn-in tests, endurance stress, and benchmark analysis. It's a single-binary, cross-platform tool with optional AI integration.

## Development Commands

### Build
```bash
go build -ldflags "-s -w" -o bench ./cmd/fire
```

### Testing
The project uses GitHub Actions for CI/CD. To run CI locally:
```bash
act -j ci
```

### Cross-Platform Builds
Uses `GOOS`/`GOARCH` with `-ldflags "-s -w"` for minimal binary size.

## Architecture

### Core Components

1. **CLI Structure** (`cmd/fire/`)
   - Commands: `test`, `schedule`, `report`, `cert`, `agent`, `ai`
   - Built with Cobra & Viper

2. **Plugin Interface**
   ```go
   type TestPlugin interface {
       Name() string
       Run(ctx context.Context, params Params) (Result, error)
   }
   ```
   Built-in plugins: CPU Stress, Memory Test, Disk I/O, 3D Benchmark, GPU Compute

3. **Key Dependencies**
   - System info: `shirou/gopsutil`, NVML Go bindings
   - Stress tools: `stress-ng`, `fio`, `glmark2`, custom CUDA/OpenCL wrappers
   - Database: `mattn/go-sqlite3`
   - Scheduler: `robfig/cron`
   - GUI: Fyne (pure-Go)
   - AI: Custom HTTP clients for OpenAI/Azure/Ollama APIs

4. **Remote Agent**
   - mTLS-secured REST endpoints
   - Live system info and log retrieval

## GitHub Actions Workflows

1. **`.github/workflows/ci.yml`** - Main CI pipeline
   - Matrix builds: ubuntu-latest, windows-latest, macos-latest
   - Linting: `go fmt`, `go vet`, `golangci-lint run`
   - Artifact names: `bench-linux-amd64`, `bench-windows-amd64.exe`, `bench-darwin-amd64`

2. **`.github/workflows/release.yml`** - Release automation
   - Triggers on GitHub release creation
   - Packages Fyne assets into binaries

3. **`.github/workflows/liveusb.yml`** - Live USB creation
   - Builds minimal Alpine/Ubuntu ISO
   - Outputs `fire-live.iso`

4. **`.github/workflows/docker-image.yml`** - Container builds (optional)

## Implementation Phases

Currently in Phase 0 (CI/CD setup). The project follows these phases:
- Phase 0: CI/CD & Automation
- Phase 1: Core CLI & Engine
- Phase 2: Scheduler & Reporting
- Phase 3: Remote Diagnostic Agent
- Phase 4: Cross-Platform GUI (Fyne)
- Phase 5: Packaging & Distribution
- Phase 6: AI-Powered Testing & Analysis

## Key Technical Decisions

- Go 1.21+ for modern features and performance
- SQLite for local data persistence
- Fyne for cross-platform GUI (single binary)
- mTLS for secure remote connections
- HTML→PDF reports via chromedp
- X.509 certificates for pass/fail attestation