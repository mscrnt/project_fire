#!/bin/bash
# Package F.I.R.E. for macOS

set -e

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Parse version
VERSION="${1:-1.0.0}"
ARCH="${2:-amd64}"

echo "Packaging F.I.R.E. v${VERSION} for macOS ${ARCH}..."

# Create dist directory
DIST_DIR="${PROJECT_ROOT}/dist/darwin-${ARCH}"
mkdir -p "${DIST_DIR}"

# Build binaries if not already built
if [ ! -f "${PROJECT_ROOT}/bench-darwin" ]; then
    echo "Building bench for macOS..."
    cd "${PROJECT_ROOT}"
    CGO_ENABLED=0 GOOS=darwin GOARCH=${ARCH} go build -ldflags "-s -w -X github.com/mscrnt/project_fire/internal/version.Version=${VERSION}" -o bench-darwin ./cmd/fire
fi

if [ ! -f "${PROJECT_ROOT}/fire-gui-darwin" ]; then
    echo "Building fire-gui for macOS..."
    cd "${PROJECT_ROOT}"
    CGO_ENABLED=0 GOOS=darwin GOARCH=${ARCH} go build -ldflags "-s -w -X github.com/mscrnt/project_fire/internal/version.Version=${VERSION}" -tags=no_glfw -o fire-gui-darwin ./cmd/fire-gui
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. Create .app bundle
create_app_bundle() {
    echo "Creating .app bundle..."
    
    APP_NAME="FIRE.app"
    APP_DIR="${DIST_DIR}/${APP_NAME}"
    
    # Remove old app if exists
    rm -rf "${APP_DIR}"
    
    # Create app structure
    mkdir -p "${APP_DIR}/Contents/MacOS"
    mkdir -p "${APP_DIR}/Contents/Resources"
    
    # Copy GUI binary as main executable
    cp "${PROJECT_ROOT}/fire-gui-darwin" "${APP_DIR}/Contents/MacOS/FIRE"
    chmod +x "${APP_DIR}/Contents/MacOS/FIRE"
    
    # Copy CLI binary
    cp "${PROJECT_ROOT}/bench-darwin" "${APP_DIR}/Contents/MacOS/bench"
    chmod +x "${APP_DIR}/Contents/MacOS/bench"
    
    # Create Info.plist
    cat > "${APP_DIR}/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>FIRE</string>
    <key>CFBundleIdentifier</key>
    <string>com.fire.testbench</string>
    <key>CFBundleName</key>
    <string>F.I.R.E.</string>
    <key>CFBundleDisplayName</key>
    <string>F.I.R.E.</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>fire</string>
    <key>CFBundleIconFile</key>
    <string>fire</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.12</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
</dict>
</plist>
EOF
    
    # Create icon (placeholder for now)
    if [ -f "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" ]; then
        # Convert PNG to ICNS if possible
        if command_exists sips && command_exists iconutil; then
            ICONSET="${DIST_DIR}/fire.iconset"
            mkdir -p "${ICONSET}"
            
            # Create multiple sizes
            sips -z 16 16     "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_16x16.png"
            sips -z 32 32     "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_16x16@2x.png"
            sips -z 32 32     "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_32x32.png"
            sips -z 64 64     "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_32x32@2x.png"
            sips -z 128 128   "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_128x128.png"
            sips -z 256 256   "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_128x128@2x.png"
            sips -z 256 256   "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_256x256.png"
            sips -z 512 512   "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_256x256@2x.png"
            sips -z 512 512   "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_512x512.png"
            sips -z 1024 1024 "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" --out "${ICONSET}/icon_512x512@2x.png"
            
            # Convert to ICNS
            iconutil -c icns "${ICONSET}" -o "${APP_DIR}/Contents/Resources/fire.icns"
            rm -rf "${ICONSET}"
        else
            # Just copy the PNG
            cp "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" "${APP_DIR}/Contents/Resources/fire.png"
        fi
    fi
}

# 2. Create DMG
create_dmg() {
    echo "Creating DMG..."
    
    DMG_NAME="fire-${VERSION}-darwin-${ARCH}.dmg"
    DMG_PATH="${DIST_DIR}/${DMG_NAME}"
    
    # Remove old DMG if exists
    rm -f "${DMG_PATH}"
    
    if command_exists hdiutil; then
        # Create temporary directory for DMG contents
        DMG_TEMP="${DIST_DIR}/dmg-temp"
        rm -rf "${DMG_TEMP}"
        mkdir -p "${DMG_TEMP}"
        
        # Copy app
        cp -R "${DIST_DIR}/FIRE.app" "${DMG_TEMP}/"
        
        # Create symlink to Applications
        ln -s /Applications "${DMG_TEMP}/Applications"
        
        # Create README
        cat > "${DMG_TEMP}/README.txt" <<EOF
F.I.R.E. - Full Intensity Rigorous Evaluation
Version ${VERSION}

To install:
1. Drag FIRE.app to the Applications folder
2. Double-click to launch

CLI Usage:
Open Terminal and run:
/Applications/FIRE.app/Contents/MacOS/bench --help

For more information:
https://github.com/mscrnt/project_fire
EOF
        
        # Create DMG
        hdiutil create -volname "F.I.R.E. ${VERSION}" \
                      -srcfolder "${DMG_TEMP}" \
                      -ov \
                      -format UDZO \
                      "${DMG_PATH}"
        
        # Clean up
        rm -rf "${DMG_TEMP}"
    else
        echo "hdiutil not found, creating ZIP instead..."
        cd "${DIST_DIR}"
        zip -r "fire-${VERSION}-darwin-${ARCH}.zip" "FIRE.app"
    fi
}

