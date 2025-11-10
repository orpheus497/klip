#!/bin/bash

set -e

# Check if configuration file exists
if [ ! -f ~/.LINK/config.sh ]; then
    echo "Error: Configuration file ~/.LINK/config.sh not found." >&2
    echo "Please create the configuration file or use 'klipr' instead." >&2
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
    echo "Usage: LINKr <remote_path>" >&2
    exit 1
fi

# Check if rsync is installed
if ! command -v rsync &> /dev/null; then
    echo "Error: rsync is not installed. Please install rsync." >&2
    exit 1
fi

# Get the remote path
remote_path=$1

# Copy files from the remote machine
rsync -avz "${LAN_REMOTE_USER}@${LAN_REMOTE_HOST}:${remote_path}" ./
