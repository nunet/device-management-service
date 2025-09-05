#!/bin/bash
set -xeuo pipefail

IMAGE_NAME="images:ubuntu/22.04/cloud"                   # Base OS image
INSTANCE_VM_NAME="ubuntu-acc-test-vm-base"               # Temporary name
INSTANCE_CONTAINER_NAME="ubuntu-acc-test-container-base" # Temporary name
IMAGE_VM_ALIAS="ubuntu-acc-test-vm"                      # Final published alias
IMAGE_CONTAINER_ALIAS="ubuntu-acc-test-container"        # Final published alias

# ======================================================================
# Function to create, prepare, and publish an instance (VM or container)
# ======================================================================
build_instance() {
   local INSTANCE_NAME=$1
   local IMAGE_ALIAS=$2
   local IS_VM=$3   # "true" or "false"

   # Cleanup (remove instance or image if they already exist)

   if incus list --format csv -c n | grep -q "^${INSTANCE_NAME}\$"; then
      echo "Removing existing instance: ${INSTANCE_NAME}"
      incus stop -f "${INSTANCE_NAME}" || true
      incus delete -f "${INSTANCE_NAME}"
   fi

   if incus image list --format csv -c L | grep -q "^${IMAGE_ALIAS}\$"; then
      echo "Removing existing image alias: ${IMAGE_ALIAS}"
      incus image delete "${IMAGE_ALIAS}"
   fi

   # Launch instance
   if [ "${IS_VM}" = "true" ]; then
      incus launch "${IMAGE_NAME}" "${INSTANCE_NAME}" --vm
   else
      incus launch "${IMAGE_NAME}" "${INSTANCE_NAME}"
   fi

   # Wait until instance is running
   MAX_RETRIES=30
   SLEEP_TIME=5
   COUNT=0
   until incus exec "${INSTANCE_NAME}" -- true; do
      COUNT=$((COUNT+1))
      if [ "${COUNT}" -ge "${MAX_RETRIES}" ]; then
         echo "ERROR: Timeout waiting for instance ${INSTANCE_NAME} to be ready"
         exit 1
      fi
      echo "Instance not ready yet... retrying (${COUNT}/${MAX_RETRIES})"
      sleep "${SLEEP_TIME}"
   done
   
   # Install applications inside the instance
   incus exec "${INSTANCE_NAME}" -- bash -c "
       set -xeuo pipefail

       # Update system packages
       apt-get update

       # Install wget (required for downloading yq)
       apt-get install -y wget

       # Install yq from GitHub release
       wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/local/bin/yq
       chmod +x /usr/local/bin/yq

       # Install certificates and curl
       apt-get install -y ca-certificates curl

       # Add Docker GPG key
       install -m 0755 -d /etc/apt/keyrings
       curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
       chmod a+r /etc/apt/keyrings/docker.asc

       # Add Docker repository
       echo \"deb [arch=\$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \$(. /etc/os-release && echo \${UBUNTU_CODENAME:-\$VERSION_CODENAME}) stable\" | tee /etc/apt/sources.list.d/docker.list > /dev/null

       # Update apt repositories
       apt-get update

       # Install Docker and sub-packages
       apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

       # Start Docker service
       systemctl start docker

       # Reset machine-id
       truncate -s 0 /etc/machine-id
       rm /var/lib/dbus/machine-id

       # Clean DHCP leases
       rm -f /var/lib/dhcp/* /var/lib/dhcp3/*
   "

   # Finalize instance image
   incus stop "${INSTANCE_NAME}"
   incus publish "${INSTANCE_NAME}" --alias "${IMAGE_ALIAS}"
   incus delete "${INSTANCE_NAME}"
}

# ======================================================================
# Run for VM and container
# ======================================================================
build_instance "${INSTANCE_VM_NAME}" "${IMAGE_VM_ALIAS}" true
build_instance "${INSTANCE_CONTAINER_NAME}" "${IMAGE_CONTAINER_ALIAS}" false
