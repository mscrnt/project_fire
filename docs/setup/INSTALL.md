# F.I.R.E. Installation Guide

This guide provides detailed instructions for installing F.I.R.E. on various platforms.

## Table of Contents

- [Quick Start](#quick-start)
- [Linux Installation](#linux-installation)
  - [AppImage (Recommended)](#appimage-recommended)
  - [Debian/Ubuntu (.deb)](#debianubuntu-deb)
  - [RedHat/Fedora (.rpm)](#redhatfedora-rpm)
  - [Generic Linux (.tar.gz)](#generic-linux-targz)
- [Windows Installation](#windows-installation)
  - [Installer (Recommended)](#installer-recommended)
  - [Portable ZIP](#portable-zip)
- [macOS Installation](#macos-installation)
  - [DMG (Recommended)](#dmg-recommended)
  - [PKG Installer](#pkg-installer)
- [Docker Installation](#docker-installation)
- [Building from Source](#building-from-source)
- [Verification](#verification)
- [Post-Installation](#post-installation)

## Quick Start

Download the appropriate package for your system from the [releases page](https://github.com/mscrnt/project_fire/releases/latest).

### One-line Installation

**Linux (AppImage)**:
```bash
curl -L https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-x86_64.AppImage -o fire && chmod +x fire && ./fire gui
```

**macOS**:
```bash
curl -L https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-darwin-amd64.dmg -o fire.dmg && hdiutil mount fire.dmg && cp -R /Volumes/FIRE/FIRE.app /Applications/
```

**Windows** (PowerShell as Administrator):
```powershell
Invoke-WebRequest -Uri https://github.com/mscrnt/project_fire/releases/latest/download/fire-installer-latest.exe -OutFile fire-installer.exe; .\fire-installer.exe
```

## Linux Installation

### AppImage (Recommended)

AppImage provides a universal package that works on most Linux distributions without installation.

1. **Download the AppImage**:
   ```bash
   wget https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-x86_64.AppImage
   ```

2. **Make it executable**:
   ```bash
   chmod +x fire-latest-x86_64.AppImage
   ```

3. **Run F.I.R.E.**:
   ```bash
   ./fire-latest-x86_64.AppImage gui
   ```

4. **Optional: Install system-wide**:
   ```bash
   sudo mv fire-latest-x86_64.AppImage /usr/local/bin/fire
   ```

### Debian/Ubuntu (.deb)

For Debian-based systems (Ubuntu, Mint, Pop!_OS, etc.):

1. **Download the .deb package**:
   ```bash
   wget https://github.com/mscrnt/project_fire/releases/latest/download/fire_latest_amd64.deb
   ```

2. **Install using dpkg**:
   ```bash
   sudo dpkg -i fire_latest_amd64.deb
   ```

3. **Fix any dependency issues**:
   ```bash
   sudo apt-get install -f
   ```

4. **Alternative: Use apt**:
   ```bash
   sudo apt install ./fire_latest_amd64.deb
   ```

### RedHat/Fedora (.rpm)

For RPM-based systems (Fedora, CentOS, RHEL, openSUSE):

1. **Download the .rpm package**:
   ```bash
   wget https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-1.x86_64.rpm
   ```

2. **Install using dnf (Fedora)**:
   ```bash
   sudo dnf install fire-latest-1.x86_64.rpm
   ```

3. **Or using yum (CentOS/RHEL)**:
   ```bash
   sudo yum install fire-latest-1.x86_64.rpm
   ```

4. **Or using zypper (openSUSE)**:
   ```bash
   sudo zypper install fire-latest-1.x86_64.rpm
   ```

### Generic Linux (.tar.gz)

For any Linux distribution:

1. **Download the archive**:
   ```bash
   wget https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-linux-amd64.tar.gz
   ```

2. **Extract the archive**:
   ```bash
   tar xzf fire-latest-linux-amd64.tar.gz
   ```

3. **Install to system**:
   ```bash
   cd fire-latest-linux-amd64
   sudo ./install.sh
   ```

4. **Or run directly**:
   ```bash
   ./bench gui
   ./fire-gui
   ```

## Windows Installation

### Installer (Recommended)

The NSIS installer provides Start Menu integration and adds F.I.R.E. to your PATH.

1. **Download the installer**:
   - Visit [releases page](https://github.com/mscrnt/project_fire/releases/latest)
   - Download `fire-installer-latest.exe`

2. **Run the installer**:
   - Double-click the downloaded file
   - Follow the installation wizard
   - Choose installation directory (default: `C:\Program Files\FIRE`)

3. **Launch F.I.R.E.**:
   - From Start Menu: F.I.R.E. → F.I.R.E. GUI
   - From Command Prompt: `bench --help`
   - From PowerShell: `fire-gui`

### Portable ZIP

For a portable installation without administrative privileges:

1. **Download the ZIP**:
   ```powershell
   Invoke-WebRequest -Uri https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-windows-amd64.zip -OutFile fire.zip
   ```

2. **Extract the archive**:
   ```powershell
   Expand-Archive -Path fire.zip -DestinationPath C:\Tools\FIRE
   ```

3. **Add to PATH (optional)**:
   ```powershell
   $env:Path += ";C:\Tools\FIRE"
   ```

4. **Run F.I.R.E.**:
   ```powershell
   C:\Tools\FIRE\fire-gui.exe
   C:\Tools\FIRE\bench.exe test cpu
   ```

## macOS Installation

### DMG (Recommended)

The DMG provides a standard macOS installation experience.

1. **Download the DMG**:
   ```bash
   curl -L https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-darwin-amd64.dmg -o fire.dmg
   ```

2. **Mount the DMG**:
   ```bash
   hdiutil mount fire.dmg
   ```

3. **Install the application**:
   - Drag FIRE.app to the Applications folder
   - Or from terminal:
     ```bash
     cp -R /Volumes/FIRE/FIRE.app /Applications/
     ```

4. **Unmount the DMG**:
   ```bash
   hdiutil unmount /Volumes/FIRE
   ```

5. **Launch F.I.R.E.**:
   - From Launchpad or Applications folder
   - From terminal: `/Applications/FIRE.app/Contents/MacOS/bench`

### PKG Installer

For automated deployment or system-wide installation:

1. **Download the PKG**:
   ```bash
   curl -L https://github.com/mscrnt/project_fire/releases/latest/download/fire-latest-installer.pkg -o fire.pkg
   ```

2. **Install the package**:
   ```bash
   sudo installer -pkg fire.pkg -target /
   ```

3. **Verify installation**:
   ```bash
   which bench
   fire --version
   ```

## Docker Installation

### Using Docker Hub / GitHub Container Registry

1. **Pull the image**:
   ```bash
   docker pull ghcr.io/mscrnt/project_fire/fire:latest
   ```

2. **Run a test**:
   ```bash
   docker run --rm ghcr.io/mscrnt/project_fire/fire:latest test cpu --duration 30s
   ```

3. **Run with persistent data**:
   ```bash
   docker run -v fire-data:/home/fire/data ghcr.io/mscrnt/project_fire/fire:latest test memory
   ```

### Using Docker Compose

1. **Download docker-compose.yml**:
   ```bash
   wget https://raw.githubusercontent.com/mscrnt/project_fire/main/docker-compose.yml
   ```

2. **Start services**:
   ```bash
   docker-compose up -d
   ```

3. **Run tests**:
   ```bash
   docker-compose run fire-cli test cpu --duration 1m
   ```

## Building from Source

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional)

### Build Steps

1. **Clone the repository**:
   ```bash
   git clone https://github.com/mscrnt/project_fire.git
   cd project_fire
   ```

2. **Build the binaries**:
   ```bash
   # CLI only
   go build -ldflags "-s -w" -o bench ./cmd/fire
   
   # GUI
   go build -ldflags "-s -w" -tags=no_glfw -o fire-gui ./cmd/fire-gui
   ```

3. **Install system-wide**:
   ```bash
   sudo cp bench fire-gui /usr/local/bin/
   ```

## Verification

### Checksum Verification

All releases include SHA256 checksums. To verify your download:

1. **Download the checksum file**:
   ```bash
   wget https://github.com/mscrnt/project_fire/releases/latest/download/SHA256SUMS.txt
   ```

2. **Verify the checksum**:
   
   **Linux/macOS**:
   ```bash
   sha256sum -c SHA256SUMS.txt
   ```
   
   **Windows (PowerShell)**:
   ```powershell
   $hash = Get-FileHash fire-installer.exe -Algorithm SHA256
   Select-String -Path SHA256SUMS.txt -Pattern $hash.Hash
   ```

### GPG Signature Verification

If GPG signatures are provided:

1. **Import the signing key**:
   ```bash
   gpg --keyserver keys.openpgp.org --recv-keys [KEY_ID]
   ```

2. **Download the signature**:
   ```bash
   wget https://github.com/mscrnt/project_fire/releases/latest/download/SHA256SUMS.txt.asc
   ```

3. **Verify the signature**:
   ```bash
   gpg --verify SHA256SUMS.txt.asc SHA256SUMS.txt
   ```

## Post-Installation

### First Run

1. **Initialize the configuration**:
   ```bash
   bench init
   ```

2. **Run a simple test**:
   ```bash
   bench test cpu --duration 10s
   ```

3. **Launch the GUI**:
   ```bash
   bench gui
   # or
   fire-gui
   ```

### System Requirements

- **Linux**: Any modern distribution with glibc 2.17+
- **Windows**: Windows 10/11 or Windows Server 2016+
- **macOS**: macOS 10.12 Sierra or later
- **RAM**: 512MB minimum, 2GB recommended
- **Disk**: 100MB for installation, 1GB for test data

### Troubleshooting

**Linux AppImage won't run**:
```bash
# Install FUSE if missing
sudo apt install fuse libfuse2  # Debian/Ubuntu
sudo dnf install fuse           # Fedora
```

**Windows: "Windows protected your PC" warning**:
- Click "More info" → "Run anyway"
- Or disable SmartScreen temporarily

**macOS: "Cannot be opened because the developer cannot be verified"**:
```bash
# Remove quarantine attribute
xattr -cr /Applications/FIRE.app
```

**Permission denied errors**:
```bash
# Ensure executable permissions
chmod +x bench fire-gui
```

### Uninstallation

**Linux (AppImage)**: Simply delete the file

**Linux (.deb)**:
```bash
sudo apt remove fire
```

**Linux (.rpm)**:
```bash
sudo dnf remove fire  # or yum/zypper
```

**Windows**: Use Add/Remove Programs or run the uninstaller

**macOS**: Drag FIRE.app to Trash

**Docker**:
```bash
docker rmi ghcr.io/mscrnt/project_fire/fire:latest
```

## Getting Help

- Documentation: https://github.com/mscrnt/project_fire/wiki
- Issues: https://github.com/mscrnt/project_fire/issues
- Discussions: https://github.com/mscrnt/project_fire/discussions