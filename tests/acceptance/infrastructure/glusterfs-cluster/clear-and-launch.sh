#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "$SCRIPT_DIR"

if ! [[ -f $SCRIPT_DIR/config.yml ]]; then
    echo generating config file...
    incus_ipv4_addr=$(incus network list --format json | jq -r '.[] | select(.name == "incusbr0") | .config["ipv4.address"]')
    dms_passphrase=$(openssl rand -base64 20)
    yq --yaml-output \
        --arg incus_ipv4_addr "$incus_ipv4_addr" \
        --arg dms_passphrase "$dms_passphrase" \
        '. + {
            "extra_allowed_ips": (.extra_allowed_ips + [$incus_ipv4_addr] | unique),
            "dms_passphrase": $dms_passphrase
        }' \
        config.dist.yml >config.yml
fi

if [[ -n "${ACC_TEST_DMS_DEB_FILE:-}" ]]; then
    yq --yaml-output \
        --arg local_dms_deb $ACC_TEST_DMS_DEB_FILE \
        '. + {"local_dms_deb": $local_dms_deb}' \
        config.yml >tmpcfg
    mv tmpcfg config.yml
fi

# must be sourced after building config.yml
source "$SCRIPT_DIR"/lib.sh

populate_entity_props() {
    local entity_type=$1
    case "$entity_type" in
    "cluster")
        prefix="${CLUSTER_PREFIX}"
        num_entities=$CLUSTER_SIZE
        vm_image=$VM_IMAGE_CLUSTER
        ;;
    "clients")
        prefix="${CLIENTS_PREFIX}"
        num_entities=$CLIENTS_SIZE
        vm_image=$VM_IMAGE_CLIENT
        ;;
    *)
        echo "Error: Unknown entity type '$entity_type'" >&2
        exit 1
        ;;
    esac
}

launch_vms() {
    local entity_type=$1
    populate_entity_props "$entity_type" # Populate global variables (prefix, num_entities, vm_image)

    echo "launching ${entity_type} vms using image ${vm_image}..."
    for i in $(seq "$num_entities"); do
        incus launch --vm --type "$VM_TYPE" \
            --config=cloud-init.user-data="$(cat cloud-init.yaml)" \
            images:"$vm_image" \
            "${prefix}$i"
    done
}

wait_for_vm_ready() {
    local entity_type=$1
    populate_entity_props "$entity_type"
    for i in $(seq 1 "$num_entities"); do
        vm_name="$prefix$i"
        while ! incus exec "$vm_name" -- echo ok >/dev/null 2>&1; do
            sleep 5
        done
        while ! get_incus_host_ips "$vm_name"; do
            sleep 5
        done
    done
}

wait_cloud_init() {
    local entity_type=$1
    populate_entity_props "$entity_type"
    for i in $(seq $num_entities); do
        incus exec "${prefix}$i" -- cloud-init status --wait || true
    done
}

generate_ansible_hosts_for_entities() {
    local entity_type=$1
    populate_entity_props "$entity_type"
    i=1
    for host_address in $(get_incus_host_ips "${prefix}[\\d]+"); do
        echo "${prefix}$i ansible_host=$host_address $ansible_host_cfg" >>hosts_glusterfs
        ((i++))
    done
}

generate_ansible_hosts() {
    ansible_host_cfg="ansible_connection=ssh ansible_user=ubuntu ansible_python_interpreter=auto_silent ansible_become=true ansible_become_method=sudo ansible_ssh_private_key_file=$SCRIPT_DIR/glusterfs-key"

    echo '[glusterfs_servers]' >hosts_glusterfs
    generate_ansible_hosts_for_entities "cluster"

    echo '[glusterfs_clients]' >>hosts_glusterfs
    generate_ansible_hosts_for_entities "clients"
}

if [[ $(incus ls --format json | jq --arg glusterfs_prefix $GLUSTERFS_TEST_PREFIX '[.[] | select(.name | test($glusterfs_prefix)) | .name] | length > 0') == "true" ]]; then
    clear_glusterfs_vms
fi

echo 'generating glusterfs ssh key...'
if ! [[ -f glusterfs-key.pub ]]; then
    ssh-keygen -q -t ed25519 -f glusterfs-key -C glusterfs-key -N ""
else
    echo 'Glusterfs ssh key already exists. Will not regenerate.'
fi

sed "s~##SSH_KEY~$(cat glusterfs-key.pub)~g" cloud-init.tpl.yaml >cloud-init.yaml

echo 'deploying glusterfs on incus vms...'

launch_vms "cluster"
launch_vms "clients"

wait_for_vm_ready "cluster"
wait_for_vm_ready "clients"

echo 'waiting for cloudinit script to finish...'
wait_cloud_init "cluster"
wait_cloud_init "clients"

echo 'generating ansible hosts file...'
generate_ansible_hosts

echo refresh server SSH ids...
"$SCRIPT_DIR"/ssh.sh refresh

if [[ -n "${DMS_ACC_TEST_CONFIG_FILE:-}" ]]; then
    add_glusterfs_ip_dms_acc_test
fi

echo 'Done.'
