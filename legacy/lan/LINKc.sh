#!/bin/bash

# Source the configuration file
source ~/.LINK/config.sh

# Check for input parameter
if [ -z "$1" ]; then
    echo "Usage: LINKc <path>"
    exit 1
fi

# Get the absolute path of the source
src_path=$(realpath "$1")

# Check if the source path exists
if [ ! -e "$src_path" ]; then
    echo "Error: Source path does not exist." >&2
    exit 1
fi

# Determine the destination path
if [[ "$src_path" == "$HOME"* ]]; then
    # Path is inside home directory, so maintain the relative structure
    relative_path=${src_path#$HOME/}
    dest_path="~/${relative_path}"
else
    # Path is outside home directory, so use an absolute path
    dest_path="$src_path"
fi

# Copy files to the remote machine
rsync -avz --rsync-path="mkdir -p ~/${relative_path%/*} && rsync" "$src_path" "${LAN_REMOTE_USER}@${LAN_REMOTE_HOST}:${dest_path}"