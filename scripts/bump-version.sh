#!/bin/bash
set -e

# Script to bump version numbers
# Usage: ./scripts/bump-version.sh [major|minor|patch]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the bump type from argument
BUMP_TYPE="${1:-patch}"

# Read current version
if [ ! -f VERSION ]; then
    echo -e "${RED}ERROR: VERSION file not found${NC}"
    exit 1
fi

CURRENT_VERSION=$(cat VERSION)
echo -e "${BLUE}Current version: ${CURRENT_VERSION}${NC}"

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Bump the appropriate component
case "$BUMP_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo -e "${RED}ERROR: Invalid bump type. Use 'major', 'minor', or 'patch'${NC}"
        exit 1
        ;;
esac

# Construct new version
NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo -e "${GREEN}New version: ${NEW_VERSION}${NC}"

# Update VERSION file
echo "$NEW_VERSION" > VERSION

# Update fallback version in GUI main.go
sed -i "s/appVersion = \"v[0-9]\+\.[0-9]\+\.[0-9]\+\"/appVersion = \"v${NEW_VERSION}\"/" cmd/fire-gui/main.go

# Check if git is available and we're in a git repo
if command -v git &> /dev/null && git rev-parse --git-dir &> /dev/null; then
    # Add the changes
    git add -f VERSION cmd/fire-gui/main.go
    
    # Create commit
    git commit -m "chore: bump version to v${NEW_VERSION}

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"
    
    # Create tag
    git tag -a "v${NEW_VERSION}" -m "Release v${NEW_VERSION}"
    
    echo -e "${GREEN}✓ Version bumped to ${NEW_VERSION}${NC}"
    echo -e "${GREEN}✓ Changes committed${NC}"
    echo -e "${GREEN}✓ Tag v${NEW_VERSION} created${NC}"
    echo ""
    echo -e "${BLUE}To push the changes and tag:${NC}"
    echo "  git push origin main"
    echo "  git push origin v${NEW_VERSION}"
else
    echo -e "${GREEN}✓ Version updated to ${NEW_VERSION}${NC}"
    echo -e "${BLUE}Note: Not in a git repository, skipping commit and tag${NC}"
fi