#!/bin/bash
# Wrapper script to run the F.I.R.E. GUI with proper locale settings

# Set locale to avoid Fyne warnings
# Use C.UTF-8 as it's more universally available
if [[ "$LANG" == "C" ]] || [[ -z "$LANG" ]]; then
    export LANG="C.UTF-8"
fi

# Find the GUI binary
GUI_BIN="fire-gui"
if [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    GUI_BIN="fire-gui.exe"
fi

# Check if binary exists in current directory
if [ -f "./$GUI_BIN" ]; then
    exec "./$GUI_BIN" "$@"
# Check if binary exists in same directory as script
elif [ -f "$(dirname "$0")/../$GUI_BIN" ]; then
    exec "$(dirname "$0")/../$GUI_BIN" "$@"
# Try to find in PATH
elif command -v "$GUI_BIN" &> /dev/null; then
    exec "$GUI_BIN" "$@"
else
    echo "Error: $GUI_BIN not found"
    echo "Please build the GUI first with: go build -o $GUI_BIN ./cmd/fire-gui"
    exit 1
fi