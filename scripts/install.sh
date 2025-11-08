#!/bin/bash
# klip installation script
# Copyright (c) 2025 orpheus497

set -e

echo "klip - Installation Script"
echo "=========================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed."
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# Get installation directory
PREFIX=${PREFIX:-/usr/local}
echo "Installation directory: $PREFIX/bin"

# Build project
echo "Building klip..."
cd "$(dirname "$0")/.."
make build

# Install binaries
echo "Installing binaries..."
sudo make install PREFIX="$PREFIX"

echo ""
echo "Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'klip init' to create your configuration"
echo "  2. Follow the prompts to create your first profile"
echo "  3. Use 'klip' to connect, 'klipc' to copy, 'klipr' to retrieve"
echo ""
echo "For help: klip --help"
echo "Documentation: https://github.com/orpheus497/klip"
