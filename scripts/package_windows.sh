#!/bin/bash
# Package F.I.R.E. for Windows

set -e

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Parse version
VERSION="${1:-1.0.0}"
ARCH="${2:-amd64}"

echo "Packaging F.I.R.E. v${VERSION} for Windows ${ARCH}..."

# Create dist directory
DIST_DIR="${PROJECT_ROOT}/dist/windows-${ARCH}"
mkdir -p "${DIST_DIR}"

# Build binaries if not already built
if [ ! -f "${PROJECT_ROOT}/bench.exe" ]; then
    echo "Building bench.exe..."
    cd "${PROJECT_ROOT}"
    CGO_ENABLED=0 GOOS=windows GOARCH=${ARCH} go build -ldflags "-s -w -H=windowsgui -X github.com/mscrnt/project_fire/internal/version.Version=${VERSION}" -o bench.exe ./cmd/fire
fi

if [ ! -f "${PROJECT_ROOT}/fire-gui.exe" ]; then
    echo "Building fire-gui.exe..."
    cd "${PROJECT_ROOT}"
    CGO_ENABLED=0 GOOS=windows GOARCH=${ARCH} go build -ldflags "-s -w -H=windowsgui -X github.com/mscrnt/project_fire/internal/version.Version=${VERSION}" -tags=no_glfw -o fire-gui.exe ./cmd/fire-gui
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. Create portable ZIP
create_portable_zip() {
    echo "Creating portable ZIP..."
    
    ZIP_DIR="${DIST_DIR}/fire-${VERSION}-windows-${ARCH}"
    rm -rf "${ZIP_DIR}"
    mkdir -p "${ZIP_DIR}"
    
    # Copy binaries
    cp "${PROJECT_ROOT}/bench.exe" "${ZIP_DIR}/"
    cp "${PROJECT_ROOT}/fire-gui.exe" "${ZIP_DIR}/"
    
    # Copy docs
    cp "${PROJECT_ROOT}/README.md" "${ZIP_DIR}/" || true
    cp "${PROJECT_ROOT}/LICENSE" "${ZIP_DIR}/" || true
    
    # Create batch files for easy launching
    cat > "${ZIP_DIR}/fire-gui.bat" <<'EOF'
@echo off
start "" "%~dp0fire-gui.exe" %*
EOF
    
    cat > "${ZIP_DIR}/fire-cli.bat" <<'EOF'
@echo off
"%~dp0bench.exe" %*
EOF
    
    # Create ZIP
    cd "${DIST_DIR}"
    if command_exists zip; then
        zip -r "fire-${VERSION}-windows-${ARCH}.zip" "fire-${VERSION}-windows-${ARCH}"
    else
        # Try Windows PowerShell compression
        powershell -Command "Compress-Archive -Path 'fire-${VERSION}-windows-${ARCH}' -DestinationPath 'fire-${VERSION}-windows-${ARCH}.zip'" 2>/dev/null || echo "ZIP creation failed"
    fi
}

# 2. Create NSIS installer
create_nsis_installer() {
    echo "Creating NSIS installer..."
    
    # Update version in NSI script
    sed -i "s/!define VERSION \".*\"/!define VERSION \"${VERSION}\"/" "${SCRIPT_DIR}/packaging/windows/fire-installer.nsi"
    
    if command_exists makensis; then
        cd "${PROJECT_ROOT}"
        makensis "${SCRIPT_DIR}/packaging/windows/fire-installer.nsi"
    else
        echo "makensis not found, trying Windows native NSIS..."
        # Try to find NSIS on Windows
        if [ -f "/c/Program Files (x86)/NSIS/makensis.exe" ]; then
            cd "${PROJECT_ROOT}"
            "/c/Program Files (x86)/NSIS/makensis.exe" "${SCRIPT_DIR}/packaging/windows/fire-installer.nsi"
        elif [ -f "/c/Program Files/NSIS/makensis.exe" ]; then
            cd "${PROJECT_ROOT}"
            "/c/Program Files/NSIS/makensis.exe" "${SCRIPT_DIR}/packaging/windows/fire-installer.nsi"
        else
            echo "NSIS not found, skipping installer creation"
        fi
    fi
}

