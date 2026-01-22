#!/bin/bash

# install.sh - Installs claude-commit CLI

set -e

APP_NAME="claude-commit"
INSTALL_DIR="/usr/local/bin"

echo "🔨 Building $APP_NAME..."
go build -o $APP_NAME main.go

echo "🚚 Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv $APP_NAME "$INSTALL_DIR/$APP_NAME"
else
    echo "Requires sudo to install to $INSTALL_DIR"
    sudo mv $APP_NAME "$INSTALL_DIR/$APP_NAME"
fi

echo "✅ $APP_NAME installed successfully!"
echo "You can now run '$APP_NAME' from anywhere in your git repositories."
