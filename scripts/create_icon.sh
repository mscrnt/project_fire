#!/bin/bash
# Script to create Windows .ico file from PNG image

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Input and output paths
INPUT_PNG="${PROJECT_ROOT}/assets/logos/fire_logo_1.png"
OUTPUT_ICO="${PROJECT_ROOT}/assets/logos/fire.ico"

echo "Creating Windows .ico file from F.I.R.E. logo..."

# Check if ImageMagick is installed
if ! command -v convert >/dev/null 2>&1; then
    echo "Error: ImageMagick is required but not installed."
    echo "Install it with:"
    echo "  Ubuntu/Debian: sudo apt-get install imagemagick"
    echo "  macOS: brew install imagemagick"
    echo "  Windows: choco install imagemagick"
    exit 1
fi

# Check if input file exists
if [ ! -f "$INPUT_PNG" ]; then
    echo "Error: Input file not found: $INPUT_PNG"
    echo "Please ensure fire_logo_1.png exists in assets/logos/"
    exit 1
fi

# Create ICO with multiple sizes for Windows
# Windows expects these specific sizes: 16x16, 32x32, 48x48, 64x64, 128x128, 256x256
echo "Generating multi-resolution .ico file..."

convert "$INPUT_PNG" \
    \( -clone 0 -resize 16x16 \) \
    \( -clone 0 -resize 32x32 \) \
    \( -clone 0 -resize 48x48 \) \
    \( -clone 0 -resize 64x64 \) \
    \( -clone 0 -resize 128x128 \) \
    \( -clone 0 -resize 256x256 \) \
    -delete 0 \
    -alpha on \
    -background transparent \
    -colors 256 \
    "$OUTPUT_ICO"

if [ -f "$OUTPUT_ICO" ]; then
    echo "Successfully created: $OUTPUT_ICO"
    echo "The .ico file contains the following resolutions:"
    echo "  16x16, 32x32, 48x48, 64x64, 128x128, 256x256"
    
    # Show file size
    ls -lh "$OUTPUT_ICO"
else
    echo "Error: Failed to create .ico file"
    exit 1
fi

# Also create other icon formats while we're at it
echo ""
echo "Creating additional icon formats..."

# Create a 512x512 PNG for macOS and Linux
convert "$INPUT_PNG" -resize 512x512 "${PROJECT_ROOT}/assets/logos/fire_512.png"
echo "Created: fire_512.png (for macOS/Linux)"

# Create a square 1024x1024 for macOS icns generation
convert "$INPUT_PNG" -resize 1024x1024 "${PROJECT_ROOT}/assets/logos/fire_1024.png"
echo "Created: fire_1024.png (for macOS .icns)"

echo ""
echo "Icon generation complete!"
echo ""
echo "To use in NSIS installer, the path is:"
echo "  !define MUI_ICON \"..\\..\\..\\assets\\logos\\fire.ico\""