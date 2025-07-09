#!/bin/bash
# Script to create GitHub repository using gh CLI

echo "Creating GitHub repository..."

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "GitHub CLI (gh) is not installed."
    echo "Install it from: https://cli.github.com/"
    echo ""
    echo "Or create the repository manually at: https://github.com/new"
    echo "Repository name: project_fire"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "Not authenticated with GitHub CLI."
    echo "Run: gh auth login"
    exit 1
fi

# Create the repository
echo "Creating public repository: project_fire"
gh repo create project_fire \
    --public \
    --description "F.I.R.E. - Full Intensity Rigorous Evaluation. A Go-powered PC test bench for burn-in tests, endurance stress, and benchmark analysis." \
    --source=. \
    --remote=origin \
    --push

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Repository created and code pushed successfully!"
    echo "🔗 Visit: https://github.com/mscrnt/project_fire"
    echo ""
    echo "GitHub Actions will now automatically:"
    echo "- Run CI pipeline"
    echo "- Build cross-platform binaries"
    echo "- Run tests"
else
    echo ""
    echo "❌ Failed to create repository."
    echo "Create manually at: https://github.com/new"
fi