#!/bin/bash
# Package F.I.R.E. for Linux distributions

set -e

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Parse version from go.mod or command line
VERSION="${1:-1.0.0}"
ARCH="${2:-amd64}"

echo "Packaging F.I.R.E. v${VERSION} for Linux ${ARCH}..."

# Create dist directory
DIST_DIR="${PROJECT_ROOT}/dist/linux-${ARCH}"
mkdir -p "${DIST_DIR}"

# Build binaries if not already built
if [ ! -f "${PROJECT_ROOT}/bench" ]; then
    echo "Building bench CLI..."
    cd "${PROJECT_ROOT}"
    CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags "-s -w -X github.com/mscrnt/project_fire/internal/version.Version=${VERSION}" -o bench ./cmd/fire
fi

if [ ! -f "${PROJECT_ROOT}/fire-gui" ]; then
    echo "Building fire-gui..."
    cd "${PROJECT_ROOT}"
    CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags "-s -w -X github.com/mscrnt/project_fire/internal/version.Version=${VERSION}" -tags=no_glfw -o fire-gui ./cmd/fire-gui
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. Create AppImage
create_appimage() {
    echo "Creating AppImage..."
    
    APPDIR="${DIST_DIR}/FIRE.AppDir"
    rm -rf "${APPDIR}"
    mkdir -p "${APPDIR}/usr/bin"
    mkdir -p "${APPDIR}/usr/share/applications"
    mkdir -p "${APPDIR}/usr/share/icons/hicolor/256x256/apps"
    
    # Copy binaries
    cp "${PROJECT_ROOT}/bench" "${APPDIR}/usr/bin/"
    cp "${PROJECT_ROOT}/fire-gui" "${APPDIR}/usr/bin/"
    chmod +x "${APPDIR}/usr/bin/bench"
    chmod +x "${APPDIR}/usr/bin/fire-gui"
    
    # Create desktop file
    cat > "${APPDIR}/usr/share/applications/fire.desktop" <<EOF
[Desktop Entry]
Name=F.I.R.E.
Comment=Full Intensity Rigorous Evaluation
Exec=fire-gui
Icon=fire
Type=Application
Categories=Utility;System;
Terminal=false
EOF
    
    # Create AppRun script
    cat > "${APPDIR}/AppRun" <<'EOF'
#!/bin/bash
HERE="$(dirname "$(readlink -f "${0}")")"
exec "${HERE}/usr/bin/fire-gui" "$@"
EOF
    chmod +x "${APPDIR}/AppRun"
    
    # Copy icon (use placeholder for now)
    if [ -f "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" ]; then
        cp "${PROJECT_ROOT}/assets/logos/fire_logo_1.png" "${APPDIR}/usr/share/icons/hicolor/256x256/apps/fire.png"
    else
        # Create a simple placeholder icon
        convert -size 256x256 xc:orange "${APPDIR}/usr/share/icons/hicolor/256x256/apps/fire.png" 2>/dev/null || true
    fi
    
    # Download appimagetool if not available
    if ! command_exists appimagetool; then
        echo "Downloading appimagetool..."
        wget -q https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage -O /tmp/appimagetool
        chmod +x /tmp/appimagetool
        APPIMAGETOOL="/tmp/appimagetool"
    else
        APPIMAGETOOL="appimagetool"
    fi
    
    # Build AppImage
    ARCH=x86_64 "${APPIMAGETOOL}" "${APPDIR}" "${DIST_DIR}/fire-${VERSION}-x86_64.AppImage" || echo "AppImage creation failed (missing tools?)"
}

