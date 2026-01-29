#!/bin/bash

# install.sh - Installs cc (Claude Commit) CLI
# Usage: curl -fsSL https://raw.githubusercontent.com/USER/REPO/main/install.sh | bash

set -e

APP_NAME="cc"
# Detect preferred installation directory
if [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="/usr/local/bin"
fi

REPO_URL="github.com/quaywin/claude-commit" # Update this to your repo
VERSION="latest"

# Detect OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "🚀 Installing $APP_NAME..."

# Check if we are in a git repo with source code
if [ -f "main.go" ] && command -v go >/dev/null 2>&1; then
    echo "🔨 Building from source..."
    go build -o $APP_NAME main.go
else
    echo "📦 Downloading binary for $OS/$ARCH..."
    # Note: This expects binaries to be named like 'cc-darwin-arm64' in GitHub Releases
    BINARY_URL="https://github.com/quaywin/claude-commit/releases/latest/download/${APP_NAME}-${OS}-${ARCH}"

    # Fallback for local testing or if you haven't set up releases yet
    if ! curl -V >/dev/null 2>&1; then
        echo "Error: curl is required."
        exit 1
    fi

    # This is a template - in a real curl | bash, you'd download the file
    # For now, we'll try to download, but warn if it fails
    if ! curl -fsSL "$BINARY_URL" -o $APP_NAME; then
        echo "❌ Could not download binary from $BINARY_URL"
        echo "If you have the source code, run: go build -o cc main.go"
        exit 1
    fi
    chmod +x $APP_NAME
fi

echo "🚚 Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv $APP_NAME "$INSTALL_DIR/$APP_NAME"
else
    echo "Password required to install to $INSTALL_DIR"
    sudo mv $APP_NAME "$INSTALL_DIR/$APP_NAME"
fi

echo "✅ $APP_NAME installed successfully!"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "⚠️  Warning: $INSTALL_DIR is not in your PATH."
    echo "To add it, add the following line to your shell profile (~/.zshrc, ~/.bashrc, etc.):"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "Run 'cc' in any git repo to start."
