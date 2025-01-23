#!/bin/bash

# This script is an easy and interactive alternative for setting up:
#
# 1. user and node identities
# 2. anchoring user with root capabilities on node
#
# It also goes through the process of generating the passphrase and
# optionally deleting all previous data for a fresh environment

set -e

if [ "$( ls -A "$HOME/.nunet/cap" )" != "" ]; then
   echo "There are already some existing identities."
   read -rp "Do you want to erase all previous data? (y/N): " ERASE_DATA
   if [[ $ERASE_DATA =~ ^[Yy]$ ]]; then
       read -rp "Are you sure? This will delete all existing NuNet data (y/N): " CONFIRM_ERASE
       if [[ $CONFIRM_ERASE =~ ^[Yy]$ ]]; then
           echo "Removing previous data..."
           rm -rf ~/.nunet/*
       fi
   fi
fi

if [[ -z "$DMS_PASSPHRASE" ]]; then
	echo && echo "Create a passphrase. This will be used to encrypt all your identities"
	while true; do
	    read -rsp "Passphrase: " DMS_PASSPHRASE
	    echo && read -rsp "Confirm passphrase: " DMS_PASSPHRASE_CONFIRM && echo
	    if [ "$DMS_PASSPHRASE" = "$DMS_PASSPHRASE_CONFIRM" ]; then
	        export DMS_PASSPHRASE
	        break
	    else
	        echo "Passphrases do not match. Please try again."
	    fi
	done
fi

read -rp "User name (default: user): " USER_NAME
USER_NAME=${USER_NAME:-user}

read -rp "Node name (default: dms): " DMS_NAME
DMS_NAME=${DMS_NAME:-dms}

nunet cap new "$USER_NAME"
USER_DID=$(nunet key did "$USER_NAME")
nunet cap new "$DMS_NAME"

echo "Anchoring DMS context..."
nunet cap anchor --context "$DMS_NAME" --root "$USER_DID"

echo "DMS setup completed successfully!"
