#!/bin/bash

# Define the Netplan configuration file
NETPLAN_FILE="/etc/netplan/50-cloud-init.yaml"

# Backup the original file before making changes
BACKUP_FILE="${NETPLAN_FILE}.bak.$(date +%F_%T)"
cp "$NETPLAN_FILE" "$BACKUP_FILE"
echo "Backup created: $BACKUP_FILE"

# Ensure correct file permissions before editing
chmod 600 "$NETPLAN_FILE"

# Process the file
awk '
          BEGIN { found_default=0; found_extra0=0 }
          /default:/ { found_default=1 }
          /extra0:/ { found_extra0=1 }
          /dhcp4: true/ {
              print
              if (found_default) {
                  print "      dhcp4-overrides:"
                  print "        route-metric: 200"
                  found_default=0
              }
              next
          }
          /route-metric: 200/ {
              if (found_extra0) {
                  print "        route-metric: 100"
                  found_extra0=0
                  next
              }
          }
          { print }
          ' "$NETPLAN_FILE" >"${NETPLAN_FILE}.tmp"

# Replace the original file with the modified version
mv "${NETPLAN_FILE}.tmp" "$NETPLAN_FILE"
chmod 600 "$NETPLAN_FILE"
echo "Applying netplan changes..."
netplan apply
echo "Netplan configuration updated successfully."