# 2. Create DEB package
create_deb() {
    echo "Creating DEB package..."
    
    DEB_DIR="${DIST_DIR}/fire_${VERSION}_${ARCH}"
    rm -rf "${DEB_DIR}"
    mkdir -p "${DEB_DIR}/DEBIAN"
    mkdir -p "${DEB_DIR}/usr/bin"
    mkdir -p "${DEB_DIR}/usr/share/applications"
    mkdir -p "${DEB_DIR}/usr/share/doc/fire"
    
    # Copy binaries
    cp "${PROJECT_ROOT}/bench" "${DEB_DIR}/usr/bin/"
    cp "${PROJECT_ROOT}/fire-gui" "${DEB_DIR}/usr/bin/"
    chmod 755 "${DEB_DIR}/usr/bin/bench"
    chmod 755 "${DEB_DIR}/usr/bin/fire-gui"
    
    # Create control file
    cat > "${DEB_DIR}/DEBIAN/control" <<EOF
Package: fire
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: F.I.R.E. Team <fire@example.com>
Description: Full Intensity Rigorous Evaluation
 F.I.R.E. is a single-binary, Go-powered PC test bench designed for
 burn-in tests, endurance stress, and benchmark analysis.
EOF
    
    # Create desktop file
    cat > "${DEB_DIR}/usr/share/applications/fire.desktop" <<EOF
[Desktop Entry]
Name=F.I.R.E.
Comment=Full Intensity Rigorous Evaluation
Exec=fire-gui
Icon=fire
Type=Application
Categories=Utility;System;
Terminal=false
EOF
    
    # Create copyright file
    cat > "${DEB_DIR}/usr/share/doc/fire/copyright" <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: fire
Source: https://github.com/mscrnt/project_fire

Files: *
Copyright: 2025 F.I.R.E. Team
License: MIT
EOF
    
    # Build DEB
    if command_exists dpkg-deb; then
        dpkg-deb --build "${DEB_DIR}" "${DIST_DIR}/fire_${VERSION}_${ARCH}.deb"
    else
        echo "dpkg-deb not found, skipping DEB creation"
    fi
}

# 3. Create RPM package (using FPM)
create_rpm() {
    echo "Creating RPM package..."
    
    if command_exists fpm; then
        cd "${PROJECT_ROOT}"
        fpm -s dir -t rpm \
            -n fire \
            -v "${VERSION}" \
            --description "Full Intensity Rigorous Evaluation" \
            --url "https://github.com/mscrnt/project_fire" \
            --license "MIT" \
            --maintainer "F.I.R.E. Team <fire@example.com>" \
            --prefix /usr \
            -p "${DIST_DIR}/fire-${VERSION}-1.${ARCH}.rpm" \
            bench=/usr/bin/bench \
            fire-gui=/usr/bin/fire-gui
    else
        echo "FPM not found, attempting alien conversion..."
        if command_exists alien && [ -f "${DIST_DIR}/fire_${VERSION}_${ARCH}.deb" ]; then
            cd "${DIST_DIR}"
            alien -r "fire_${VERSION}_${ARCH}.deb" || echo "RPM creation failed"
        else
            echo "Neither fpm nor alien found, skipping RPM creation"
        fi
    fi
}

# 4. Create tarball
create_tarball() {
    echo "Creating tarball..."
    
    TAR_DIR="${DIST_DIR}/fire-${VERSION}-linux-${ARCH}"
    rm -rf "${TAR_DIR}"
    mkdir -p "${TAR_DIR}"
    
    # Copy binaries and docs
    cp "${PROJECT_ROOT}/bench" "${TAR_DIR}/"
    cp "${PROJECT_ROOT}/fire-gui" "${TAR_DIR}/"
    cp "${PROJECT_ROOT}/README.md" "${TAR_DIR}/" || true
    cp "${PROJECT_ROOT}/LICENSE" "${TAR_DIR}/" || true
    
    # Create simple install script
    cat > "${TAR_DIR}/install.sh" <<'EOF'
#!/bin/bash
echo "Installing F.I.R.E. to /usr/local/bin..."
sudo cp bench fire-gui /usr/local/bin/
sudo chmod +x /usr/local/bin/bench /usr/local/bin/fire-gui
echo "Installation complete!"
EOF
    chmod +x "${TAR_DIR}/install.sh"
    
    # Create tarball
    cd "${DIST_DIR}"
    tar czf "fire-${VERSION}-linux-${ARCH}.tar.gz" "fire-${VERSION}-linux-${ARCH}"
}

# Run all packaging steps
create_appimage
create_deb
create_rpm
create_tarball

echo "Linux packaging complete!"
echo "Artifacts created in: ${DIST_DIR}"
ls -la "${DIST_DIR}/"