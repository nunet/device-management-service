get_incus_host_ips() {
    incus list --format json | jq -r --arg filter $1 '
    .[] | select(.name | test($filter))
    | .state.network.enp5s0.addresses[]
    | select(.family == "inet")
    | .address
'
}

add_glusterfs_ip_dms_acc_test() {
    local cfg_file_bk="/tmp/$(basename $DMS_ACC_TEST_CONFIG_FILE).bk"
    echo backing up "$DMS_ACC_TEST_CONFIG_FILE" to "$cfg_file_bk"
    cp "$DMS_ACC_TEST_CONFIG_FILE" "$cfg_file_bk"
    yq --yaml-roundtrip \
        --arg glusterfs_vm_name "${CLUSTER_PREFIX}1" \
        --arg glusterfs_vm_ip "$(get_incus_host_ips "${CLUSTER_PREFIX}1")" \
        '. + {"glusterfs_vm_ip": $glusterfs_vm_ip, "glusterfs_vm_name": $glusterfs_vm_name}' \
        "$DMS_ACC_TEST_CONFIG_FILE" >"$DMS_ACC_TEST_CONFIG_FILE".new
    mv "$DMS_ACC_TEST_CONFIG_FILE".new "$DMS_ACC_TEST_CONFIG_FILE"
}

clear_glusterfs_vms() {
    echo "clearing vms..."
    incus ls --format json | jq -r '.[].name' | grep "$GLUSTERFS_TEST_PREFIX" | xargs -n 1 incus rm --force
}

VM_IMAGE_CLUSTER=ubuntu/24.04/cloud
VM_IMAGE_CLIENT=ubuntu/24.04/cloud
VM_TYPE=aws:t3.small
CLUSTER_SIZE=$(yq .cluster_size config.yml)
CLIENTS_SIZE=$(yq .clients_size config.yml)
GLUSTERFS_TEST_PREFIX="${GLUSTERFS_TEST_PREFIX:-glusterfs-test}"
CLUSTER_PREFIX="${GLUSTERFS_TEST_PREFIX}-node"
CLIENTS_PREFIX="${GLUSTERFS_TEST_PREFIX}-client"
