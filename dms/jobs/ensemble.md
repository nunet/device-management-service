# EnsembleConfig Documentation

## Overview

EnsembleConfig is a versioned structure that defines the configuration for an ensemble of interconnected nodes, allocations, and network constraints that will be used for a deployment.

## Structure Summary

The following yaml file contains the available fields of an ensemble with the explanation for each field:

```
version: "V1"

# (Optional) Escalation strategy for the ensemble; options include "redeploy" and "tear_down"
escalation_strategy: redeploy

allocations:
  # Named allocation configurations in the ensemble
  alloc1:
    # Specifies the executor for this allocation; options include "docker", "firecracker", etc.
    executor: docker
    resources:
      # CPU resource requirements for the allocation
      cpu:
        clockspeed: 2  # (Optional) Clock speed of CPU in Hz
        cores: 2       # Number of CPU cores required
        model: ""      # (Optional) Model of the CPU
        vendor: ""     # (Optional) Vendor of the CPU (e.g., Intel, AMD)
        threads: 2     # (Optional) Number of threads per core
        architecture: "" # (Optional) Architecture of the CPU (e.g., x86, ARM)
        cachesize: 0   # (Optional) Cache size

      #  (Optional) GPU resource requirements, if applicable
      gpus:
        - vendor: "Nvidia" # (Optional) Vendor of the GPU (e.g., Nvidia)
          model: ""        # (Optional) Model of the GPU
          vram: 2      # (Optional) VRAM amount

      # RAM resource requirements for the allocation
      ram:
        size: 2     # RAM size in MB
        clockspeed: 3  # (Optional) RAM clock speed in Hz
        type: ""       # (Optional) Type of RAM

      # Disk resource requirements for the allocation
      disk:
        size: 2    # Disk size in MB
        model: ""      # (Optional) Model of the disk
        vendor: ""     # (Optional) Vendor of the disk
        type: ""       # (Optional) Disk type (e.g., SSD, HDD)
        interface: ""  # (Optional) Disk interface (e.g., SATA, NVMe)
        readspeed: 0   # (Optional) Read speed
        writespeed: 0  # (Optional) Write speed

    # Execution configuration for the allocation
    execution:
      type: docker            # Type of execution environment (e.g., docker, firecracker vm)
      cmd:                    # (Optional) Command to run in the execution environment
      entrypoint:             # (Optional) Entry point script or executable
      environment:            # (Optional) List of environment variables for the allocation
        - env1
      image: image-name       # Name of the container image or VM image
      working_directory: /    # Working directory within the execution environment

    # Network configuration for the allocation
    dnsname: mydocker         # Internal DNS name for this allocation

    # List of SSH keys authorized for this allocation
    keys: []

    # List of provisioning scripts, executed in the order specified
    provision: []

    # Health check script name for verifying allocation health
    healthcheck: ""

    # Failure recovery strategy for the allocation; options include "stay_down", "one_for_one", "one_for_all", and "rest_for_one"
    failure_recovery: stay_down

    # (Optional) List of allocations that this allocation depends on
    depends_on: ["alloc2"]


nodes:
  # Named node configurations in the ensemble
  node1:
    allocations:
      - alloc1               # List of allocations assigned to this node

    # (Optional) redundancy factor for the node
    redundancy: 1

    # Failure recovery strategy for the node; options include "stay_down", "restart", "redeploy"
    failure_recovery: stay_down

    ports:
        public: 1            # Number of public ports requested
        private: 1           # The private port mapping
        allocation: alloc1   # Name of the allocation associated with this port

    # (Optional)
    location:
      # List of acceptable locations for this
      accept:
        # regions: Africa, Antarctica, Asia, Europe, North America, Oceania, South America
        - region: Europe
          # iso code of country
          country: "US"
          city: "Mountain View"
          asn: 15169
          isp: "Google LLC"
      reject: []             # List of locations to avoid (e.g., restricted for GDPR)

    peer: peeridhere         # (Optional) Fixed peer ID that we want to deploy to


# List of network edge constraints defining connectivity between nodes
# Each edge includes source (S) and target (T) nodes, round-trip time (RTT) limits, and bandwidth (BW) constraints
# (Optional)
edge_constraints:
  - edges: ["edge1", "edge2"]
    rtt: 0
    bw: 0
    symetric: false

# (Optional) supervisor
supervisor:
  # (Optional) Supervision strategy for the ensemble
  # available values: OneForOne, AllForOne, RestForOne
  strategy: ""
  alloctions:
    - alloc1
    - alloc2
  children:
    - strategy: ""
      alloctions:
        - allocation3
        - allocation4

# (Optional)
keys:
  # Named SSH keys relevant to allocations
  keys1: "/home/dave/.ssh/id_dbr.pub"
  keys2: "/home/dave/.ssh/id_ed25519"

# (Optional)
scripts:
  # Named provisioning scripts for allocations or nodes
  script1: "nunet.sh"
  script2: "main.go"

```

