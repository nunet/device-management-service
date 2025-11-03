# Deploying Workloads on the Network

## Overview

Every node on the NuNet network has the ability to deploy jobs across the network's available compute resources, given the necessary capabilities. You can leverage the distributed computing power offered by compute providers to run your workloads efficiently.

## How

1. Ensure your DMS is running and properly connected to the network
2. Create an ensemble configuration file that defines your deployment requirements
3. Use the DMS CLI to deploy your jobs

### Creating an Ensemble Configuration

An ensemble configuration defines the resources and requirements for your deployment. Here's a basic example:

```yaml
version: "V1"

allocations:
  alloc1:
    type: task
    executor: docker
    resources:
      cpu:
        cores: 1
      gpus: []
      ram:
        size: 4 # GiB
      disk:
        size: 2 # GiB
    execution:
      type: docker
      image: hello-world
    dnsname: mydocker
nodes:
  node1:
    allocations:
      - alloc1
```

For more details about the ensemble configuration format and all possible fields, see the [EnsembleConfig Documentation](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/jobs/ensemble.md?ref_type=heads).

### Deploying Jobs

To deploy a job:

1. Save your ensemble configuration to a file (e.g., `ensemble.yaml`)
2. Use the DMS CLI to create a new deployment:

```bash
nunet actor cmd --context user /dms/node/deployment/new -f ensemble.yaml
```

## Monitoring Deployments

You can monitor your deployments using these commands:

```bash
# List all deployments
nunet actor cmd --context user /dms/node/deployment/list

# Get status of a specific deployment
nunet actor cmd --context user /dms/node/deployment/status --id <deployment_id>

# Get deployment manifest
nunet actor cmd --context user /dms/node/deployment/manifest --id <deployment_id>
```

## Best Practices

1. Monitor resource usage and costs
2. Use appropriate resource constraints in your ensemble configurations
3. Implement proper error handling in your deployments

## Troubleshooting

Common issues and their solutions:

1. **Deployment fails to start**

   - Verify your ensemble configuration
   - Ensure proper network connectivity (e.g.: list peers using `nunet actor cmd --context user /dms/node/peers/list`)

2. **Capabilities**

   - Ensure you have the required capabilities to invoke deployment behaviors upon peers on the network
