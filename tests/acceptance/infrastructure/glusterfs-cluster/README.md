# GlusterFS Cluster deployment scripts

<!--toc:start-->
- [GlusterFS Cluster deployment scripts](#glusterfs-cluster-deployment-scripts)
  - [Requirements](#requirements)
    - [System](#system)
    - [Dependencies](#dependencies)
  - [Usage](#usage)
    - [Local usage](#local-usage)
      - [Using docker](#using-docker)
      - [Customization](#customization)
    - [Remote usage](#remote-usage)
<!--toc:end-->

This folder hosts code for provisioning and testing a GlusterFS cluster ready
with DMS deployed to the main node.

## Requirements

### System

- 16GB ram (2GB for each of the four VMs and 8GB for the host)
- 8 VCPU

### Dependencies

- incus
- ansible
- jq
- openssh-client
- unzip

## Usage

These scripts are intended to be used locally or for provisioning a remote
cluster.

### Local usage

First, make sure incus is installed and configured with `incus admin init`.

Use `./clear-and-launch.sh` to clear previous glusterfs VMs and reprovision
them.

If `config.yml` isn't set, a new one is created from `config.dist.yml`. At the
time of creation, the attribute `extra_allowed_ips` is modified to add the ipv4
address of the `incusbr0` network interface, so that the glusterfs cluster
trusts all the IPs from the incus server, including the host IP.

If `ACC_TEST_DMS_DEB_FILE` is set, the script will write it to `config.yml`
under the `local_dms_deb` attribute.

The script will also create an ssh key in the root folder and add it to
`cloud-init.yml` that is used for launching the VMs. It'll wait for
`cloud-init` to finish and will populate a `hosts_glusterfs` file ready for
ansible.

After this initial provisioning, run the ansible scripts with `ssh-agent` to
deploy the glusterfs cluster:

```bash
ssh-agent bash run.sh
```

After the deployment is done, you can test it with:

```bash
ssh-agent bash run.sh test
```

Machines can be accessed either via `incus shell` or via `ssh.sh`:

```shell
incus shell glusterfs-test-node1
# or
./ssh.sh [client|server]
```

#### Using docker

A prepared `Dockerfile` can be found in the project root. First build the image:

```bash
docker build -t glusterfs-cluster-ansible .
```

Then launch the container:

```bash
docker run -it --rm \
  --network host \
  -v $PWD:/app \
  -v /var/lib/incus/unix.socket:/var/lib/incus/unix.socket \
  --workdir /app \
  glusterfs-cluster-ansible bash
```

Then you can proceed with the instructions in [Local usage](#local-usage).

#### Customization

You can customize the deployment of the GlusterFS cluster by copying
`config.dist.yml` to `config.yml` and overriding the variables in it.

Supported configuration:

| Variable | Description | Default value|
| ------------- | -------------- | -------------- |
| `cluster_size` | Specifies the size of the cluster when deploying locally | 1 |
| `clients_size` | Specifies the number of clients when deploying locally | 1 |
| `dms_passphrase` | The passphrase used to deploy and configure DMS in the cluster's main node and clients | `very-secure-passphrase` |
| `extra_allowed_ips` | Extra IPs to be added to the glusterfs cluster firewall. Format `IP/mask`. Ex: `170.11.22.33/32`, `10.0.0.1/16`, `192.168.0.1/24` | `[]` |
| `local_dms_deb` | If set this binary is used instead of latest available via permalink (<d.nunet.io/nunet-dms-amd64-latest.zip) | null |

Environment variables:

| Variable | Description | Default value |
| ------------- | -------------- | -------------- |
| `GLUSTERFS_ROLE_CONFIG` | Config file to be used by glusterfs-ubuntu ansible role | `config.yml` |
| `GLUSTERFS_SSH_KEY`  | SSH Key used to authenticate to the nodes | `glusterfs-key` |
| `GLUSTERFS_ANSIBLE_HOSTS` | Hosts file used by glusterfs-ubuntu ansible role | `hosts_glusterfs` |

### Remote usage

The remote cluster can be pre-provisioned using `cloud-init.tpl.yml`, you only
have to change `##SSH_KEY` with an ssh public key. The private key will be used
by ansible to authenticate with the servers and clients for provisioning.

The ssh private and public keys should be named with the prefix `glusterfs-key`
and be deployed in this project's root folder at
`test-suite/infrastrucure/glusterfs-cluster`.

To create the key and populate the `cloud-init-remote.yml` run the following
command:

```bash
ssh-keygen -q -t ed25519 -f glusterfs-key-remote -C glusterfs-key-remote -N ""
sed "s~##SSH_KEY~$(cat glusterfs-key-remote.pub)~g" cloud-init.tpl.yaml >cloud-init-remote.yaml
```

After provisioning the remote servers with `cloud-init-remote.yml`, populate
the file `hosts_glusterfs_remote` according to the following structure:

```config
[glusterfs_servers]
glusterfs-main-node ansible_host={server_ip} ansible_connection=ssh ansible_user=ubuntu ansible_python_interpreter=auto_silent ansible_become=true ansible_become_method=sudo ansible_ssh_private_key_file=/path/to/glusterfs-key-remote
# ...

[glusterfs_clients]
glusterfs-client ansible_host={client_ip} ansible_connection=ssh ansible_user=ubuntu ansible_python_interpreter=auto_silent ansible_become=true ansible_become_method=sudo ansible_ssh_private_key_file=/path/to/glusterfs-key-remote
# ...
```

You can also customize the role config specifically for production deployment.
This is useful if adding for instance extra IPs to the glusterfs cluster
firewall without affecting a local deployment. To do so, `cp config.dist.yml
config.prod.yml` and `export GLUSTERFS_ROLE_CONFIG=config.prod.yml` and change
the file accordingly.

To execute the ansible role `glusterfs-ubuntu` with the correct key and hosts file, populate `GLUSTERFS_SSH_KEY` and `GLUSTERFS_ANSIBLE_HOSTS` with the path to their respective files:

```bash
export GLUSTERFS_ROLE_CONFIG=config.prod.yml
export GLUSTERFS_SSH_KEY=$PWD/glusterfs-key-remote
export GLUSTERFS_ANSIBLE_HOSTS=$PWD/hosts_glusterfs_remote
ssh-agent bash run.sh
```

These assume you are running from `test-suite/infrastructure/glusterfs-cluster`.
