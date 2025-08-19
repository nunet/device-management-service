#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "$SCRIPT_DIR"

if [[ "${1:-deploy}" == "test" ]]; then
    PLAYBOOK="$SCRIPT_DIR/glusterfs-ubuntu/tests/${2:-glusterfs}.yml"
else
    PLAYBOOK=$SCRIPT_DIR/playbook.yml
fi

ssh-add "${GLUSTERFS_SSH_KEY:-glusterfs-key}"

ansible-playbook "$PLAYBOOK" -i "${GLUSTERFS_ANSIBLE_HOSTS:-hosts_glusterfs}"
