# NuNet Command-Line Interface (CLI)

## Introduction

This manual provides instructions on how to use the NuNet Command Line Interface (CLI) to onboard a device, manage resources, wallets, and interact with peers.

## Getting Started

1. Open the terminal and run the following command to access the CLI:

```
nunet
```

## Usage

The CLI provides several commands and options for managing your device on the NuNet platform. The general syntax is:

```
nunet [OPTIONS] COMMAND
```

## Commands

Here's the complete list of the command line options that can be used with the CLI:

- `capacity`: Display capacity of device resources
- `wallet`: Get Current Wallet Address
- `onboard`: Onboard the device to NuNet
- `info`: Get information about the currently onboarded machine
- `onboard-gpu`: Install NVIDIA GPU driver and Container Runtime
- `onboard-ml`: Prepare the system for Machine Learning with GPU
- `resource-config`: Change the configuration of onboarded device
- `shell`: Send commands and receive answers to a vm instance via DMS
- `peer`: Interact with currently visible peers
- `chat`: Start, Join, or List Chat Requests
- `log`: Returns the path of an archive containing all log files

Let's look into each of them and how they work.

## Onboarding a Device

1. Check the resources capacity on your device by running the following command:

```
nunet capacity --pretty
```

2. If you don't have an existing wallet address, create a new one using either Ethereum or Cardano blockchain (We currently recommend using Cardano at the moment as this is the primary blockchain for testing and will be the focus for our Public Alpha) but have included both as NuNet is a multichain protocol and will support many chains in the future:

- For Cardano:

```
nunet wallet new --cardano
```

- For Ethereum:

```
nunet wallet new --ethereum
```

When we support other blockchains in the future, you would simply need to change the blockchain name when creating a wallet through the above command.
Make sure you backup the mnemonic and wallet address for safe keeping. Do not share it with anyone. This is the same wallet address that you would be providing on the Compute Provider Dashboard.

3. Onboard your device to the NuNet platform using the following command:

```
nunet onboard -m <memory in MB> -c <cpu in MHz> -n nunet-test -a <address> [-C] [-l]
```

Replace `<memory in MB>`, `<cpu in MHz>`, and `<address>` with the appropriate values based on your device's resources (noted in onboarding step 1) and your wallet address.

For example,

```
nunet onboard -m 4000 -c 15000 -n nunet-test -a addr1q8pakf7kuac2fupvvwym4nq9rvu80vd5cvdtp2h0gpg8ppeetw8gxhrfckc4q3gjdg2eprnezpyx6sjauqj4mevleavql8n8kd 
```

- The `-C` option is optional and allows deployment of a Cardano node. Your device must have at least 10,000 MB of memory and 6,000 MHz of compute capacity to be eligible.
- The `-l` option is optional but important. Use `-l` when running the DMS on a local machine (e.g., a laptop or desktop computer) to enable advertisement and discovery on a local network address. Do not use `-l` when running the DMS on a machine from a datacenter.

4. Onboard your NVIDIA GPU

Install the NVIDIA GPU driver and container runtime. To run this command, use the following command:

```
nunet onboard-gpu
```

This command will work on both native Linux (Debian) and WSL machines. It also checks for Secure Boot if
necessary.

5. Prepare the system for Machine Learning

Prepare the system for machine learning with GPU. We include this step to reduce the time for starting jobs
because of large-sized GPU based ML images of TensorFlow and PyTorch. To do this, use the following
command:

```
sudo nunet onboard-ml
```

The above command preloads (downloads) the latest ML on GPU images for training/inferencing/computing
on NuNet.

4. Wait a few minutes for components to start and peers to be discovered.

5. Check your peer information and the peers your DMS is connected to by running the following commands:

​		You can lookup connected peers. To list visible peers, use the following command:

```
nunet peer list
```

​		To know you own peer info, use:

```
nunet peer self
```

If you see other peers in the list, congratulations! Your device is successfully onboarded to NuNet. If you only see your node, don't worry. It may take time to find other peers, especially if your device is behind symmetric NAT.


6. At any time after onboarding, you can also check how much resources had been allocated to NuNet previously with the following command:

```
nunet capacity --pretty --onboarded
```

​		To check your machine's full capacity, you can always use:

```
nunet capacity --pretty --full
```


## Enter a NuNet Peer's Shell

You can also send commands and receive answers to a VM instance via DMS. To do that, use the following format:

```
nunet shell --node-id <node-id>
```

The node-id can be obtained from the `nunet peer list` command. For example:

```
nunet shell --node-id Qmd8GeqGmdkQc5arhEs4i9tPRFNJoFLLURsBZsY9Riu4Kw
```

## Chat with Peers

To start a chat with a peer, use the following format:

```
nunet chat start <node-id>
```

For example:

```
nunet chat start Qmd8GeqGmdkQc5arhEs4i9tPRFNJoFLLURsBZsY9Riu4Kw
```

To list open chat requests:

```
nunet chat list
```

To clear open chat requests:

```
nunet chat clear
```

To join a chat stream using the request ID:

```
nunet chat join <request-id>
```

The request-id mentioned above can be obtained from the `nunet chat list` command stated earlier.

## Collect Logs

You can return the path of an archive containing all NuNet log files. To run this command, use: 

```
nunet log
```

This should return the path to the archive containing the log files, such as `/tmp/nunet-log/nunet-log.tar`.

## Display NuNet System Configuration with ML Readiness

Get information about the currently onboarded machine. To run this command, use the following command:

```
nunet info
```

## Check the GPU status in real time

To use this option, add the `-gpu` or the `--gpu-status` flag to `capacity` command like this:

```
nunet capacity --gpu-status
```

This allows you to check the GPU utilization, memory usage, free memory, temperature and power draw when the machine is idle or busy.

## Check the availability of CUDA and Tensor Cores

To use this option, add it to the `capacity` command like this:

```
nunet capacity --cuda-tensor
```

As a shorter alternative, you can also use `-ct`. To perform this check, the command leverages the NuNet PyTorch container used for `onboard-ml`.

## More Information

Additionally, you can find NuNet Network Status Dashboard at https://stats-grafana.dev.nunet.io/ for real-time statistics about computational processes executed on the network, telemetry information, and more.
