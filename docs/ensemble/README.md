# DMS Ensemble Format

## Introduction

Job deployments in the Device Management Service (DMS) revolve around **ensembles**: declarative manifests that describe the logical nodes you need, the allocations (containers for now) that run on those nodes, and the operational constraints that bind everything together. When you request a deployment, an orchestrator actor ingests the ensemble, either broadcasts the deployment request (request for bid) or sends it to a specifc peer, collects bids from providers that are in capability agreement with the orchestrator, evaluates constraints, and provisions allocations. Once bids are selected, a commit request is sent to selected bidders to reserve their resources for a short period. Allocations then get created, receive addresses for an overlay network on top of libp2p and proxied through raw quic connections so that they can communicate privately even when running on different physical peers.

## CLI Overview

- `nunet deploy -c <context_name> -f <ensemble.yaml> -t <timeout>`: hands an ensemble to `/dms/node/deployment/new` behavior, spins up an orchestrator if the ensemble file is valid, and sends the deployment request resulting in an ensemble ID / deployment ID.
- `nunet get deployments`: queries the orchestrator behavior `/dms/node/deployment/list` for deployments. The list contains ensemble ID and status.
- `nunet get allocations`: queries the node behaviors for allocations `/dms/node/allocations/list`. The list contains information about all allocations the node is hosting containing resources being consumed, orchestrtor peer ID, status etc...
- `nunet translate docker-compose.yml`: converts a Docker Compose spec into an ensemble description. Output by default goes to stdout along with errors on stderr. To avoid the mix, either redirect stdout to a file or use the flag `-o <file_path|name>` 
- `nunet validate <ensemble.yaml>`: runs a validator against the input ensemble so you can catch schema, constraint, and reference issues before calling `deploy`.

These commands are wrappers around the parser and behavior endpoints.

## The Ensemble

The ensemble format is structured to define all the necessary components for deploying workloads with the Device Management Service (DMS). Below is a detailed explanation of the key sections:

- **`allocations`**: This section defines the individual workload units that need to be deployed. Each allocation belogs to a Node in the deployment. An Allocation correspond to a single unit of execution such as a container. At the very least, it must specify the amount of resource its execution needs along with what executor to use and what to execute. Each allocation can specify:
    - **Type**: Whether the workload is a `service` (long-running) or a `task` (transient).
    - **Executor**: The runtime environment, such as `docker`.
    - **Resources**: The hardware requirements, including CPU cores, GPU specifications, RAM, and disk space.
    - **Execution Parameters**: Executor-specific details, such as the Docker image, command to run, working directory etc...
    - **DNS Name**: A name used to address the allocation within the overlay network amongst allocations.
    - **Keys**: For injecting keys into the runtime environment. Currently ssh supported.
    - **Provision Script**: Scripts to run before the actual job in the runtime environment.
    - **Health Checks**: Commands and expected responses to ensure the workload is running correctly.
    - **Failure Recovery**: Strategies to handle failures, such as retries or restarts.
    - **Dependencies**: Other allocations that this workload depends on, ensuring proper startup order.

- **`nodes`**: This section maps allocations to physical or virtual nodes (peers) in the DMS network. Each node can include:
    - **Peer**: The specific DMS peer where the node should run. Specifying `peer` avoids broadcasting a bid request for this node.
    - **Location Rules**: Criteria to accept or reject nodes based on geographic or other constraints.
    - **Redundancy**: The ability to define standby nodes that can take over in case of failure.
    - **Ports**: Public-to-private port mappings for network communication, tied to specific allocations.

- **`subnet`**: This section controls the overlay network that connects allocations. Setting `join: true` allows the orchestrator to participate in the libp2p/QUIC VPN, enabling direct communication with allocations. If not set, only the allocations will be able to communicate with each other over the network.

- **Additional Sections**:
    - **`metadata`**: Any key-value pair metadata about the deployment. Metadata values can be used to filter deployments with. For example, the actor cmd `/dms/node/deployment/list` or alias `get deployments` can accept the `--filter <key>=<value>` flag and arg to filter the list of deployments matching the `<value>` for the specified `<key>`
    - **`exclude_peers`**: A blacklist of providers/nodes peerIDs that should not participate in the deployment.


### Deployment Scenarios

#### 1. Single Node, Single Service Allocation

```yml
version: "V1"

allocations:
    nginx1:
        type: service
        resources:
            cpu:
                cores: 1
            gpus: []
            ram:
                size: 1 # in GB
            disk:
                size: 1 # in GB
        executor: docker
        execution:
            type: docker
            image: nginxdemos/hello:plain-text
            working_directory: /
        healthcheck:
            type: command
            exec: ["curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost"]
            response:
                type: string
                value: "200"
        dns_name: alloc1

nodes:
    node1:
        allocations:
            - nginx1
        ports:
            - private: 80
              public: 16480
              allocation: nginx1
```

This scenario describes almost all fields for the allocation. 

```yml
allocations:
    nginx1:
        type: service
```
`allocations` starts the allocation description level. `nginx1` is the name of the allocation and because `type: service`, the job keeps running indefinitely and runs a health check every 30 seconds until the orchestrtor decides to shutdown assuming no machine goes down in the meantime.

```yml
resources:
    cpu:
        cores: 1
    gpus: []
    ram:
        size: 1 # in GB
    disk:
        size: 1 # in GB
```

The `resources` section describes the amount required for the job which comes into effect twice: 1, when requesting for bids where bidders will not submit a bid if they do not has at least the requested amount onboarded and free. 2, to constrain the job to those limits when it's running. 


