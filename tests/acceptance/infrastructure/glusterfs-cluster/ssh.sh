#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

source "$SCRIPT_DIR"/lib.sh

case "$1" in
server)
    incus_vm_ip=$(get_incus_host_ips ${CLUSTER_PREFIX}1)
    ;;
client)
    incus_vm_ip=$(get_incus_host_ips ${CLIENTS_PREFIX}1)
    ;;
refresh)
    if [[ -f ~/.ssh/known_hosts ]]; then
        for ip in $(get_incus_host_ips $GLUSTERFS_TEST_PREFIX); do
            ssh-keygen -R "$ip"
        done
    fi
    exit 0
    ;;
*)
    echo "$0 server|client"
    ;;
esac

ssh -o IdentitiesOnly=yes \
    -i "$SCRIPT_DIR/glusterfs-key" \
    "ubuntu@$incus_vm_ip"
