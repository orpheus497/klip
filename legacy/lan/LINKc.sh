#!/bin/bash

set -e

# Check if configuration file exists
if [ ! -f ~/.LINK/config.sh ]; then
    echo "Error: Configuration file ~/.LINK/config.sh not found." >&2
    echo "Please create the configuration file or use 'klipc' instead." >&2
    exit 1
fi

# Source the configuration file
source ~/.LINK/config.sh

# Validate required variables
if [ -z "${LAN_REMOTE_USER}" ] || [ -z "${LAN_REMOTE_HOST}" ]; then
    echo "Error: LAN_REMOTE_USER and LAN_REMOTE_HOST must be set in ~/.LINK/config.sh" >&2
    exit 1
fi

# Check for input parameter
if [ -z "$1" ]; then
    echo "Usage: LINKc <path>" >&2
    exit 1
fi

# Check if rsync is installed
if ! command -v rsync &> /dev/null; then
    echo "Error: rsync is not installed. Please install rsync." >&2
    exit 1
fi

# Check if realpath is available
if ! command -v realpath &> /dev/null; then
    echo "Error: realpath is not installed. Please install coreutils." >&2
    exit 1
fi

# Get the absolute path of the source
src_path=$(realpath "$1")

# Check if the source path exists
if [ ! -e "$src_path" ]; then
    echo "Error: Source path does not exist: $src_path" >&2
    exit 1
fi

# Determine the destination path
if [[ "$src_path" == "$HOME"* ]]; then
    # Path is inside home directory, so maintain the relative structure
    relative_path=${src_path#$HOME/}
    dest_path="~/${relative_path}"
    # Copy files to the remote machine with directory creation
    rsync -avz --rsync-path="mkdir -p ~/${relative_path%/*} && rsync" "$src_path" "${LAN_REMOTE_USER}@${LAN_REMOTE_HOST}:${dest_path}"
else
    # Path is outside home directory, so use an absolute path
    dest_path="$src_path"
    # Copy files to the remote machine
    rsync -avz "$src_path" "${LAN_REMOTE_USER}@${LAN_REMOTE_HOST}:${dest_path}"
fi
