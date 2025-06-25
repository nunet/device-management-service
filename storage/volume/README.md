# GlusterFS

## Glusterfs client is now in its own image

I decided to put the glusterfs client in its own container instead of running `mount -t glusterfs` and installing the client on the host system.

There are 2 reasons:

1. Linux distro compatibility (some distro's dont ship the same version and the `mount` command doesnt work)
2. Most important, it isolates each glusterfs clients certificate/key in its own container

to build the image:

```
make build-nunet-glusterfs-client
```

load the fuse kernel module:

```
sudo modprobe fuse
```

it will create an image named: `nunet-glusterfs-client`

## Server Configuration

### Run glusterfs server container:

```
docker run --name gluster-server-container -v /sys/fs/cgroup:/sys/fs/cgroup:rw -d --privileged=true --net=host --cgroupns=host ghcr.io/gluster/gluster-containers:fedora
```

use docker exec to gain shell access to the container:
```
docker container exec -it gluster-server-container bash
echo "option transport.socket.ssl-cert-depth 3" >  /var/lib/glusterd/secure-access
```


We need to make sure a glusterfs server is setup with proper keys. We need to generate x509 certificates for both server and clients to authenticate and encrypt transit data.

For server we generate:
```
openssl genrsa -out glusterfs.key 2048
openssl req -new -x509 -key glusterfs.key -subj "/CN=puthotnamehere" -out glusterfs.pem
```

For this test, we will be using the docker glusterfs container which is running in net=host mode, so we can use the host's hostname for now.

Replace CN=puthotnamehere with your hostname in the above command

Now we need to copy these certs to the glusterfs container at `/etc/pki/tls`

If openssl isnt available, create them on your machine and then copy them:

```
docker cp glusterfs.key gluster-server-container:/etc/pki/tls/
docker cp glusterfs.pem gluster-server-container:/etc/pki/tls/
```

### Deploy DMS to glusterfs container

Make sure glusterfs server container is started, and then copy the dms binary to the container :

```
docker cp dms gluster-server-container:/home/
```

Make sure you create key and caps before running dms

Make sure to start dms on the container with the following configuration:

```
storage_mode=true
storage_ca_directory=somewhere or there is default value
storage_bricks_dir=somewhere you want all the data to be stored for each volume
storage_glusterfs_hostname=for this test it can be your host's hostname
```

```
mkdir -p /root/.nunet/storage_ca_directory/glusterfs_nodes
```

Copy the `/etc/pki/tls/glusterfs.pem` file to `${storage_ca_directory}/glusterfs_nodes/` directory.

```
cp /etc/pki/tls/glusterfs.pem /root/.nunet/storage_ca_directory/glusterfs_nodes
```

Explanation:

Each storage dms, has to keep the server pem files in one place so it can create a chain CA file and send it back to the clients when they create a volume. For this purpose inside `storage_ca_directory` we have the following:

```
/glusterfs_nodes
/clients
```

### On your host run another DMS instance that will connect to the dms we ran in the container

Note that we need to give this dms capabilities to run the following behaviours on the glusterfs dms:

```
VolumeCreateBehavior = "/dms/volume/create"
VolumeDeleteBehavior = "/dms/volume/delete"
VolumeStartBehavior  = "/dms/volume/start"
```


## Client Side

### 1. Generate glusterfs client key,certificate

```
make generate-glusterfs-client-certs CN=clientX
```

here we are defining clientX as the Common name of the cert. We could use a key did here for example.

At this point we have the a directory where the clients certificates are stored.

### 2. Create a volume

On the host dms create a volume and obtain the CA file returned from the glusterfs dms.

```
./dms actor cmd --context dms /dms/volume/create --name testingdms30march --client-pem-file /home/glusterfs_certificates/glusterfs.pem --ca-output-dir "/home/glusterfs_certificates/"  --dest {dms peer/did on the glusterfs container}
```

Now if we go to the `client certificate directory` we should see one additional `.ca` file.

```
glusterfs.key
glusterfs.pem
glusterfs.ca <- new file
```

### 3. Start the volume

```
./dms actor cmd --context dms /dms/volume/start --name testingdms30march --dest {dms peer/did on the glusterfs container}
```

### 4.A. Reload gluster

For now, since we are not using a CA Root for the glusterfs server, each glusterfs is their own "Roots". We need to do one manual step which will be removed in future by introducing a Root Authority that will allow certificate of depth of more than 0 which means just by keeping the intermidiate certifivates we will be able to still verify the chain and no need each time to restart/reload.

On the glusterfs server
```
cd ${storage_ca_directory}
cp glusterfs.ca /etc/pki/tls/
systemctl restart glusterfsd
```

### 4.B. Run an allocation with storage

We can run an allocation with storage now:

```
  nginxAllocWithStorage2:
    type: service
    executor: docker
    resources:
      cpu:
        cores: 1
      gpus: []
      ram:
        size: 1 # in GB
      disk:
        size: 1 # in GB
    execution:
      type: docker
      image: nginxdemos/hello:plain-text
      working_directory: /
    dnsname: mydocker
    keys: []
    provision: []
    healthcheck:
      type: command
      exec: ["nginx", "-t"]
      response:
        type: string
        value: "nginx: the configuration file /etc/nginx/nginx.conf syntax is ok"
    dns_name: nginxdemo-alloc2
    volume:
      - type: glusterfs
        name: nunet_vol2
        servers: ["${hostname}"]
        mount_destination: "/home"
        client_private_key:
        client_pem:
        client_ca:
```

The `volume` shows the requirements.
We need to pass the the client auth data so it can read client certs and ca file we created earlier

after the allocation is started, go to the container and check the mount volume. Write to it and then go the the glususterfs container where the bricks storage directory is and observe the data.

### 4.C. Test manually without an ensemble

You can always run directly from your host:

```
docker run -d --privileged \
    --name glusterfs-temp1 \
    -e GLUSTER_VOLUME=nunet_vol \
    -e GLUSTER_HOST=host \
    -e MOUNT_PATH=/mounted \
    -v /home/folderxyz:/mounted \
    -v /path/to/glusterfs.pem:/tmp/glusterfs.pem \
    -v /path/to/glusterfs.key:/tmp/glusterfs.key \
    -v /path/to/glusterfs.ca:/tmp/glusterfs.ca \
    nunet-glusterfs-client
```

and unomunt:

```
docker run --rm --privileged \
    -e GLUSTER_VOLUME=nunet_vol \
    -e GLUSTER_HOST=host \
    -e MOUNT_PATH=/magic \
    -e UNMOUNT=true \
    glusterfs-temp1
```


# Other Notes


## dms on the glusterfs config

```
{
  "apm": {
    "api_key": "",
    "environment": "production",
    "server_url": "http://apm.telemetry.nunet.io",
    "service_name": "nunet-dms"
  },
  "general": {
    "data_dir": "/root/nunet/data",
    "debug": false,
    "port_available_range_from": 16384,
    "port_available_range_to": 32768,
    "storage_bricks_dir": "/root/.nunet/storage_bricks_dir",
    "storage_ca_directory": "/root/.nunet/storage_ca_directory",
    "user_dir": "/root/.nunet",
    "work_dir": "/root/nunet",
    "storage_mode": true,
    "storage_glusterfs_hostname": "host"
  },
  "job": {
    "allow_privileged_docker": false
  },
  "observability": {
    "elasticsearch_api_key": "",
    "elasticsearch_enabled": false,
    "elasticsearch_index": "nunet-dms",
    "elasticsearch_url": "http://localhost:9200",
    "flush_interval": 5,
    "insecure_skip_verify": true,
    "log_file": "/root/nunet/logs/nunet-dms.log",
    "log_level": "INFO",
    "max_age": 28,
    "max_backups": 3,
    "max_size": 100
  },
  "p2p": {
    "bootstrap_peers": [],
    "fd": 512,
    "listen_address": [
      "/ip4/0.0.0.0/tcp/9001",
      "/ip4/0.0.0.0/udp/9001/quic-v1"
    ],
    "memory": 1024
  },
  "profiler": {
    "addr": "127.0.0.1",
    "enabled": true,
    "port": 6061
  },
  "rest": {
    "addr": "127.0.0.1",
    "port": 9991
  }
}
```


When running e2e tests these should be availabe on the machine

```
sudo modprobe fuse
docker pull ghcr.io/gluster/gluster-containers:fedora
docker pull nginxdemos/hello:plain-text
docker pull ubuntu:22.04
docker pull hello-world
sudo chmod 777 "/etc/glusterfs" "/var/lib/glusterd" "/var/log/glusterfs" "/glusterfs_data"
sudo sed -i 's/#user_allow_other/user_allow_other/g' /etc/fuse.conf
```