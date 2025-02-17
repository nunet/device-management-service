## Setup GlusterServer

```
sudo apt install glusterfs-client
```

```
sudo modprobe fuse
```

## Docker


#### Download image

```
docker pull ghcr.io/gluster/gluster-containers:centos
```

#### Create directories
create these dirs on host:

```
/etc/glusterfs
/var/lib/glusterd
/var/log/glusterfs
```

#### Run

```
docker run --name gluster-container -v /etc/glusterfs:/etc/glusterfs:z -v /var/lib/glusterd:/var/lib/glusterd:z -v /var/log/glusterfs:/var/log/glusterfs:z -v /sys/fs/cgroup:/sys/fs/cgroup:rw -d --privileged=true --net=host -v /dev/:/dev -v /glusterfs_data:/data --cgroupns=host ghcr.io/gluster/gluster-containers:centos
```

### Create volume

my hostname is `host`

bash the container
```
docker exec -ti gluster-container bash
```

```
gluster volume create nunet_vol ${hostname}:/data/brick1 force // Please provide a valid hostname/ip other than localhost, 127.0.0.1 or loopback address (0.0.0.0 to 0.255.255.255).

gluster volume start nunet_vol
```

```
sudo mount -t glusterfs ${hostname}:/nunet_vol /mnt
```

