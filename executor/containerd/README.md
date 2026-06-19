# containerd executor

Containerd executor that runs jobs through containerd with CNI networking. Works with any containerd runtime shim (runc, kata, etc...)

## Requirements

- **Linux** : MacOS not supported right now
- **Root** : dms needs to run as root for access to containerd socket `/run/containerd/containerd.sock`, network namespaces in `/var/run/netns` through cni plugin in `/opt/cni/bin`
- **containerd** : containerd should be installed (at least v2.x) and daemon running
- **CNI plugins** : for networking, CNI plugins should be installed

DMS must be able to create network namespaces, run CNI plugins, bind-mount volumes, and configure iptables for subnet port forwarding. In practice, run DMS as root on compute providers that use this executor.


## Setup

### 1. Install containerd and shims

Install [containerd](https://containerd.io/) and the runtime shim you plan to use (typically `containerd-shim-runc-v2`). Start the daemon and confirm:

```bash
systemctl enable --now containerd
ls /run/containerd/containerd.sock
ls /etc/containerd/config.toml
which containerd-shim-runc-v2
```

#### 1.1 Install Kata

Install [Kata](https://github.com/kata-containers/kata-containers) runtime shim from the github releases page: https://github.com/kata-containers/kata-containers/releases/tag/3.31.0 

Download the platform appropriate file and extract it. Then copy the `containerd-shim-kata-v2` binary (likely from `opt/kata/bin` of the extracted `opt` folder) to `/usr/local/bin/` or symlink it. Then confirm:
```bash
which containerd-shim-kata-v2
```

Follow the instructions at [Kata containerd install](https://github.com/kata-containers/kata-containers/blob/main/docs/install/container-manager/containerd/containerd-install.md) for further details.

### 2. Install CNI plugins

Download the [containernetworking/plugins](https://github.com/containernetworking/plugins) release bundle and install these binaries:

```bash
sudo mkdir -p /opt/cni/bin
sudo cp bridge host-local portmap firewall /opt/cni/bin/
sudo chmod +x /opt/cni/bin/*
```


### 3. Prepare networking on the host

Enable IP forwarding:
```bash
sudo sysctl -w net.ipv4.ip_forward=1
```

To persist IP forwarding, add `net.ipv4.ip_forward=1` in `/etc/sysctl.conf`.

Additionally, if installing from source or downloaded the zip binaries, make sure to create the cni plugin configuration file `80-nunet-bridge.conflist` in `/etc/cni/net.d` with the content from [maint-scripts/nunet-dms/etc/cni/net.d/80-nunet-bridge.conflist](../../maint-scripts/nunet-dms/etc/cni/net.d/80-nunet-bridge.conflist)

### 4. Run DMS and use the executor

Start DMS as root on the compute provider node. In your ensemble, set the allocation execution type to `containerd`:

```yaml
execution:
  type: containerd
  params:
    image: docker.io/library/busybox:latest
    runtime: runc   # (optional) default is runc, can be kata
```

Port bindings use the same `PortsToBind` mechanism as the docker executor. Traffic is forwarded through the CNI bridge and DMS subnet rules.

## Defaults

| Setting | Path / value |
|---------|----------------|
| containerd socket | `/run/containerd/containerd.sock` |
| containerd config | `/etc/containerd/config.toml` |
| CNI plugins | `/opt/cni/bin` |
| CNI config dir | `/etc/cni/net.d` |
| CNI network name | `nunet-bridge` |
| Bridge interface | `cni-nunet0` |
| Netns directory | `/var/run/netns` |
| containerd namespace | `nunet` |
