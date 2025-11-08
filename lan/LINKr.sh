#!/bin/bash

# Source the configuration file
source ~/.LINK/config.sh

# Check for input parameter
if [ -z "$1" ]; then
    echo "Usage: LINKr <remote_path>"
    exit 1
fi

# Get the remote path
remote_path=$1

# Copy files from the remote machine
rsync -avz "${LAN_REMOTE_USER}@${LAN_REMOTE_HOST}:${remote_path}" ./