#!/bin/bash

# Source the configuration file
source ~/.LINK/config.sh

# Connect to the remote machine
ssh ${LAN_REMOTE_USER}@${LAN_REMOTE_HOST}
