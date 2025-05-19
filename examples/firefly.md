# Running a FireFly Node

This guide explains how to run a FireFly node using dms

## Prerequisites for compute providers
- Docker installed on the compute machines.
- Load fuse kernel mod
- Clone dms to the compute provider machines
- Build the glusterfs client
- Firefly docker image
- Glusterfs client certificates and volumes

### Clone dms to the compute provider machines

```
git clone https://gitlab.com/nunet/device-management-service.git
cd device-management-service

make linux_amd64
sudo cp dms_linux_amd64 /usr/bin/dms
sudo setcap cap_net_admin,cap_sys_admin+ep /usr/bin/dms
```

### Load fuse kernel mod

Compute providers must ensure they have fuse kernel loaded

```
sudo modprobe fuse
sudo sed -i 's/#user_allow_other/user_allow_other/g' /etc/fuse.conf
```

### Build the glusterfs client

In order to mount gluster volumes we need to build the image on the compute providers machines. Inside DMS run:

```
make build-nunet-glusterfs-client
```

to verify check the output of this command:

```
docker images
```

### Firefly image

To run the firefly nodes we need the image which i pushed to our registry

```
docker pull registry.gitlab.com/nunet/device-management-service/f1r3fly-rnode
```

### Glusterfs client certificates and volumes

For now i have created a glusterfs server running in one instance. I have created also the following volumes:

```
firefly_bootstrap
firefly_validator
```

We can find the client certs attached

## Run firefily nodes

For this demo, we will run a bootstrap and validator node. We need to setup 2 compute provider machines and an orchestrator.
Install dms on those machines, setup caps and onboard them.

Note that i have already copied to configuration files for the bootstrap and validator node in the glusterfs volumes.

On the orchestrator machines we need the ensemble

We also need the client certs on the orchestrator and the correct path in the above ensemble.

We create a deployment from the orchestrator and see 2 running nodes on 2 different machines.

The ensemble can be found in `examples/firefly.yaml`