# 3. Create MSI installer (using WiX or msitools)
create_msi_installer() {
    echo "Creating MSI installer..."
    
    # Create WiX source file
    cat > "${SCRIPT_DIR}/packaging/windows/fire.wxs" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
    <Product Id="*" Name="F.I.R.E." Language="1033" Version="${VERSION}.0" Manufacturer="F.I.R.E. Team" UpgradeCode="12345678-1234-1234-1234-123456789012">
        <Package InstallerVersion="200" Compressed="yes" InstallScope="perMachine" />
        
        <MajorUpgrade DowngradeErrorMessage="A newer version of [ProductName] is already installed." />
        <MediaTemplate EmbedCab="yes" />
        
        <Feature Id="ProductFeature" Title="F.I.R.E." Level="1">
            <ComponentGroupRef Id="ProductComponents" />
            <ComponentGroupRef Id="Shortcuts" />
        </Feature>
    </Product>
    
    <Fragment>
        <Directory Id="TARGETDIR" Name="SourceDir">
            <Directory Id="ProgramFiles64Folder">
                <Directory Id="INSTALLFOLDER" Name="FIRE" />
            </Directory>
            <Directory Id="ProgramMenuFolder">
                <Directory Id="ApplicationProgramsFolder" Name="F.I.R.E." />
            </Directory>
        </Directory>
    </Fragment>
    
    <Fragment>
        <ComponentGroup Id="ProductComponents" Directory="INSTALLFOLDER">
            <Component Id="BenchExe" Guid="12345678-1234-1234-1234-123456789013">
                <File Id="bench.exe" Source="bench.exe" KeyPath="yes" />
            </Component>
            <Component Id="FireGuiExe" Guid="12345678-1234-1234-1234-123456789014">
                <File Id="fire_gui.exe" Source="fire-gui.exe" KeyPath="yes" />
            </Component>
        </ComponentGroup>
        
        <ComponentGroup Id="Shortcuts" Directory="ApplicationProgramsFolder">
            <Component Id="ApplicationShortcut" Guid="12345678-1234-1234-1234-123456789015">
                <Shortcut Id="ApplicationStartMenuShortcut" Name="F.I.R.E. GUI" Target="[INSTALLFOLDER]fire-gui.exe" WorkingDirectory="INSTALLFOLDER" />
                <RemoveFolder Id="ApplicationProgramsFolder" On="uninstall" />
                <RegistryValue Root="HKCU" Key="Software\FIRE" Name="installed" Type="integer" Value="1" KeyPath="yes" />
            </Component>
        </ComponentGroup>
    </Fragment>
</Wix>
EOF
    
    if command_exists wixl; then
        cd "${PROJECT_ROOT}"
        wixl -o "${DIST_DIR}/fire-${VERSION}.msi" "${SCRIPT_DIR}/packaging/windows/fire.wxs"
    else
        echo "WiX tools not found, skipping MSI creation"
    fi
}

# 4. Sign executables (if certificate available)
sign_executables() {
    echo "Checking for code signing..."
    
    if [ -n "${WINDOWS_CERT_FILE}" ] && [ -n "${WINDOWS_CERT_PASSWORD}" ]; then
        echo "Signing executables..."
        
        if command_exists signtool; then
            signtool sign /f "${WINDOWS_CERT_FILE}" /p "${WINDOWS_CERT_PASSWORD}" /t http://timestamp.digicert.com "${PROJECT_ROOT}/bench.exe"
            signtool sign /f "${WINDOWS_CERT_FILE}" /p "${WINDOWS_CERT_PASSWORD}" /t http://timestamp.digicert.com "${PROJECT_ROOT}/fire-gui.exe"
            
            if [ -f "${DIST_DIR}/fire-installer-${VERSION}.exe" ]; then
                signtool sign /f "${WINDOWS_CERT_FILE}" /p "${WINDOWS_CERT_PASSWORD}" /t http://timestamp.digicert.com "${DIST_DIR}/fire-installer-${VERSION}.exe"
            fi
        else
            echo "signtool not found, skipping code signing"
        fi
    else
        echo "No signing certificate configured"
    fi
}

# Run all packaging steps
create_portable_zip
create_nsis_installer
create_msi_installer
sign_executables

echo "Windows packaging complete!"
echo "Artifacts created in: ${DIST_DIR}"
ls -la "${DIST_DIR}/"