```yml
executor: docker
execution:
    type: docker
    image: nginxdemos/hello:plain-text
    working_directory: /
```

Describes the executor we're looking to run this job on, which is "docker." For the docker executor/runner, specifying `image` is mandatory. Other parameters, such as `entrypoint`, `cmd`, and `working_directory`, may be set by the image itself. Additionally, if the image to be deployed is located in a private registry, it is possible to use the `registry_auth` field. Through this field, a username and password can be specified to authenticate with the registry and allow the compute provider to pull the image.
```yml
execution:
  type: docker
  image: registry.private.example/image/path
  working_directory: /
  registry_auth:
    username: theuser
    password: thepassword
```
Note that credentials placed in the ensemble will be visible to the compute provider and, therefore, should not be used if the compute provider is not trusted. Consider making a specific image public for this purpose instead of including a username and password for a registry in an ensemble spec. This approach is particularly useful in scenarios where the compute providers are known and trusted, such as within a private cluster of an organization.


```yml
healthcheck:
    type: command
    exec: ["curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost"]
    response:
        type: string
        value: "200"
```

The `healthcheck` section describes a command to run in an interval to make sure the app running inside the allocation's executor is healthy. For this example, we're querying the nginx itself on localhost witth `curl -s -o /dev/null -w "%{http_code}" http://localhost` to get only the status code output so that the health checker routine can match it against the expected output of "200"

```yml
dns_name: alloc1
```

The filed `dns_name` is more relevant in the case of the multiple allocations or the orchestrator joining the overlay network subnet. It helps other allocations and the orchestrator identify the allocation with its name "alloc" instead of its IP which is generated during the provisioning stage of deployment randomly based on free IP addresses. If the `dns_name` field wasn't specified, the allocation will default to using its name, which in this case is "nginx1".

```yml
nodes:
  node1:
    allocations:
      - nginx1
    ports:
      - private: 80
        public: 16480
        allocation: nginx1
```

The `nodes` section lists all the nodes needed for this deployment. The "node1" is the name of the node and the `alloctions` field specifies which allocation in the spec should be deployed on this node. It's possible to use the same allocation accross different nodes by simply adding more nodes and naming the same allocation.

The `ports` section describes what ports to map on the node for the allocation. In this example, `private` port `80` specifies port 80 on the container. Whereas `public` port `16480` describes port on the node/host. This will make the allocation "nginx1" map port `16480:80` on the docker executor. The public 16480 will also be availble on the overlay network amongst the allocations.

#### 2. Single Node, Single Task Allocation

```yml
allocations:
  alloc1:
    type: task
    executor: docker
    resources:
      cpu:
        cores: 1
      gpus: []
      ram:
        size: 1
      disk:
        size: 1
    execution:
      type: docker
      image: ubuntu:24.04
      cmd: ["echo", "Hello, World"]
nodes:
  node1:
    allocations:
      - alloc1
```

This job will only print a "Hello, World" text from an ubuntu:24.04 image and exit immediately. For that reason, the allocation is marked as `type: task`  - this will cause the orchestrator to consider it completed when it exits by itself unlike `service` type jobs which are considered to be in error if exited without a shutdown from Orchestrator.


#### 3. Multi-Node, Mixed Allocations with Overlay Networking and GPU resource

```yml
version: "V1"

allocations:
    nginx1:
        type: service
        executor: docker
        resources:
            cpu:
                cores: 1
            gpus:
                - vendor: NVIDIA
                  vram: 2
            ram:
                size: 1
            disk:
                size: 1
        execution:
            type: docker
            image: nginxdemos/hello:plain-text
            working_directory: /
        healthcheck:
            type: command
            exec: ["curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost"]
            response:
                type: string
                value: "200"
        dns_name: alloc1

    nginx2:
        type: service
        executor: docker
        resources:
            cpu:
                cores: 1
            gpus: []
            ram:
                size: 1
            disk:
                size: 1
        execution:
            type: docker
            image: nginxdemos/hello:plain-text
            working_directory: /
        keys: []
        provision: []
        healthcheck:
            type: command
            exec: ["nginx", "-t"]
            response:
                type: string
                value: "nginx: the configuration file /etc/nginx/nginx.conf syntax is ok"
        dns_name: alloc2

nodes:
    node1:
        allocations:
            - nginx1
        ports:
            - private: 80
              public: 16480
              allocation: nginx1
    node2:
        allocations:
            - nginx2
        ports:
            - private: 80
              public: 16481
              allocation: nginx2

subnet:
    join: true
```
This ensemble extends the first one by adding one other node with an additional allocation. The second allocation does a different type of health check just as an example. It also specifies the spec:
```yml
subnet:
    join: true
```

to make the orchestrator join the subnet. It will allow the orchestrator to be able to directly communicate with the allocations through the overlay network. Both allocations specify different port maps on the public network to avoid confilct.


### Port Mapping

- Each `nodes.<name>.ports` entry specifies `public`, `private`, and `allocation`. The validator forces public ports into a range specified by the provider. The config values `PortAvailableRangeFrom` and `PortAvailableRangeTo` default to values 16384 and 65536 but the compute provider can adjust as needed. If the port requested by the orchestrator isn't within the range, the provider will not be able to allocate. 

## Putting It All Together

1. Write a new ensemble or translate from a docker-compose using the structures above. Translating needs a human review since there isn't a one-to-one relationship between docker-compose ensemble specs.
2. Run `nunet validate ensemble.yaml` to validate locally before deployment.
3. Deploy with `nunet deploy -f ensemble.yaml -t 5m`
4. Monitor with `nunet get deployments`, interact with allocations via overlay network if orchestrator has joined the subnet.

