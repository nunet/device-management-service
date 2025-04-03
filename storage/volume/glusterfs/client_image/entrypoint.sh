#!/bin/bash
set -e

if [[ -z "$GLUSTER_VOLUME" || -z "$GLUSTER_HOST" || -z "$MOUNT_PATH" ]]; then
    echo "Error: Missing environment variables."
    echo "Please provide GLUSTER_VOLUME, GLUSTER_HOST, and MOUNT_PATH."
    exit 1
fi

if [[ -n "$GLUSTERFS_PEM" && -n "$GLUSTERFS_KEY" && -n "$GLUSTERFS_CA" ]]; then
    mkdir -p /var/lib/glusterd
    echo "option transport.socket.ssl-cert-depth 3" > /var/lib/glusterd/secure-access
    echo "Writing SSL certificates to /usr/lib/ssl..."
    echo "$GLUSTERFS_PEM" > /usr/lib/ssl/glusterfs.pem
    echo "$GLUSTERFS_KEY" > /usr/lib/ssl/glusterfs.key
    echo "$GLUSTERFS_CA" > /usr/lib/ssl/glusterfs.ca

    echo "$GLUSTERFS_PEM" > /etc/ssl/glusterfs.pem
    echo "$GLUSTERFS_KEY" > /etc/ssl/glusterfs.key
    echo "$GLUSTERFS_CA" > /etc/ssl/glusterfs.ca
else
    echo "Warning: SSL certificates not found in environment variables. Skipping..."
fi

mkdir -p "$MOUNT_PATH"

echo "Mounting GlusterFS volume $GLUSTER_VOLUME from $GLUSTER_HOST to $MOUNT_PATH..."
mount -t glusterfs "$GLUSTER_HOST:/$GLUSTER_VOLUME" "$MOUNT_PATH"

# check if the volume was mounted
if mountpoint -q $MOUNT_PATH; then
    echo "GlusterFS volume $GLUSTER_VOLUME mounted successfully at $MOUNT_PATH."
else
    echo "failed mounting glusterfs volume"
    exit 1
fi

# keep the container alive
tail -f /dev/null