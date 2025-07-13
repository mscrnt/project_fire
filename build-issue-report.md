# Build Issue Report - Project FIRE

## Summary
Unable to build the Linux GUI binary due to OpenGL/Fyne dependencies requiring CGO, which conflicts with the build instructions.

## What I Tried

### 1. CLI Build (SUCCESS)
```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -o bench ./cmd/fire
```
**Result**: ✅ Successfully built the CLI binary (`bench`)

### 2. GUI Build - Following CLAUDE.md Instructions (FAILED)
```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -tags=no_glfw -o fire-gui ./cmd/fire-gui
```
**Result**: ❌ Build failed with error:
```
package github.com/mscrnt/project_fire/cmd/fire-gui
	imports fyne.io/fyne/v2/app
	imports fyne.io/fyne/v2/internal/driver/glfw
	imports fyne.io/fyne/v2/internal/driver/common
	imports fyne.io/fyne/v2/internal/painter/gl
	imports github.com/go-gl/gl/v2.1/gl: build constraints exclude all Go files
```

### 3. Using Official Build Script (FAILED)
```bash
./scripts/build-gui.sh
```
**Result**: ❌ Same error - the script uses the same command as in CLAUDE.md

### 4. Attempted CGO Build (INTERRUPTED)
```bash
CGO_ENABLED=1 go build -ldflags "-s -w" -o fire-gui ./cmd/fire-gui
```
**Result**: User interrupted before completion

## Root Cause Analysis

### The Problem
1. **Fyne GUI Framework** requires OpenGL for rendering
2. **OpenGL bindings** (`github.com/go-gl/gl`) require CGO
3. The build instructions specify `CGO_ENABLED=0` and `tags=no_glfw`
4. These flags are incompatible with Fyne's OpenGL requirements

### Build Tag Investigation
- The `no_glfw` tag is supposed to disable GLFW dependency
- However, Fyne v2 still imports OpenGL packages even with this tag
- The GL packages have build constraints that exclude all files when CGO is disabled

## Environment Details
- **Platform**: Linux (WSL2 - Ubuntu)
- **Go Version**: (not checked, but likely 1.20+)
- **Working Directory**: `/mnt/d/Projects/project_fire`
- **Dependencies**: Fyne v2 GUI framework

## Possible Solutions

### Option 1: Enable CGO (Linux Desktop)
```bash
# Install required dependencies
sudo apt-get install -y libgl1-mesa-dev xorg-dev

# Build with CGO
CGO_ENABLED=1 go build -ldflags "-s -w" -o fire-gui ./cmd/fire-gui
```

### Option 2: Cross-compile for Windows from Linux
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  CC=x86_64-w64-mingw32-gcc \
  go build -ldflags "-s -w" -o fire-gui.exe ./cmd/fire-gui
```

### Option 3: Use Docker Build
The project has a Dockerfile that might handle dependencies:
```bash
docker build -t fire:latest .
```

### Option 4: Headless/Server Build
Build without GUI support for server/CI environments

## Questions for Resolution

1. **Is the `no_glfw` tag actually working?** 
   - The build still tries to import GLFW packages

2. **What's the intended Linux build process?**
   - The CLAUDE.md instructions don't work as written
   - The official script uses the same failing command

3. **Are there missing build dependencies?**
   - The Windows build instructions mention CGO_ENABLED=1
   - Linux instructions specify CGO_ENABLED=0

## Files Examined
- `/mnt/d/Projects/project_fire/CLAUDE.md` - Build instructions
- `/mnt/d/Projects/project_fire/scripts/build-gui.sh` - Official build script
- `/mnt/d/Projects/project_fire/cmd/fire-gui/main.go` - GUI entry point

## Next Steps
1. Check if there are Linux-specific build dependencies to install
2. Investigate if the Dockerfile has the correct build process
3. Determine if CGO should actually be enabled for Linux GUI builds
4. Check go.mod for Fyne version and any replace directives