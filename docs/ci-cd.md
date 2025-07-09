# F.I.R.E. CI/CD Documentation

## Overview

The F.I.R.E. project uses GitHub Actions for continuous integration and deployment. From day one, every commit is automatically:

- Linted and formatted
- Tested across multiple platforms
- Cross-compiled for Linux, Windows, and macOS
- Packaged with single-binary distribution
- Released with automated artifact uploads

## Workflows

### 1. CI Pipeline (`ci.yml`)

**Triggers:** Push to main/develop, Pull requests to main

**Jobs:**
- **Lint**: Runs `go fmt`, `go vet`, and `golangci-lint`
- **Test**: Matrix testing on Ubuntu, Windows, and macOS
- **Build**: Cross-compilation for multiple platforms and architectures
- **Integration**: Smoke tests on built binaries

**Artifacts:**
- `bench-linux-amd64`
- `bench-linux-arm64`
- `bench-windows-amd64.exe`
- `bench-darwin-amd64`
- `bench-darwin-arm64`

### 2. Release Pipeline (`release.yml`)

**Triggers:** GitHub Release creation, Manual workflow dispatch

**Features:**
- Rebuilds all platform binaries with version metadata
- Generates SHA256 checksums
- Uploads binaries to GitHub Release
- Creates release notes with installation instructions

**Release Assets:**
- Platform-specific binaries
- SHA256 checksum files
- Installation instructions

### 3. Live USB Builder (`liveusb.yml`)

**Triggers:** Manual dispatch, Tags matching `live-*`

**Options:**
- Base distribution: Alpine (default) or Ubuntu
- GUI components inclusion

**Output:**
- `fire-live.iso` - Bootable ISO image
- `persistence.img` - Persistence storage file
- `ISO_README.md` - Usage instructions

**Features:**
- Auto-starts F.I.R.E. agent on boot
- Includes stress testing tools
- Supports persistent storage
- Network-ready configuration

### 4. Docker Image (`docker-image.yml`)

**Triggers:** Push to main, Release publication, Manual dispatch

**Features:**
- Multi-platform builds (AMD64, ARM64)
- Automated security scanning with Trivy
- Pushes to GitHub Container Registry
- Includes Docker Compose example

**Image Tags:**
- `latest` - Latest main branch
- `v1.2.3` - Semantic version tags
- `main-sha123` - Commit-specific builds

## GitHub Secrets Configuration

Configure these secrets in your repository settings:

### Required Secrets
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

### Optional Secrets
- `AI_API_KEY` - For AI-powered testing features
- `CODE_SIGNING_CERT` - For binary code signing
- `ISO_SIGN_KEY` - For Live USB image signing

## Local Development

### Running CI Locally

Use [act](https://github.com/nektos/act) to test workflows locally:

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run CI workflow
act -j lint
act -j test
act -j build

# Run with specific event
act pull_request
act release
```

### Testing Workflows

1. **Test CI Pipeline:**
   ```bash
   # Create a test branch
   git checkout -b test/ci-pipeline
   
   # Make a small change
   echo "// test" >> cmd/fire/main.go
   
   # Commit and push
   git add .
   git commit -m "test: CI pipeline"
   git push origin test/ci-pipeline
   
   # Create PR and verify workflows run
   ```

2. **Test Release Pipeline:**
   ```bash
   # Create and push a tag
   git tag -a v0.1.0 -m "Test release"
   git push origin v0.1.0
   
   # Create GitHub release from the tag
   gh release create v0.1.0 --title "Test Release" --notes "Testing release pipeline"
   ```

3. **Test Live USB Builder:**
   ```bash
   # Trigger manually via GitHub UI or CLI
   gh workflow run liveusb.yml -f base_distro=alpine -f include_gui=false
   ```

## Workflow Maintenance

### Adding New Platforms

To add a new platform to the build matrix:

1. Edit `.github/workflows/ci.yml`
2. Add new matrix entry:
   ```yaml
   - os: ubuntu-latest
     goos: freebsd
     goarch: amd64
     binary: bench
   ```

### Updating Go Version

Update the `GO_VERSION` environment variable in all workflow files:
```yaml
env:
  GO_VERSION: '1.22'  # Update this
```

### Adding New Dependencies

For build dependencies, update the appropriate install step:
```yaml
# Linux
- name: Install dependencies (Linux)
  if: runner.os == 'Linux'
  run: |
    sudo apt-get update
    sudo apt-get install -y new-package

# macOS
- name: Install dependencies (macOS)
  if: runner.os == 'macOS'
  run: |
    brew install new-package

# Windows
- name: Install dependencies (Windows)
  if: runner.os == 'Windows'
  run: |
    choco install -y new-package
```

## Troubleshooting

### Common Issues

1. **Workflow not triggering:**
   - Check workflow syntax with `yamllint`
   - Verify branch protection rules
   - Check GitHub Actions is enabled for the repository

2. **Build failures:**
   - Check Go module dependencies are committed
   - Verify platform-specific code is properly tagged
   - Review build logs for missing dependencies

3. **Release upload failures:**
   - Ensure GITHUB_TOKEN has write permissions
   - Check release asset size limits (2GB max)
   - Verify release exists before upload

### Debugging

Enable debug logging:
```yaml
env:
  ACTIONS_RUNNER_DEBUG: true
  ACTIONS_STEP_DEBUG: true
```

View workflow runs:
```bash
# List recent workflow runs
gh run list

# View specific run
gh run view <run-id>

# Download artifacts
gh run download <run-id>
```

## Best Practices

1. **Use Matrix Builds**: Parallelize testing across platforms
2. **Cache Dependencies**: Speed up builds with proper caching
3. **Fail Fast**: Set `fail-fast: false` for comprehensive testing
4. **Version Everything**: Tag releases and Docker images consistently
5. **Security First**: Scan containers and dependencies regularly
6. **Document Changes**: Update this file when modifying workflows

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [act - Local GitHub Actions](https://github.com/nektos/act)
- [GitHub CLI](https://cli.github.com/)
- [Workflow Syntax](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)