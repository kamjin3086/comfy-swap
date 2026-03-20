#!/bin/bash
# Trigger a new release via GitHub Actions
# Usage: ./scripts/release.sh v1.0.0

set -e

VERSION=${1:-}

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.0.0"
    exit 1
fi

# Validate version format
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "Error: Version must match format v1.0.0 or v1.0.0-beta.1"
    exit 1
fi

echo "Triggering release for version: $VERSION"

# Using GitHub CLI
gh workflow run release.yml -f version="$VERSION"

echo ""
echo "Release workflow triggered!"
echo "View progress: gh run list --workflow=release.yml"
echo "Or visit: https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/actions"
