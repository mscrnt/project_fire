# CI Platform Support for Windows-Specific Code

This document explains how the CI/CD workflows handle platform-specific code, particularly Windows-specific features like SPD reading and enhanced storage detection.

## Key Changes

### 1. MinGW Installation for Windows Builds
- All Windows CI jobs now install MinGW to support CGO
- This enables building the GUI with native Windows features
- MinGW is installed via Chocolatey package manager

### 2. Platform-Specific Build Constraints
The codebase uses Go build constraints to handle platform differences:
- `//go:build windows` for Windows-only code
- `//go:build !windows` for non-Windows stubs

Key files with platform constraints:
- `pkg/spdreader/spdreader.go` (Windows)
- `pkg/spdreader/spdreader_linux.go` (non-Windows stub)
- `pkg/gui/storage_info_windows.go` (Windows)
- `pkg/gui/storage_info_stubs.go` (non-Windows stub)

### 3. Test Strategy
- Platform-specific tests are written to handle both Windows and non-Windows behavior
- On non-Windows platforms, stub implementations return appropriate errors or empty values
- Tests check runtime.GOOS to determine expected behavior

### 4. CI Workflow Updates

#### ci.yml
- Windows test job now uses CGO_ENABLED=1 with MinGW
- Added Windows-specific test job for SPD reader and storage components
- Enhanced caching to include Go build cache directories

#### release.yml & release-packages.yml
- Windows builds now install MinGW for GUI compilation
- GUI builds use CGO_ENABLED=1 on supported platforms

### 5. Linting Configuration
Created `.golangci.yml` with:
- Platform-specific exclusions for Windows/Linux files
- SPD reader package excluded from security checks (requires admin)
- GUI platform-specific files have reduced linter strictness

## Testing Locally

### Windows
```powershell
# Install MinGW (if not already installed)
choco install mingw -y

# Run tests
$env:CGO_ENABLED = "1"
go test -v ./pkg/spdreader/...
go test -v ./pkg/gui/...
```

### Linux/macOS
```bash
# Tests will use stub implementations
go test -v ./pkg/spdreader/...
go test -v ./pkg/gui/...
```

## Known Limitations
1. SPD reading requires administrator privileges on Windows
2. Some storage detection features are Windows-only
3. GUI requires CGO on Windows but can build without it on other platforms using `no_glfw` tag