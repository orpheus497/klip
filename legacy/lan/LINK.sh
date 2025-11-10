#!/bin/bash

set -e

# Check if configuration file exists
if [ ! -f ~/.LINK/config.sh ]; then
    echo "Error: Configuration file ~/.LINK/config.sh not found." >&2
    echo "Please create the configuration file or use 'klip' instead." >&2
    exit 1
fi

# Source the configuration file
source ~/.LINK/config.sh

# Validate required variables
if [ -z "${LAN_REMOTE_USER}" ] || [ -z "${LAN_REMOTE_HOST}" ]; then
    echo "Error: LAN_REMOTE_USER and LAN_REMOTE_HOST must be set in ~/.LINK/config.sh" >&2
    exit 1
fi

# Connect to the remote machine
ssh "${LAN_REMOTE_USER}@${LAN_REMOTE_HOST}"
