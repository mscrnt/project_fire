#!/bin/bash
# Build script for F.I.R.E. GUI

set -e

echo "Building F.I.R.E. GUI..."

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT"

# Build the GUI
echo "Building GUI binary..."
CGO_ENABLED=0 go build -ldflags "-s -w" -tags=no_glfw -o fire-gui ./cmd/fire-gui

# Make it executable
chmod +x fire-gui

echo "GUI build complete: fire-gui"
echo ""
echo "To run the GUI:"
echo "  ./fire-gui"
echo ""
echo "Or through the CLI:"
echo "  ./bench gui"