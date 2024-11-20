# Onboarding as a Compute Provider

## Overview

As a compute provider in the NuNet network, you can offer your computer's resources (CPU, RAM, GPU, and storage) to other network participants.

This guide will walk you through the process of onboarding your resources to the network.

## Prerequisites

Before onboarding, ensure:

- Your DMS is properly installed and running
- You have met the [minimum system requirements](../README.md#system-requirements)
- You have the necessary [permissions and features](../README.md#permissions-and-features) configured
- (Optional) For GPU providers: GPU drivers are correctly installed

## Onboarding Process

### Onboard

Use the following command to onboard your resources:

```bash
nunet actor cmd --context user /dms/node/onboarding/onboard --ram <GB> --cpu <cores> --disk <GB>
```

Example:

```bash
# Onboard 4GB RAM, 2 CPU cores, and 20GB disk space
nunet actor cmd --context user /dms/node/onboarding/onboard --ram 4 --cpu 2 --disk 20
```

## Managing Your Resources

### Verify Onboarding Status

Check your onboarding status:

```bash
nunet actor cmd --context user /dms/node/onboarding/status
```

### Checking Resource Status

Monitor your resources allocation:

```bash
# Check allocated resources
nunet actor cmd --context user /dms/node/resources/allocated

# Check onboarded resources
nunet actor cmd --context user /dms/node/resources/onboarded
```

### Offboarding Resources

To remove the availability of your machine's resources from the network, execute:

```bash
# Offboard current resources
nunet actor cmd --context user /dms/node/onboarding/offboard
```

### Modifying Onboarded Resources

> Note: we'll facilitate this process in the future so that users do not have to reonboard again.

To modify your resource allocation:

1. First, offboard your current resources
2. Then onboard again with new resource values

```bash
# Offboard current resources
nunet actor cmd --context user /dms/node/onboarding/offboard

# Onboard with new values
nunet actor cmd --context user /dms/node/onboarding/onboard --ram <new_GB> --cpu <new_cores> --disk <new_GB>
```

## Best Practices

1. **Resource Allocation**

   - Don't onboard all available resources
   - Leave enough resources for system operations
   - Consider your system's stability and cooling capabilities

2. **System Maintenance**

   - Regularly update your DMS instance
   - Monitor system health and performance
   - Maintain stable internet connectivity

3. **Security**

   - Monitor system logs for unusual activity
   - Keep your system updated with security patches

## Troubleshooting

Common issues and solutions:

1. **Onboarding fails**

   - Verify system requirements are met
   - Ensure proper permissions are set

2. **Resource allocation issues**

   - Verify if other allocations are still running (check allocated resources)
   - Ensure proper GPU drivers if using GPUs
   - Verify available resources
   - Check for competing processes
