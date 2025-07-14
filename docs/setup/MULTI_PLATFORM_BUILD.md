# Multi-Platform Build Guide for FIRE

This guide provides the complete solution for building FIRE GUI and CLI binaries for Linux, Windows, and macOS.

## Overview

FIRE uses the Fyne GUI framework which requires CGO and system OpenGL libraries. This means:
- ✅ CLI can be built with `CGO_ENABLED=0` (pure Go)
- ❌ GUI requires `CGO_ENABLED=1` and platform-specific dependencies

## Option 1: Native Builds on Each Platform (Recommended)

### Linux Build

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev libglfw3-dev

# Build CLI
CGO_ENABLED=0 go build -ldflags "-s -w" -o bench ./cmd/fire

# Build GUI
CGO_ENABLED=1 go build -ldflags "-s -w" -o fire-gui ./cmd/fire-gui
```

### Windows Build

```powershell
# Install MinGW (if not already installed)
choco install mingw -y

# Build CLI
$env:CGO_ENABLED = "0"
go build -ldflags "-s -w" -o bench.exe ./cmd/fire

# Build GUI
$env:CGO_ENABLED = "1"
go build -ldflags "-s -w" -o fire-gui.exe ./cmd/fire-gui
```

### macOS Build

```bash
# Install dependencies
brew install glfw

# Build CLI
CGO_ENABLED=0 go build -ldflags "-s -w" -o bench ./cmd/fire

# Build GUI (Intel)
CGO_ENABLED=1 GOARCH=amd64 go build -ldflags "-s -w" -o fire-gui ./cmd/fire-gui

# Build GUI (Apple Silicon)
CGO_ENABLED=1 GOARCH=arm64 go build -ldflags "-s -w" -o fire-gui ./cmd/fire-gui
```

## Option 2: CI/CD Build Matrix (GitHub Actions)

The project already has a comprehensive CI setup in `.github/workflows/ci.yml` that builds for all platforms:

### Supported Platforms
- ✅ Linux (amd64, arm64) - CLI only for arm64
- ✅ Windows (amd64)
- ✅ macOS (amd64, arm64) - GUI only for amd64

### To trigger builds:
```bash
git push origin main  # or create a pull request
```

Artifacts are automatically built and can be downloaded from the Actions tab.

## Option 3: Cross-Compilation (Limited)

### From Linux → Windows
```bash
# Install MinGW cross-compiler
sudo apt-get install -y gcc-mingw-w64

# Cross-compile for Windows
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC=x86_64-w64-mingw32-gcc \
  go build -ldflags "-s -w" -o fire-gui.exe ./cmd/fire-gui
```

### From Linux → macOS (CLI only)
```bash
# CLI can be cross-compiled without CGO
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
  go build -ldflags "-s -w" -o bench-darwin ./cmd/fire
```

**Note**: Cross-compiling GUI for macOS from Linux is extremely difficult due to Apple's SDK requirements.

## Option 4: Docker Multi-Stage Build

Create a `Dockerfile.multiplatform`:

```dockerfile
# Linux build stage
FROM golang:1.23 AS linux-builder
RUN apt-get update && apt-get install -y \
    gcc libgl1-mesa-dev xorg-dev libglfw3-dev
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o bench-linux ./cmd/fire
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o fire-gui-linux ./cmd/fire-gui

# Windows cross-compile stage
FROM golang:1.23 AS windows-builder
RUN apt-get update && apt-get install -y gcc-mingw-w64
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=windows go build -ldflags "-s -w" -o bench.exe ./cmd/fire
RUN CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc \
    go build -ldflags "-s -w" -o fire-gui.exe ./cmd/fire-gui

# Final stage - collect all binaries
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=linux-builder /build/bench-linux /build/fire-gui-linux ./
COPY --from=windows-builder /build/bench.exe /build/fire-gui.exe ./
```

Build with:
```bash
docker build -f Dockerfile.multiplatform -t fire-multiplatform .
docker run --rm -v $(pwd)/dist:/dist fire-multiplatform sh -c "cp * /dist/"
```

## Option 5: Local Build Script

Create `build-all-platforms.sh`:

```bash
#!/bin/bash
set -e

echo "Building FIRE for all platforms..."

# Create output directory
mkdir -p dist

# Linux build (if on Linux)
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Building Linux binaries..."
    CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/bench-linux-amd64 ./cmd/fire
    CGO_ENABLED=1 go build -ldflags "-s -w" -o dist/fire-gui-linux-amd64 ./cmd/fire-gui
fi

# macOS build (if on macOS)
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Building macOS binaries..."
    CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/bench-darwin-amd64 ./cmd/fire
    CGO_ENABLED=1 go build -ldflags "-s -w" -o dist/fire-gui-darwin-amd64 ./cmd/fire-gui
fi

# Windows build (if on Windows with Git Bash)
if [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    echo "Building Windows binaries..."
    CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/bench-windows-amd64.exe ./cmd/fire
    CGO_ENABLED=1 go build -ldflags "-s -w" -o dist/fire-gui-windows-amd64.exe ./cmd/fire-gui
fi

echo "Build complete! Check the dist/ directory for binaries."
```

## Troubleshooting

### Linux: "cannot find -lGL"
```bash
sudo apt-get install libgl1-mesa-dev
```

### Windows: "gcc not found"
```powershell
choco install mingw
# or
scoop install gcc
```

### macOS: "ld: framework not found"
```bash
xcode-select --install
brew install glfw
```

### All platforms: OpenGL version issues
Ensure your system has OpenGL 2.1 or higher:
```bash
glxinfo | grep "OpenGL version"  # Linux
```

## Best Practices

1. **Use CI/CD**: Let GitHub Actions handle multi-platform builds
2. **Version tagging**: Tag releases to trigger automated builds
3. **Test locally**: Build and test on your primary platform first
4. **Dependencies**: Document system dependencies in README
5. **Binary signing**: Consider code signing for distribution

## Quick Start

For immediate results, install dependencies and run:

```bash
# Linux
sudo apt-get update && sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev libglfw3-dev
CGO_ENABLED=1 go build -ldflags "-s -w" -o fire-gui ./cmd/fire-gui

# The existing CI will handle other platforms when you push
```

This approach ensures clean, native builds for each platform without the complexity of cross-compilation toolchains.