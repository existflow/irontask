#!/bin/bash
set -e

# Configuration
REPO="existflow/irontask"
BINARY_NAME="irontask"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.irontask"

# Default: include pre-releases
USE_PRERELEASE=true

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
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  --install    Install the latest release of $BINARY_NAME"
    echo "  --update     Update $BINARY_NAME to the latest release"
    echo "  --remove     Remove $BINARY_NAME and its configuration"
    echo "  --help       Show this help message"
    echo ""
    echo "Options:"
    echo "  --stable     Install only stable releases (no pre-releases)"
    echo "  --pre        Install pre-releases (default)"
}

# Install or Update logic
install_cli() {
    echo "Checking for releases on GitHub..."
    
    if [ "$USE_PRERELEASE" = true ]; then
        echo "   (Including pre-releases)"
        # Get the most recent release (including pre-releases)
        LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases" | grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
    else
        echo "   (Stable releases only)"
        # Get the latest stable release only
        LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    fi
    
    if [ -z "$LATEST_RELEASE" ]; then
        echo "Error: Could not find any releases. Please check: $GITHUB_URL/releases"
        exit 1
    fi

    ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
    DOWNLOAD_URL="${GITHUB_URL}/releases/download/${LATEST_RELEASE}/${ASSET_NAME}"
    
    echo "Downloading ${BINARY_NAME} ${LATEST_RELEASE} for ${OS}/${ARCH}..."
    
    tmp_dir=$(mktemp -d)
    if ! curl -sSL -f -o "${tmp_dir}/${BINARY_NAME}" "${DOWNLOAD_URL}"; then
        echo "Error: Failed to download. Asset may not exist for your platform."
        rm -rf "$tmp_dir"
        exit 1
    fi
    
    chmod +x "${tmp_dir}/${BINARY_NAME}"
    
    echo "Installing to ${INSTALL_DIR}/${BINARY_NAME} (requires sudo)..."
    sudo mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    rm -rf "$tmp_dir"
    
    echo "${BINARY_NAME} ${LATEST_RELEASE} installed successfully!"
    echo "Try running: ${BINARY_NAME} --help"
}

# Removal logic
remove_cli() {
    echo "Removing ${BINARY_NAME} binary from ${INSTALL_DIR}..."
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        sudo rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        echo "Binary removed."
    else
        echo "Binary not found in ${INSTALL_DIR}."
    fi

    echo "Removing configuration directory ${CONFIG_DIR}..."
    if [ -d "${CONFIG_DIR}" ]; then
        rm -rf "${CONFIG_DIR}"
        echo "Config directory removed."
    else
        echo "Config directory not found."
    fi
    
    echo "IronTask has been completely removed."
}

# Parse arguments
COMMAND=""
for arg in "$@"; do
    case "$arg" in
        --install|--update)
            COMMAND="install"
            ;;
        --remove)
            COMMAND="remove"
            ;;
        --help|-h)
            COMMAND="help"
            ;;
        --stable)
            USE_PRERELEASE=false
            ;;
        --pre)
            USE_PRERELEASE=true
            ;;
        *)
            echo "Error: Unknown option '$arg'"
            usage
            exit 1
            ;;
    esac
done

# Execute command
case "$COMMAND" in
    install)
        install_cli
        ;;
    remove)
        remove_cli
        ;;
    help)
        usage
        ;;
    "")
        usage
        exit 0
        ;;
esac
