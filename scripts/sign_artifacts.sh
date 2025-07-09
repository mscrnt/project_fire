#!/bin/bash
# Generate checksums and signatures for release artifacts

set -e

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Parse arguments
ARTIFACTS_DIR="${1:-$PROJECT_ROOT/dist}"
GPG_KEY="${2:-}"

echo "Generating checksums and signatures for artifacts in: $ARTIFACTS_DIR"

# Function to generate checksums
generate_checksums() {
    local dir="$1"
    local checksum_file="$dir/SHA256SUMS.txt"
    
    echo "Generating SHA256 checksums..."
    
    # Remove old checksum file if exists
    rm -f "$checksum_file"
    
    # Generate checksums for all files
    cd "$dir"
    for file in *; do
        if [ -f "$file" ] && [ "$file" != "SHA256SUMS.txt" ] && [ "$file" != "SHA256SUMS.txt.asc" ]; then
            if command -v sha256sum >/dev/null 2>&1; then
                sha256sum "$file" >> "SHA256SUMS.txt"
            elif command -v shasum >/dev/null 2>&1; then
                shasum -a 256 "$file" >> "SHA256SUMS.txt"
            else
                echo "Error: No SHA256 tool found"
                exit 1
            fi
        fi
    done
    
    if [ -f "SHA256SUMS.txt" ]; then
        echo "Checksums written to: $checksum_file"
        cat "SHA256SUMS.txt"
    fi
}

# Function to sign checksums
sign_checksums() {
    local dir="$1"
    local key="$2"
    local checksum_file="$dir/SHA256SUMS.txt"
    
    if [ ! -f "$checksum_file" ]; then
        echo "No checksum file found to sign"
        return
    fi
    
    if [ -z "$key" ]; then
        echo "No GPG key specified, skipping signing"
        return
    fi
    
    echo "Signing checksums with GPG key: $key"
    
    cd "$dir"
    
    # Check if key exists
    if ! gpg --list-secret-keys "$key" >/dev/null 2>&1; then
        echo "Error: GPG key $key not found"
        return 1
    fi
    
    # Sign the checksum file
    gpg --batch --yes --detach-sign --armor --local-user "$key" "SHA256SUMS.txt"
    
    if [ -f "SHA256SUMS.txt.asc" ]; then
        echo "Signature created: SHA256SUMS.txt.asc"
        
        # Verify the signature
        if gpg --verify "SHA256SUMS.txt.asc" "SHA256SUMS.txt" 2>&1; then
            echo "Signature verification passed"
        else
            echo "Warning: Signature verification failed"
        fi
    fi
}

# Function to create a summary file
create_summary() {
    local dir="$1"
    local summary_file="$dir/ARTIFACTS.md"
    
    echo "Creating artifact summary..."
    
    cat > "$summary_file" <<EOF
# F.I.R.E. Release Artifacts

## Package Types

### Linux
- **AppImage**: Universal Linux package that runs on most distributions
- **.deb**: Debian/Ubuntu package
- **.rpm**: RedHat/Fedora/CentOS package  
- **.tar.gz**: Generic Linux archive

### Windows
- **.exe installer**: NSIS installer with Start Menu integration
- **.zip**: Portable Windows archive

### macOS
- **.dmg**: macOS disk image with drag-to-Applications installer
- **.pkg**: macOS installer package

### Docker
- **ghcr.io/mscrnt/project_fire/fire**: Container image

## Verification

All artifacts come with SHA256 checksums in \`SHA256SUMS.txt\`.

To verify your download:
\`\`\`bash
# Linux/macOS
sha256sum -c SHA256SUMS.txt

# Windows (PowerShell)
Get-FileHash <filename> -Algorithm SHA256
\`\`\`

If GPG signatures are provided (\`SHA256SUMS.txt.asc\`), verify with:
\`\`\`bash
gpg --verify SHA256SUMS.txt.asc SHA256SUMS.txt
\`\`\`

## File Listing

EOF
    
    # Add file listing
    cd "$dir"
    ls -lah | grep -v "^total" >> "ARTIFACTS.md"
}

# Process each subdirectory in artifacts
if [ -d "$ARTIFACTS_DIR" ]; then
    # Process platform-specific directories
    for platform_dir in "$ARTIFACTS_DIR"/*; do
        if [ -d "$platform_dir" ]; then
            echo ""
            echo "Processing $(basename "$platform_dir")..."
            generate_checksums "$platform_dir"
            sign_checksums "$platform_dir" "$GPG_KEY"
        fi
    done
    
    # Process root artifacts directory
    echo ""
    echo "Processing root artifacts directory..."
    generate_checksums "$ARTIFACTS_DIR"
    sign_checksums "$ARTIFACTS_DIR" "$GPG_KEY"
    create_summary "$ARTIFACTS_DIR"
else
    echo "Error: Artifacts directory not found: $ARTIFACTS_DIR"
    exit 1
fi

echo ""
echo "Checksum and signature generation complete!"