# 3. Create PKG installer
create_pkg() {
    echo "Creating PKG installer..."
    
    if command_exists pkgbuild && command_exists productbuild; then
        PKG_ROOT="${DIST_DIR}/pkg-root"
        PKG_SCRIPTS="${DIST_DIR}/pkg-scripts"
        
        # Create package root
        rm -rf "${PKG_ROOT}"
        mkdir -p "${PKG_ROOT}/Applications"
        cp -R "${DIST_DIR}/FIRE.app" "${PKG_ROOT}/Applications/"
        
        # Create CLI tools directory
        mkdir -p "${PKG_ROOT}/usr/local/bin"
        cp "${PROJECT_ROOT}/bench-darwin" "${PKG_ROOT}/usr/local/bin/bench"
        chmod +x "${PKG_ROOT}/usr/local/bin/bench"
        
        # Create scripts
        mkdir -p "${PKG_SCRIPTS}"
        
        # Post-install script
        cat > "${PKG_SCRIPTS}/postinstall" <<'EOF'
#!/bin/bash
# Create symlink for CLI access
ln -sf /Applications/FIRE.app/Contents/MacOS/bench /usr/local/bin/fire
exit 0
EOF
        chmod +x "${PKG_SCRIPTS}/postinstall"
        
        # Build component package
        pkgbuild --root "${PKG_ROOT}" \
                --scripts "${PKG_SCRIPTS}" \
                --identifier "com.fire.testbench" \
                --version "${VERSION}" \
                --install-location "/" \
                "${DIST_DIR}/fire-component.pkg"
        
        # Create distribution XML
        cat > "${DIST_DIR}/distribution.xml" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>F.I.R.E. ${VERSION}</title>
    <organization>com.fire</organization>
    <domains enable_localSystem="true"/>
    <options customize="never" require-scripts="true" rootVolumeOnly="true" />
    <pkg-ref id="com.fire.testbench" version="${VERSION}" onConclusion="none">fire-component.pkg</pkg-ref>
    <choices-outline>
        <line choice="default">
            <line choice="com.fire.testbench"/>
        </line>
    </choices-outline>
    <choice id="default"/>
    <choice id="com.fire.testbench" visible="false">
        <pkg-ref id="com.fire.testbench"/>
    </choice>
</installer-gui-script>
EOF
        
        # Build distribution package
        productbuild --distribution "${DIST_DIR}/distribution.xml" \
                    --package-path "${DIST_DIR}" \
                    "${DIST_DIR}/fire-${VERSION}-installer.pkg"
        
        # Clean up temporary files
        rm -f "${DIST_DIR}/fire-component.pkg"
        rm -f "${DIST_DIR}/distribution.xml"
        rm -rf "${PKG_ROOT}"
        rm -rf "${PKG_SCRIPTS}"
    else
        echo "pkg tools not found, skipping PKG creation"
    fi
}

# 4. Code signing (if certificate available)
sign_app() {
    echo "Checking for code signing..."
    
    if [ -n "${MACOS_CERT_NAME}" ]; then
        echo "Signing app bundle..."
        
        if command_exists codesign; then
            # Sign binaries first
            codesign --force --sign "${MACOS_CERT_NAME}" "${DIST_DIR}/FIRE.app/Contents/MacOS/bench"
            codesign --force --sign "${MACOS_CERT_NAME}" "${DIST_DIR}/FIRE.app/Contents/MacOS/FIRE"
            
            # Sign the app bundle
            codesign --force --deep --sign "${MACOS_CERT_NAME}" "${DIST_DIR}/FIRE.app"
            
            # Verify signature
            codesign --verify --deep --strict "${DIST_DIR}/FIRE.app"
            
            # Sign DMG if exists
            if [ -f "${DIST_DIR}/fire-${VERSION}-darwin-${ARCH}.dmg" ]; then
                codesign --force --sign "${MACOS_CERT_NAME}" "${DIST_DIR}/fire-${VERSION}-darwin-${ARCH}.dmg"
            fi
            
            # Sign PKG if exists
            if [ -f "${DIST_DIR}/fire-${VERSION}-installer.pkg" ]; then
                productsign --sign "${MACOS_CERT_NAME}" "${DIST_DIR}/fire-${VERSION}-installer.pkg" "${DIST_DIR}/fire-${VERSION}-installer-signed.pkg"
                mv "${DIST_DIR}/fire-${VERSION}-installer-signed.pkg" "${DIST_DIR}/fire-${VERSION}-installer.pkg"
            fi
        else
            echo "codesign not found"
        fi
    else
        echo "No signing certificate configured"
    fi
}

# 5. Notarization (if credentials available)
notarize_app() {
    echo "Checking for notarization..."
    
    if [ -n "${APPLE_ID}" ] && [ -n "${APPLE_PASSWORD}" ] && [ -n "${APPLE_TEAM_ID}" ]; then
        echo "Notarizing app..."
        
        if command_exists xcrun; then
            # Notarize DMG
            if [ -f "${DIST_DIR}/fire-${VERSION}-darwin-${ARCH}.dmg" ]; then
                echo "Notarizing DMG..."
                xcrun altool --notarize-app \
                            --primary-bundle-id "com.fire.testbench" \
                            --username "${APPLE_ID}" \
                            --password "${APPLE_PASSWORD}" \
                            --team-id "${APPLE_TEAM_ID}" \
                            --file "${DIST_DIR}/fire-${VERSION}-darwin-${ARCH}.dmg"
            fi
        else
            echo "xcrun not found"
        fi
    else
        echo "No notarization credentials configured"
    fi
}

# Run all packaging steps
create_app_bundle
create_dmg
create_pkg
sign_app
notarize_app

echo "macOS packaging complete!"
echo "Artifacts created in: ${DIST_DIR}"
ls -la "${DIST_DIR}/"