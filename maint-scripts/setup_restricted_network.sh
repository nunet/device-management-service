#!/bin/bash

# This script allows the user to either create or join a restricted network interactively.
#
# It assumes both user and node local identities are created beforehand, as well as the node
# having your user set as anchor root. Please refer to quickstart.sh
#
# Create: it will perform all operations locally and you should run this script in the machine
# in which it will reside your organization. It also grants the local user/node on that same machine, if any.

set -e

setup() {
	nunet cap anchor --context "$USER_NAME" --provide "$GRANT_FROM_ORG"

	echo "Creating public behaviors grant..."
	GRANT_FROM_USER=$(nunet cap grant --context "$USER_NAME" --cap /dms/deployment --cap /public --cap /broadcast --topic /nunet --expiry "$EXPIRY_DATE" "$ORG_DID")

	echo "Adding require anchor to DMS..."
	nunet cap anchor --context "$DMS_NAME" --require "$GRANT_FROM_USER"

	echo "Creating delegation to DMS..."
	DELEGATE_TOKEN=$(nunet cap delegate --context "$USER_NAME" --cap /dms/deployment --cap /public --cap /broadcast --topic /nunet --expiry "$EXPIRY_DATE" "$DMS_DID")

	echo "Adding provide anchor to DMS..."
	nunet cap anchor --context "$DMS_NAME" --provide "$DELEGATE_TOKEN"
	echo
}

if [[ -z "$DMS_PASSPHRASE" ]]; then
	echo "Your passphrase is not set. Setting a passphrase will prevent the script from prompting it every time."
	read -srp "Passphrase: " PASSPHRASE
	export DMS_PASSPHRASE="$PASSPHRASE"
	echo
fi

PS3="Select an option: "

select opt in Create Join Grant Quit; do
	case $opt in
	Create)
		read -rp "Organization name (default: org): " ORG_NAME
		ORG_NAME=${ORG_NAME:-org}
		read -rp "User name (default: user): " USER_NAME
		USER_NAME=${USER_NAME:-user}
		read -rp "Node name (default: dms): " DMS_NAME
		DMS_NAME=${DMS_NAME:-dms}

		DEFAULT_EXPIRY=$(date -d "+30 days" +%Y-%m-%d)
		read -rp "Set an expiry date for your tokens (default: $DEFAULT_EXPIRY): " EXPIRY_DATE
		EXPIRY_DATE=${EXPIRY_DATE:-$DEFAULT_EXPIRY}

		if [ -f "$HOME/.nunet/cap/$ORG_NAME.cap" ]; then
			while true; do
				read -rp "'$ORG_NAME' already exists. Do you want to overwrite it? (y/N) " yn
				case $yn in
				[Yy]*)
					nunet cap new -f "$ORG_NAME"
					break
					;;
				[Nn]*) break ;;
				*) echo "Please answer yes or no." ;;
				esac
			done
		else
			nunet cap new "$ORG_NAME"
		fi

		ORG_DID=$(nunet key did "$ORG_NAME")
		USER_DID=$(nunet key did "$USER_NAME")
		DMS_DID=$(nunet key did "$DMS_NAME")
		GRANT_FROM_ORG=$(nunet cap grant --context "$ORG_NAME" --cap /dms/deployment --cap /public --cap /broadcast --topic /nunet --expiry "$EXPIRY_DATE" "$USER_DID")

		setup
		;;
	Join)
		read -rp "Organization DID: " ORG_DID
		while [ "$ORG_DID" = "" ]; do
			echo "Please specify a DID"
			read -rp "Organization DID: " ORG_DID
		done

		read -rp "User name (default: user): " USER_NAME
		USER_NAME=${USER_NAME:-user}
		read -rp "Node name (default: dms): " DMS_NAME
		DMS_NAME=${DMS_NAME:-dms}

		read -rp "Token from organization: " GRANT_FROM_ORG
		while [ "$GRANT_FROM_ORG" = "" ]; do
			echo "Please specify a DID"
			read -rp "Token from organization: " GRANT_FROM_ORG
		done

		DEFAULT_EXPIRY=$(date -d "+30 days" +%Y-%m-%d)
		read -rp "Set an expiry date for your tokens (default: $DEFAULT_EXPIRY): " EXPIRY_DATE
		EXPIRY_DATE=${EXPIRY_DATE:-$DEFAULT_EXPIRY}

		USER_DID=$(nunet key did "$USER_NAME")
		DMS_DID=$(nunet key did "$DMS_NAME")

		setup
		;;
	Grant)
		read -rp "Organization name (default: org): " ORG_NAME
		ORG_NAME=${ORG_NAME:-org}

		read -rp "User DID to grant: " USER_DID
		while [ "$USER_DID" = "" ]; do
			echo "Please specify a DID"
			read -rp "User DID to grant: " USER_DID
		done

		DEFAULT_EXPIRY=$(date -d "+30 days" +%Y-%m-%d)
		read -rp "Set an expiry date for your tokens (default: $DEFAULT_EXPIRY): " EXPIRY_DATE
		EXPIRY_DATE=${EXPIRY_DATE:-$DEFAULT_EXPIRY}

		nunet cap grant --context "$ORG_NAME" --cap /dms/deployment --cap /public --cap /broadcast --topic /nunet --expiry "$EXPIRY_DATE" "$USER_DID" && echo
		;;
	Quit)
		exit 1
		;;
	*)
		echo "Invalid option."
		;;
	esac
	REPLY=
done
