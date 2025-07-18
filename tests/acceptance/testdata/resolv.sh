#!/bin/bash

systemctl stop systemd-resolved || true
systemctl disable systemd-resolved || true
apt remove -y systemd-resolved

# Define the resolv.conf file
RESOLV_CONF_FILE="/etc/resolv.conf"

# Backup the original file before making changes
BACKUP_FILE="${RESOLV_CONF_FILE}.bak.$(date +%F_%T)"
cp "$RESOLV_CONF_FILE" "$BACKUP_FILE"
echo "Backup created: $BACKUP_FILE"

rm -f "$RESOLV_CONF_FILE"
echo "nameserver 8.8.8.8" > "$RESOLV_CONF_FILE"
echo "nameserver 1.1.1.1" >> "$RESOLV_CONF_FILE"
chmod 0644 "$RESOLV_CONF_FILE"
cat "$RESOLV_CONF_FILE"
sleep 5
echo "resolv.conf updated successfully."
