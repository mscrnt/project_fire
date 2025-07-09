#!/bin/bash
# Test script for verifying package creation

set -e

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo "F.I.R.E. Package Testing Script"
echo "==============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "OK" ]; then
        echo -e "${GREEN}[✓]${NC} $message"
    elif [ "$status" = "FAIL" ]; then
        echo -e "${RED}[✗]${NC} $message"
    else
        echo -e "${YELLOW}[!]${NC} $message"
    fi
}

# Check if binaries exist
check_binaries() {
    echo ""
    echo "Checking binaries..."
    
    if [ -f "$PROJECT_ROOT/bench" ] || [ -f "$PROJECT_ROOT/bench.exe" ]; then
        print_status "OK" "CLI binary found"
    else
        print_status "FAIL" "CLI binary not found"
        return 1
    fi
    
    if [ -f "$PROJECT_ROOT/fire-gui" ] || [ -f "$PROJECT_ROOT/fire-gui.exe" ]; then
        print_status "OK" "GUI binary found"
    else
        print_status "FAIL" "GUI binary not found"
        return 1
    fi
}

# Test Linux packaging
test_linux_packages() {
    echo ""
    echo "Testing Linux packages..."
    
    if [ "$(uname -s)" != "Linux" ]; then
        print_status "SKIP" "Not on Linux, skipping Linux package tests"
        return
    fi
    
    # Check for packaging tools
    if command -v appimagetool >/dev/null 2>&1; then
        print_status "OK" "appimagetool available"
    else
        print_status "WARN" "appimagetool not found - AppImage creation will fail"
    fi
    
    if command -v dpkg-deb >/dev/null 2>&1; then
        print_status "OK" "dpkg-deb available"
    else
        print_status "WARN" "dpkg-deb not found - DEB creation will fail"
    fi
    
    if command -v fpm >/dev/null 2>&1; then
        print_status "OK" "fpm available"
    elif command -v alien >/dev/null 2>&1; then
        print_status "OK" "alien available (RPM via conversion)"
    else
        print_status "WARN" "Neither fpm nor alien found - RPM creation will fail"
    fi
}

# Test Windows packaging
test_windows_packages() {
    echo ""
    echo "Testing Windows packages..."
    
    if [ "$(uname -s)" != "MINGW"* ] && [ "$(uname -s)" != "MSYS"* ]; then
        print_status "SKIP" "Not on Windows, skipping Windows package tests"
        return
    fi
    
    # Check for NSIS
    if command -v makensis >/dev/null 2>&1; then
        print_status "OK" "NSIS available"
    else
        print_status "WARN" "NSIS not found - installer creation will fail"
    fi
}

# Test macOS packaging
test_macos_packages() {
    echo ""
    echo "Testing macOS packages..."
    
    if [ "$(uname -s)" != "Darwin" ]; then
        print_status "SKIP" "Not on macOS, skipping macOS package tests"
        return
    fi
    
    # Check for macOS tools
    if command -v hdiutil >/dev/null 2>&1; then
        print_status "OK" "hdiutil available"
    else
        print_status "WARN" "hdiutil not found - DMG creation will fail"
    fi
    
    if command -v pkgbuild >/dev/null 2>&1; then
        print_status "OK" "pkgbuild available"
    else
        print_status "WARN" "pkgbuild not found - PKG creation will fail"
    fi
    
    if command -v codesign >/dev/null 2>&1; then
        print_status "OK" "codesign available"
    else
        print_status "WARN" "codesign not found - signing will fail"
    fi
}

# Test Docker build
test_docker() {
    echo ""
    echo "Testing Docker..."
    
    if command -v docker >/dev/null 2>&1; then
        print_status "OK" "Docker available"
        
        # Check if Dockerfile exists
        if [ -f "$PROJECT_ROOT/Dockerfile" ]; then
            print_status "OK" "Dockerfile found"
        else
            print_status "FAIL" "Dockerfile not found"
        fi
    else
        print_status "WARN" "Docker not available"
    fi
}

# Test packaging scripts
test_scripts() {
    echo ""
    echo "Testing packaging scripts..."
    
    # Check if scripts exist and are executable
    scripts=(
        "package_linux.sh"
        "package_windows.sh" 
        "package_macos.sh"
        "sign_artifacts.sh"
    )
    
    for script in "${scripts[@]}"; do
        if [ -f "$SCRIPT_DIR/$script" ]; then
            if [ -x "$SCRIPT_DIR/$script" ]; then
                print_status "OK" "$script is executable"
            else
                print_status "WARN" "$script exists but not executable"
            fi
        else
            print_status "FAIL" "$script not found"
        fi
    done
}

# Test GitHub Actions workflow
test_workflow() {
    echo ""
    echo "Testing CI/CD workflow..."
    
    if [ -f "$PROJECT_ROOT/.github/workflows/release-packages.yml" ]; then
        print_status "OK" "release-packages.yml found"
        
        # Basic YAML validation
        if command -v yq >/dev/null 2>&1; then
            if yq eval '.' "$PROJECT_ROOT/.github/workflows/release-packages.yml" >/dev/null 2>&1; then
                print_status "OK" "Workflow YAML is valid"
            else
                print_status "FAIL" "Workflow YAML is invalid"
            fi
        else
            print_status "SKIP" "yq not available for YAML validation"
        fi
    else
        print_status "FAIL" "release-packages.yml not found"
    fi
}

# Run all tests
main() {
    check_binaries
    test_linux_packages
    test_windows_packages
    test_macos_packages
    test_docker
    test_scripts
    test_workflow
    
    echo ""
    echo "Package testing complete!"
    echo ""
    echo "To create packages, run:"
    echo "  ./scripts/package_linux.sh   # On Linux"
    echo "  ./scripts/package_windows.sh # On Windows"
    echo "  ./scripts/package_macos.sh   # On macOS"
}

main