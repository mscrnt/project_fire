<div align="center">
  <img src="assets/logos/github_banner.png" alt="F.I.R.E. Banner" width="100%">
  
  # F.I.R.E.
  
  **Full Intensity Rigorous Evaluation**  
  _Ignite your hardware's true endurance._
  
  <img src="assets/logos/fire_logo_1.png" alt="F.I.R.E. Logo" width="200">
  
  [![CI Pipeline](https://github.com/mscrnt/project_fire/actions/workflows/ci.yml/badge.svg)](https://github.com/mscrnt/project_fire/actions/workflows/ci.yml)
  [![Release](https://github.com/mscrnt/project_fire/actions/workflows/release.yml/badge.svg)](https://github.com/mscrnt/project_fire/actions/workflows/release.yml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/mscrnt/project_fire)](https://goreportcard.com/report/github.com/mscrnt/project_fire)
</div>

---

## 🔥 Overview

F.I.R.E. is a single-binary, Go-powered, all-in-one PC test bench designed for burn-in tests, endurance stress, and benchmark analysis. It runs on Linux and Windows, is fully portable (USB-bootable live image & portable EXE), and integrates optional AI-driven test planning and log analysis.

## 🚀 Key Features

- **🔧 Modular Test Engine**: CPU, memory, disk I/O, 3D benchmarks, GPU compute, stability loops  
- **📅 Scheduler & Orchestrator**: One-off runs or cron-style recurring jobs  
- **📊 Data Persistence & Reporting**: SQLite logging, CSV export, HTML→PDF reports  
- **🏆 Certificate Generator**: Issue branded X.509 pass/fail certificates  
- **🌐 Remote Diagnostic Agent**: mTLS-secured REST endpoints for live sysinfo & logs  
- **🖥️ Cross-Platform GUI**: Pure-Go Fyne interface with dashboards, wizards, history, and compare views  
- **📦 Single-Binary Distribution**: Cross-compiled Go executable for Linux, Windows, macOS  
- **💿 Portable Live-USB**: Boot a minimal Linux image with persistent overlay and F.I.R.E. bundled  
- **🤖 AI-Powered Insights** (optional): Test plan generation, log analysis, OpenAI/Azure/Ollama integration

## 📸 Screenshots

*Coming soon - GUI dashboard, test results, and certificate examples*

## 🛠️ Quick Start

### Installation

F.I.R.E. is available as native packages for all major platforms. See [INSTALL.md](INSTALL.md) for detailed instructions.

#### Linux
```bash
# AppImage (Universal - Recommended)
wget https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-x86_64.AppImage
chmod +x fire-latest-x86_64.AppImage
./fire-latest-x86_64.AppImage gui

# Debian/Ubuntu
wget https://github.com/mscrnt/project_fire/releases/latest/download/fire_latest_amd64.deb
sudo apt install ./fire_latest_amd64.deb

# Fedora/RHEL
wget https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-1.x86_64.rpm
sudo dnf install fire-latest-1.x86_64.rpm
```

#### Windows
```powershell
# Download installer from releases page
# Or use PowerShell:
Invoke-WebRequest -Uri https://github.com/mscrnt/project_fire/releases/latest/download/fire-installer-latest.exe -OutFile fire-installer.exe
.\fire-installer.exe
```

#### macOS
```bash
# Download DMG from releases page
# Or use terminal:
curl -L https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-darwin-amd64.dmg -o fire.dmg
hdiutil mount fire.dmg
cp -R /Volumes/FIRE/FIRE.app /Applications/
hdiutil unmount /Volumes/FIRE
```

#### Docker
```bash
docker pull ghcr.io/mscrnt/project_fire/fire:latest
docker run --rm ghcr.io/mscrnt/project_fire/fire:latest test cpu --duration 30s
```

### Build from Source

```bash
git clone https://github.com/mscrnt/project_fire.git
cd project_fire
go build -ldflags "-s -w" -o bench ./cmd/fire
```

## 📘 Usage Examples

```bash
# Run CPU stress test
./bench test cpu --duration 5s --threads 8

# Schedule nightly memory test
./bench schedule add --name "Nightly Memory" --cron "0 2 * * *" --plugin memory

# Generate PDF report
./bench report generate --latest --format pdf

# Issue test certificate
./bench cert issue --latest

# Start remote diagnostic agent with mTLS
./bench agent serve --cert server.pem --key server.key --ca ca.pem

# Connect to remote agent
./bench agent connect --host 192.168.1.100 --endpoint sysinfo \
  --cert client.pem --key client.key --ca ca.pem
```

## 🌐 Remote Agent

The F.I.R.E. agent provides secure remote monitoring capabilities with mTLS authentication:

### Features
- **Real-time System Info**: CPU, memory, disk, and network statistics
- **Hardware Sensors**: Temperature and fan speed monitoring  
- **Log Collection**: Stream application logs remotely
- **mTLS Security**: Certificate-based mutual authentication

### Quick Start
```bash
# Initialize CA (one time)
./bench cert init

# Generate certificates (see docs/agent-certificates.md)
# Start agent on target machine
./bench agent serve --cert server.pem --key server.key --ca ca.pem

# Connect from management workstation
./bench agent connect --host target.local --endpoint sysinfo \
  --cert client.pem --key client.key --ca ca.pem --pretty
```

## 🏗️ Architecture

```
project_fire/
├── cmd/fire/          # CLI entry point
├── pkg/               # Public packages
│   ├── plugin/        # Test plugin interface
│   ├── db/            # Database layer
│   ├── schedule/      # Cron scheduler
│   ├── report/        # Report generation
│   ├── cert/          # Certificate issuance
│   └── agent/         # Remote agent
├── internal/          # Internal packages
│   └── version/       # Version information
├── assets/            # Branding and static files
│   └── logos/         # Generated logos
├── docs/              # Documentation
├── scripts/           # Build scripts
└── .github/workflows/ # CI/CD pipelines
```

## 🎨 Branding

The F.I.R.E. project features custom AI-generated branding created with Stable Diffusion. Our visual identity combines flame imagery with technology elements to represent the intense testing capabilities of the platform.

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🖥️ GUI

F.I.R.E. includes a native cross-platform GUI built with Fyne:

```bash
# Launch the GUI
./bench gui
```

### GUI Features
- **Live Dashboard**: Real-time system monitoring with charts
- **Test Wizard**: Step-by-step test configuration
- **History View**: Browse and analyze past test runs
- **Run Comparison**: Compare metrics between different runs
- **AI Insights**: Generate test plans with AI assistance
- **Certificate Manager**: Issue and verify test certificates

### GUI Requirements
- OpenGL support (most modern systems)
- Linux: libgl1-mesa-dev, xorg-dev
- Windows: No additional requirements
- macOS: No additional requirements

## 📦 Available Packages

F.I.R.E. is distributed in multiple formats for easy installation:

### Package Types
- **Linux**: AppImage (universal), .deb (Debian/Ubuntu), .rpm (Fedora/RHEL), .tar.gz
- **Windows**: NSIS installer (.exe), portable ZIP
- **macOS**: DMG disk image, PKG installer
- **Container**: Docker image on GitHub Container Registry
- **Source**: Build from source with Go 1.21+

### Verification
All releases include SHA256 checksums and optional GPG signatures. See [INSTALL.md](INSTALL.md#verification) for verification instructions.

## 🗺️ Roadmap

- [x] Phase 0: CI/CD Setup & Branding
- [x] Phase 1: Core CLI & Test Engine
- [x] Phase 2: Scheduler & Reporting
- [x] Phase 3: Remote Diagnostic Agent
- [x] Phase 4: Cross-Platform GUI
- [x] Phase 5: Packaging & Distribution
- [ ] Phase 6: AI-Powered Analysis

---

<div align="center">
  <img src="assets/logos/fire_logo_1.png" width="100" alt="F.I.R.E. Logo">
  
  **F.I.R.E.** - Delivering full-intensity, rigorous evaluations  
  Portable • Cross-platform • AI-enhanced • CI/CD-automated
</div>