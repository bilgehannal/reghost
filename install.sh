#!/bin/sh
set -e

# reghost installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/bilgehannal/reghost/main/install.sh | sudo sh

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    *)
        echo "${RED}Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

PLATFORM="${OS}-${ARCH}"
echo "${GREEN}Detected platform: $PLATFORM${NC}"

# Check if running as root
if [ "$(id -u)" -ne 0 ]; then
    echo "${RED}Error: This script must be run as root (use sudo)${NC}"
    exit 1
fi

# Get latest release URL
REPO="bilgehannal/reghost"
RELEASE_URL="https://github.com/${REPO}/releases/latest/download/reghost-${PLATFORM}.tar.gz"

echo "${YELLOW}Downloading reghost from ${RELEASE_URL}...${NC}"

# Download and extract
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$RELEASE_URL" -o reghost.tar.gz
elif command -v wget >/dev/null 2>&1; then
    wget -q "$RELEASE_URL" -O reghost.tar.gz
else
    echo "${RED}Error: Neither curl nor wget is available${NC}"
    exit 1
fi

echo "${YELLOW}Extracting binaries...${NC}"
tar -xzf reghost.tar.gz

# Install binaries
INSTALL_DIR="/usr/local/bin"
echo "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"

mv reghostd "$INSTALL_DIR/reghostd"
mv reghostctl "$INSTALL_DIR/reghostctl"

# Set permissions
chmod 6755 "$INSTALL_DIR/reghostd"
chmod 6755 "$INSTALL_DIR/reghostctl"

# Set group ownership
if [ "$OS" = "darwin" ]; then
    chgrp admin "$INSTALL_DIR/reghostd"
    chgrp admin "$INSTALL_DIR/reghostctl"
else
    chgrp root "$INSTALL_DIR/reghostd"
    chgrp root "$INSTALL_DIR/reghostctl"
fi

# Cleanup
cd -
rm -rf "$TEMP_DIR"

echo "${GREEN}✓ reghost installed successfully!${NC}"
echo ""
echo "Next steps:"
echo "  1. Start the daemon: ${YELLOW}reghostd${NC}"
echo "  2. View configuration: ${YELLOW}reghostctl show${NC}"
echo "  3. Add a record: ${YELLOW}reghostctl add-record default --domain example.local --ip 127.0.0.1${NC}"
echo ""
echo "For more information, visit: https://github.com/${REPO}"
