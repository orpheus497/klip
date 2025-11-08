#!/bin/bash

# Create the installation directory
mkdir -p ~/.LINK

# Copy the script directories
cp -r lan ~/.LINK/
cp -r tailscale ~/.LINK/

# Copy the configuration file
cp config.sh ~/.LINK/config.sh

# Make all scripts executable
chmod +x ~/.LINK/lan/*.sh
chmod +x ~/.LINK/tailscale/*.sh

# Provide instructions for adding aliases
echo "Installation complete."
echo ""
echo "Please add the following aliases to your shell configuration file (e.g., ~/.bashrc or ~/.zshrc):"
echo ""
echo "# LINK Aliases"
echo "alias LINKlan='~/.LINK/lan/LINK.sh'"
echo "alias LINKclan='~/.LINK/lan/LINKc.sh'"
echo "alias LINKrlan='~/.LINK/lan/LINKr.sh'"
echo "alias LINKts='~/.LINK/tailscale/LINK.sh'"
echo "alias LINKcts='~/.LINK/tailscale/LINKc.sh'"
echo "alias LINKrts='~/.LINK/tailscale/LINKr.sh'"
echo ""
echo "After adding the aliases, please reload your shell configuration with 'source ~/.bashrc' or 'source ~/.zshrc'."
