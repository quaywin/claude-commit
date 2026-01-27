#!/bin/bash

# release.sh - Automated build and release script for cc
# Usage: ./release.sh [version-tag] [notes]
# If version-tag is omitted, it automatically increments the patch version.

set -e

APP_NAME="cc"
PLATFORMS=("darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64" "windows/amd64" "windows/arm64")

# Check for gh CLI
if ! command -v gh &> /dev/null; then
    echo "❌ Error: GitHub CLI (gh) is not installed."
    echo "Please install it: brew install gh"
    exit 1
fi

# Determine version
if [ -z "$1" ]; then
    echo "🔍 No version provided. Detecting latest tag..."
    if git describe --tags --abbrev=0 >/dev/null 2>&1; then
        LAST_TAG=$(git describe --tags --abbrev=0)
        echo "Last tag found: $LAST_TAG"

        # Strip 'v' if present
        VERSION_PART=${LAST_TAG#v}

        # Split by dot
        IFS='.' read -r major minor patch <<< "$VERSION_PART"

        # Increment patch
        NEW_PATCH=$((patch + 1))
        VERSION="v$major.$minor.$NEW_PATCH"
        echo "🚀 Auto-incremented to $VERSION"
    else
        echo "❌ No tags found. Please provide a version (e.g., ./release.sh v1.0.0)"
        exit 1
    fi
else
    VERSION="$1"
fi

CUSTOM_NOTES="$2"

# 1. Update Version in main.go
echo "🏷️ Updating version in main.go to $VERSION..."
# Use sed to replace the VERSION constant in main.go
# Using a temp file for cross-platform compatibility with sed -i
sed "s/const VERSION = \".*\"/const VERSION = \"$VERSION\"/" main.go > main.go.tmp && mv main.go.tmp main.go

# Commit the version bump if there are changes
if ! git diff --quiet main.go; then
    echo "💾 Committing version bump..."
    git add main.go
    git commit -m "chore: bump version to $VERSION"
    git push origin main
fi

# 2. Build Binaries
echo "📦 Building releases for $VERSION..."
mkdir -p dist

for PLATFORM in "${PLATFORMS[@]}"; do
    OS=${PLATFORM%/*}
    ARCH=${PLATFORM#*/}
    OUTPUT_NAME="dist/${APP_NAME}-${OS}-${ARCH}"

    # Add .exe extension for Windows
    if [ "$OS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo "🔨 Building for $OS/$ARCH..."
    GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT_NAME" main.go
done

# Generate checksums
echo "🔐 Generating checksums..."
cd dist
shasum -a 256 cc-* > checksums.txt
cd ..

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
