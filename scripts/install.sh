#!/bin/bash
set -e

# Configuration
REPO="existflow/irontask"
BINARY_NAME="irontask"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.irontask"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
    echo "Error: Unsupported OS '$OS'. This script supports Linux and macOS."
    exit 1
fi

# Detect Architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture '$ARCH'."
        exit 1
        ;;
esac

GITHUB_URL="https://github.com/$REPO"

# Documentation
usage() {
    echo "IronTask CLI Management Script"
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  --install    Install the latest release of $BINARY_NAME"
    echo "  --update     Update $BINARY_NAME to the latest release"
    echo "  --remove     Remove $BINARY_NAME and its configuration"
    echo "  --help       Show this help message"
}

# Install or Update logic
install_cli() {
    echo "🔍 Checking for the latest release on GitHub..."
    
    # Get the most recent release (including pre-releases) from GitHub API
    LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases" | grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$LATEST_RELEASE" ]; then
        echo "❌ Error: Could not find any releases. Please check if there are any releases at $GITHUB_URL/releases"
        exit 1
    fi

    ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
    DOWNLOAD_URL="${GITHUB_URL}/releases/download/${LATEST_RELEASE}/${ASSET_NAME}"
    
    echo "📥 Downloading ${BINARY_NAME} ${LATEST_RELEASE} for ${OS}/${ARCH}..."
    
    tmp_dir=$(mktemp -d)
    curl -sSL -o "${tmp_dir}/${BINARY_NAME}" "${DOWNLOAD_URL}"
    
    chmod +x "${tmp_dir}/${BINARY_NAME}"
    
    echo "🚀 Installing to ${INSTALL_DIR}/${BINARY_NAME} (requires sudo)..."
    sudo mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    rm -rf "$tmp_dir"
    
    echo "✅ ${BINARY_NAME} ${LATEST_RELEASE} installed successfully!"
    echo "Try running: ${BINARY_NAME} --help"
}

# Removal logic
remove_cli() {
    echo "🗑️  Removing ${BINARY_NAME} binary from ${INSTALL_DIR}..."
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        sudo rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        echo "✅ Binary removed."
    else
        echo "ℹ️  Binary not found in ${INSTALL_DIR}."
    fi

    echo "🗑️  Removing configuration directory ${CONFIG_DIR}..."
    if [ -d "${CONFIG_DIR}" ]; then
        rm -rf "${CONFIG_DIR}"
        echo "✅ Config directory removed."
    else
        echo "ℹ️  Config directory not found."
    fi
    
    echo "✨ IronTask has been completely removed."
}

# Main logic
if [ $# -eq 0 ]; then
    usage
    exit 0
fi

case "$1" in
    --install|--update)
        install_cli
        ;;
    --remove)
        remove_cli
        ;;
    --help|-h)
        usage
        ;;
    *)
        echo "❌ Error: Unknown command '$1'"
        usage
        exit 1
        ;;
esac
