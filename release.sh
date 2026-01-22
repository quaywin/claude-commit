#!/bin/bash

# release.sh - Automated build and release script for cc
# Usage: ./release.sh v1.0.1 "Optional custom notes"

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <version-tag> [notes]"
    echo "Example: $0 v1.0.1"
    exit 1
fi

VERSION="$1"
CUSTOM_NOTES="$2"
APP_NAME="cc"
PLATFORMS=("darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64")

# Check for gh CLI
if ! command -v gh &> /dev/null; then
    echo "❌ Error: GitHub CLI (gh) is not installed."
    echo "Please install it: brew install gh"
    exit 1
fi

# 1. Build Binaries
echo "📦 Building releases for $VERSION..."
mkdir -p dist

for PLATFORM in "${PLATFORMS[@]}"; do
    OS=${PLATFORM%/*}
    ARCH=${PLATFORM#*/}
    OUTPUT_NAME="dist/${APP_NAME}-${OS}-${ARCH}"

    echo "🔨 Building for $OS/$ARCH..."
    GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT_NAME" main.go
done

# 2. Generate Changelog
echo "📝 Generating changelog..."
if git describe --tags --abbrev=0 >/dev/null 2>&1; then
    LAST_TAG=$(git describe --tags --abbrev=0)
    CHANGELOG=$(git log "$LAST_TAG"..HEAD --pretty=format:"- %s")
else
    CHANGELOG=$(git log --pretty=format:"- %s")
fi

if [ -n "$CUSTOM_NOTES" ]; then
    NOTES="$CUSTOM_NOTES\n\n## Changes\n$CHANGELOG"
else
    NOTES="## Changes\n$CHANGELOG"
fi

echo "------------------------------------------------"
echo -e "$NOTES"
echo "------------------------------------------------"

# 3. Create Release
echo "🚀 Publishing release $VERSION to GitHub..."
gh release create "$VERSION" dist/* --title "$VERSION" --notes "$NOTES"

echo "✅ Release $VERSION published successfully!"
