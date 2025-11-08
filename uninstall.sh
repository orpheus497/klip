#!/bin/bash

# Remove the installation directory
rm -rf ~/.LINK

# Provide instructions for removing aliases
echo "Uninstallation complete."
echo ""
echo "Please remove the following aliases from your shell configuration file (e.g., ~/.bashrc or ~/.zshrc):"
echo ""
echo "# LINK Aliases"
echo "alias LINKlan='~/.LINK/lan/LINK.sh'"
echo "alias LINKclan='~/.LINK/lan/LINKc.sh'"
echo "alias LINKrlan='~/.LINK/lan/LINKr.sh'"
echo "alias LINKts='~/.LINK/tailscale/LINK.sh'"
echo "alias LINKcts='~/.LINK/tailscale/LINKc.sh'"
echo "alias LINKrts='~/.LINK/tailscale/LINKr.sh'"
echo ""
echo "After removing the aliases, please reload your shell configuration with 'source ~/.bashrc' or 'source ~/.zshrc'."
