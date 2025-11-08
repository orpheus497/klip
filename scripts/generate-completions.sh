#!/bin/bash
# Shell completion generation script for klip
# Created by orpheus497

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPLETION_DIR="$SCRIPT_DIR/completion"
BUILD_DIR="$(cd "$SCRIPT_DIR/../build" && pwd)"

# Ensure completion directory exists
mkdir -p "$COMPLETION_DIR"

# Ensure binaries are built
if [ ! -f "$BUILD_DIR/klip" ]; then
    echo "Error: klip binary not found. Please run 'make build' first."
    exit 1
fi

echo "Generating shell completions..."

# Generate Bash completion
echo "  - Generating Bash completion..."
"$BUILD_DIR/klip" completion bash > "$COMPLETION_DIR/klip.bash"

# Generate Zsh completion
echo "  - Generating Zsh completion..."
"$BUILD_DIR/klip" completion zsh > "$COMPLETION_DIR/_klip"

# Generate Fish completion
echo "  - Generating Fish completion..."
"$BUILD_DIR/klip" completion fish > "$COMPLETION_DIR/klip.fish"

# Generate PowerShell completion
echo "  - Generating PowerShell completion..."
"$BUILD_DIR/klip" completion powershell > "$COMPLETION_DIR/klip.ps1"

echo "Completions generated successfully!"
echo ""
echo "To install completions:"
echo ""
echo "Bash:"
echo "  sudo cp $COMPLETION_DIR/klip.bash /etc/bash_completion.d/klip"
echo "  or source $COMPLETION_DIR/klip.bash in your ~/.bashrc"
echo ""
echo "Zsh:"
echo "  cp $COMPLETION_DIR/_klip to a directory in your \$fpath"
echo "  or add this to your ~/.zshrc:"
echo "    source $COMPLETION_DIR/_klip"
echo ""
echo "Fish:"
echo "  cp $COMPLETION_DIR/klip.fish ~/.config/fish/completions/"
echo ""
echo "PowerShell:"
echo "  Add this to your PowerShell profile:"
echo "    . $COMPLETION_DIR/klip.ps1"
echo ""
