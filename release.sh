#!/bin/bash

# release.sh - Cross-compiles cc for different platforms

set -e

APP_NAME="cc"
PLATFORMS=("darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64")

echo "📦 Building releases..."

mkdir -p dist

for PLATFORM in "${PLATFORMS[@]}"; do
    OS=${PLATFORM%/*}
    ARCH=${PLATFORM#*/}
    OUTPUT_NAME="dist/${APP_NAME}-${OS}-${ARCH}"

    echo "🔨 Building for $OS/$ARCH..."
    GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT_NAME" main.go
done

echo "✅ Releases built in ./dist/"
ls -l dist/
