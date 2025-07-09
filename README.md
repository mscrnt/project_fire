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

<div align="center">
  <img src="assets/logos/fire_logo_2.png" width="200" alt="F.I.R.E. Logo">
</div>

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

### Download Pre-built Binary

```bash
# Linux
wget https://github.com/mscrnt/project_fire/releases/latest/download/bench-linux-amd64
chmod +x bench-linux-amd64
sudo mv bench-linux-amd64 /usr/local/bin/bench

# macOS
wget https://github.com/mscrnt/project_fire/releases/latest/download/bench-darwin-amd64
chmod +x bench-darwin-amd64
sudo mv bench-darwin-amd64 /usr/local/bin/bench

# Windows
# Download bench-windows-amd64.exe and add to PATH
```

### Build from Source

```bash
git clone https://github.com/mscrnt/project_fire.git
cd project_fire
go build -ldflags "-s -w" -o bench ./cmd/fire
```

## 📘 Usage Examples

```bash
# Burn-in CPU + memory for 1h
./bench test cpu --duration 1h
./bench test memory --size 80% --duration 1h

# Schedule nightly full test suite
./bench schedule add --cron "0 2 * * *" --test full

# Generate PDF report + certificate
./bench report generate --run 42 --format pdf --out run42-report.pdf
./bench cert issue --run 42 --out run42-cert.pdf

# Start remote diagnostic agent
./bench agent --port 8080

# AI-driven test planning
./bench ai plan --spec "Ryzen 9 7950X, RTX 4080, 32 GB RAM"
```

## 🏗️ Architecture

```
project_fire/
├── cmd/fire/          # CLI entry point
├── internal/          # Internal packages
│   ├── version/       # Version information
│   ├── tests/         # Test implementations
│   ├── report/        # Report generation
│   └── agent/         # Web agent
├── assets/            # Branding and static files
│   └── assets/logos/         # Generated logos
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

## 🗺️ Roadmap

- [x] Phase 0: CI/CD Setup & Branding
- [ ] Phase 1: Core CLI & Test Engine
- [ ] Phase 2: Scheduler & Reporting
- [ ] Phase 3: Remote Diagnostic Agent
- [ ] Phase 4: Cross-Platform GUI
- [ ] Phase 5: Packaging & Distribution
- [ ] Phase 6: AI-Powered Analysis

---

<div align="center">
  <img src="assets/logos/fire_logo_1.png" width="100" alt="F.I.R.E. Logo">
  
  **F.I.R.E.** - Delivering full-intensity, rigorous evaluations  
  Portable • Cross-platform • AI-enhanced • CI/CD-automated
</div>