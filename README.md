# F.I.R.E.  
**Full Intensity Rigorous Evaluation**  
_Ignite your hardware’s true endurance._

---

## Overview  
F.I.R.E. is a single-binary, Go-powered, all-in-one PC test bench designed for burn-in tests, endurance stress, and benchmark analysis. It runs on Linux and Windows, is fully portable (USB-bootable live image & portable EXE), and integrates optional AI-driven test planning and log analysis via OpenAI, Azure OpenAI, or Ollama.

---

## Key Features  
- **Modular Test Engine**: CPU, memory, disk I/O, 3D benchmarks, GPU compute, stability loops  
- **Scheduler & Orchestrator**: One-off runs or cron-style recurring jobs  
- **Data Persistence & Reporting**: SQLite logging, CSV export, HTML→PDF reports  
- **Certificate Generator**: Issue branded X.509 pass/fail certificates  
- **Remote Diagnostic Agent**: mTLS-secured REST endpoints for live sysinfo & logs  
- **Cross-Platform GUI**: Pure-Go Fyne interface with dashboards, wizards, history, and compare views  
- **Single-Binary Distribution**: Cross-compiled Go executable for Linux, Windows, macOS  
- **Portable Live-USB**: Boot a minimal Linux image with persistent overlay and F.I.R.E. bundled  
- **AI-Powered Insights** (optional):  
  - **Test Plan Generation**  
  - **Log Analysis**  
  - Integrations: OpenAI, Azure OpenAI, Ollama local LLM  

---

## Tech Stack  

| Component                 | Technology                           |
|---------------------------|--------------------------------------|
| Language & CLI            | Go 1.21+, Cobra & Viper              |
| Sysinfo                   | `shirou/gopsutil`, NVML Go bindings  |
| Stress & Benchmarks       | Shell out to `stress-ng`, `fio`, `glmark2`, custom Go wrappers for CUDA/OpenCL |
| Database                  | `mattn/go-sqlite3`                   |
| Reporting Templates       | Jet or Go `html/template` + `chromedp` for HTML→PDF |
| Certificate Generation    | Go `crypto/x509` + OpenSSL CLI       |
| Scheduler                 | `robfig/cron`                        |
| Remote Agent              | Go `net/http` with mTLS              |
| GUI                       | Fyne (pure-Go, single binary)        |
| Cross-Compilation         | `GOOS`/`GOARCH` builds + `-ldflags "-s -w"` |
| Live-USB Image            | Minimal Ubuntu/Alpine ISO + persistence, Ventoy/Rufus |
| AI Integration            | Custom Go HTTP client for OpenAI/Azure/Ollama APIs |
| **CI/CD & Automation**    | **GitHub Actions** for build, test, cross-compile, release, and runners for Live-USB image builds |

---

## CI/CD & GitHub Actions  
All of F.I.R.E.’s build, test, packaging and release steps are automated via GitHub Actions workflows. Key pipelines:

1. **`.github/workflows/ci.yml`**  
   - **Triggers**: on `push` to `main` or PRs  
   - **Steps**:  
     1. Checkout & Go setup (matrix: `ubuntu-latest`, `windows-latest`, `macos-latest`)  
     2. Install dependencies (`stress-ng`, `fio`, headless Chromium on Linux runner)  
     3. `go fmt` & `go vet` & `golangci-lint run`  
     4. Unit tests & integration smoke tests  
     5. Build binaries (`GOOS`/`GOARCH` matrix) with `-ldflags "-s -w"`  
     6. Upload artifacts (`bench-linux-amd64`, `bench-windows-amd64.exe`, `bench-darwin-amd64`)  

2. **`.github/workflows/release.yml`**  
   - **Triggers**: on creating a **release** in GitHub  
   - **Steps**:  
     1. Checkout & Go setup  
     2. Build all targets (same matrix as CI)  
     3. Package Fyne assets into each binary  
     4. Create GitHub release draft, attach binaries and sample Live-USB ISO  

3. **`.github/workflows/liveusb.yml`**  
   - **Triggers**: manual workflow_dispatch or on tag  
   - **Steps**:  
     1. Bootstraps a minimal Alpine/Ubuntu ISO build environment  
     2. Copies the latest `bench` binary into the ISO’s `/usr/local/bin`  
     3. Configures persistence overlay  
     4. Outputs `fire-live.iso` as an artifact  

4. **`.github/workflows/docker-image.yml`** *(optional)*  
   - **Builds** a Docker image containing F.I.R.E. for container-based runs  
   - Pushes to GitHub Container Registry on release  

Each workflow uses reusable “job” templates where possible, and stores secrets (e.g. AI API keys, signing certificates) in GitHub Secrets. Artifacts are available to download automatically on each build.

---

## Roadmap & Phases

### Phase 0: CI/CD & Automation  
- Define GitHub Actions workflows (CI, release, Live-USB build, Docker image)  
- Configure linting, formatting, testing, cross-compile matrix  
- Store build artifacts & auto-publish releases

### Phase 1: Core CLI & Engine  
1. **Initialize** Go module, CLI commands (`test`, `schedule`, `report`, `cert`, `agent`, `ai`)  
2. **Plugin Interface**  
   ```go
   type TestPlugin interface {
     Name() string
     Run(ctx context.Context, params Params) (Result, error)
   }

    Built-In Plugins

        CPU Stress, Memory Test, Disk I/O, 3D Benchmark, GPU Compute

    Logging & Export

        SQLite schema & bench export csv --run <id>

Phase 2: Scheduler & Reporting

    Scheduler via robfig/cron

    HTML Reports with Jet or html/template

    PDF Generation via chromedp

    Certificate Issuance CLI

Phase 3: Remote Diagnostic Agent

    HTTP Server with mTLS

    CLI Client connect & fetch logs/sysinfo

Phase 4: Cross-Platform GUI (Fyne)

    Dashboard (live charts)

    Test Wizard

    History & Compare View

    Certificate Dialog

Phase 5: Packaging & Distribution

    Cross-Compile for all OS/ARCH targets

    Bundle Fyne assets into binary

    Live-USB Build ISO with persistence

Phase 6: AI-Powered Testing & Analysis

    AI Client Interface

    OpenAI/Azure/Azure OpenAI integration

    Ollama local LLM support

    CLI & GUI commands for plan & analysis

    Embed AI insights in reports

Installation & Build

    Clone

git clone https://github.com/your-org/fire.git
cd fire

Run CI Locally (optional)

act -j ci

Build

    go build -ldflags "-s -w" -o bench ./cmd/fire

Quick Start
CLI Examples

# Burn-in CPU + memory for 1h
./bench test cpu --duration 1h
./bench test memory --size 80% --duration 1h

# Nightly full suite
./bench schedule add --cron "0 2 * * *" --test full

# PDF report + certificate
./bench report generate --run 42 --format pdf --out run42-report.pdf
./bench cert issue --run 42 --out run42-cert.pdf

# Remote agent
./bench agent connect --host 192.168.1.55 --cert client.pem --key client.key

# AI-driven plan
./bench ai plan --spec "Ryzen 9 7950X, RTX 4080, 32 GB RAM"

GUI Launch

# Linux:
./bench gui

# Windows:
bench.exe gui

Live-USB Deployment

    Build ISO via GitHub Actions or locally with liveusb.yml workflow

    Copy bench into USB persistence

    Boot “Try F.I.R.E. Live” and launch from desktop

    F.I.R.E. delivers full-intensity, rigorous evaluations—portable, cross-platform, AI-enhanced, and CI/CD-automated from day one.