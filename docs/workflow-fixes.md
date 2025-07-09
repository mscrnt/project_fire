# GitHub Actions Workflow Fixes

This document summarizes the fixes applied to resolve workflow failures.

## Issues Fixed

### 1. Self-Hosted Runner References
- **Files**: `ci.yml`, `release.yml`
- **Fix**: Changed `self-hosted` to `windows-latest` in matrix configurations
- **Reason**: Self-hosted runners require manual setup and were not available

### 2. Deprecated GitHub Actions
- **File**: `release.yml`
- **Fix**: Replaced `actions/upload-release-asset@v1` with GitHub CLI (`gh release upload`)
- **Reason**: v1 action is deprecated and may have authentication issues

### 3. GUI Build References
- **File**: `ci.yml`
- **Fix**: Added directory existence check before attempting GUI build
- **Reason**: GUI directory (`cmd/fire-gui`) exists but build was failing without proper checks

### 4. Missing Dependencies
- **File**: `ci.yml`
- **Fix**: Updated Windows dependency installation, added OpenSSL check
- **Reason**: Tests were failing due to missing tools

### 5. Docker Workflow
- **File**: `docker-image.yml`
- **Fix**: Removed inline Dockerfile creation, now verifies existing Dockerfile
- **Reason**: Dockerfile already exists in repository

### 6. Integration Tests
- **File**: `ci.yml`
- **Fix**: Made certificate tests more robust with proper error handling
- **Reason**: Tests assumed commands were implemented

## Remaining Potential Issues

If workflows are still failing, check:

1. **Go Module Issues**
   - Ensure `go.mod` and `go.sum` are up to date
   - Run `go mod tidy` locally and commit changes

2. **Binary Paths**
   - Verify all referenced binaries exist after build
   - Check artifact upload/download names match

3. **Secrets Configuration**
   - Ensure `GITHUB_TOKEN` has proper permissions
   - GPG keys for signing (if used) must be configured

4. **Tool Availability**
   - Some tools like `stress-ng`, `fio` might not be available on all runners
   - Consider making these optional or using mocks for testing

## Testing Locally

To test workflows locally before pushing:

```bash
# Install act (GitHub Actions local runner)
brew install act  # or your package manager

# Test CI workflow
act -j test

# Test with specific event
act release -e release-event.json
```

## Common Workflow Commands

```bash
# Check workflow syntax
yamllint .github/workflows/*.yml

# Validate workflow files
actionlint .github/workflows/*.yml

# View workflow runs
gh run list

# View specific run details
gh run view <run-id>

# Download workflow artifacts
gh run download <run-id>
```