#!/bin/bash
# klip uninstallation script
# Copyright (c) 2025 orpheus497

set -e

echo "klip - Uninstallation Script"
echo "============================"
echo ""

# Get installation directory
PREFIX=${PREFIX:-/usr/local}
BINDIR="$PREFIX/bin"

echo "This will remove klip binaries from: $BINDIR"
echo "Your configuration at ~/.config/klip will NOT be removed."
echo ""

read -p "Continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Uninstallation cancelled."
    exit 0
fi

# Remove binaries
echo "Removing binaries..."
sudo rm -f "$BINDIR/klip" "$BINDIR/klipc" "$BINDIR/klipr"

echo ""
echo "Uninstallation complete!"
echo ""
echo "To remove configuration:"
echo "  rm -rf ~/.config/klip"
echo ""
echo "To remove legacy LINK configuration:"
echo "  rm -rf ~/.LINK"
