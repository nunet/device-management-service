# Device Management Service (DMS)

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

- [Device Management Service (DMS)](#device-management-service-dms)
  - [Table of Contents](#table-of-contents)
  - [About](#about)
    - [Payment](#payment)
  - [Installation](#installation)
    - [Binary releases](#binary-releases)
      - [Ubuntu/Debian](#ubuntudebian)
    - [Building from source](#building-from-source)
      - [Dependencies](#dependencies)
    - [Installation on VMs](#installation-on-vms)
    - [Installation on WSL](#installation-on-wsl)
    - [Permissions and features (for compute providers using Linux)](#permissions-and-features)
      - [Required: Net-admin permission and IP over libp2p](#net-admin-permission-and-ip-over-libp2p)
      - [May be required: iptables upgrade](#iptables-upgrade)
      - [Optional: Firecracker (for compute providers)](#optional-firecracker)
    - [System Requirements](#system-requirements)
      - [CPU-only machines](#cpu-only-machines)
        - [Minimum System Requirements](#minimum-system-requirements)
        - [Recommended System Requirements](#recommended-system-requirements)
      - [GPU Machines](#gpu-machines)
        - [Minimum System Requirements](#minimum-system-requirements)
        - [Recommended System Requirements](#recommended-system-requirements)
    - [GPU Driver Installation](#gpu-driver-installation)
  - [Usage](#usage)
    - [Quick Start](#quick-start)
      - [Creating identities](#creating-identities)
      - [Using a Ledger Wallet](#using-a-ledger-wallet)
      - [Setting up Capabilities](#setting-up-capabilities)
        - [Create capability contexts](#create-capability-contexts)
          - [Add a root anchor for your DMS context](#add-a-root-anchor-for-your-dms-context)
        - [Setup your DMS for the public testnet](#setup-your-dms-for-the-public-testnet)
      - [Running DMS](#running-dms)
    - [Provide Compute Power to the Network](#provide-compute-power-to-the-network)
    - [Deploy Jobs on the Network](#deploy-jobs-on-the-network)
    - [REST Endpoints](#rest-endpoints)
  - [Configuration](#configuration)
    - [Config file](#config-file)
      - [Run Two DMS Instances Side by Side](#run-two-dms-instances-side-by-side)
  - [Tests](#tests)
  - [Specification](#specification)
    - [Description](#description)
    - [Design and Architecture](#design-and-architecture)
    - [Functionality](#functionality)
    - [Data Types](#data-types)
    - [References](#references)

## About

**Device Management Service** or **DMS** enables a machine to join the decentralized NuNet network both as a compute provider, offering its resources to the network, or to leverage the compute power of other machines in the network for processing tasks. Eventually users with available hardware resources will get compensated whenever their machine is utilized for a computational job by other users in the network. The ultimate goal of the platorm is to create a decentralized compute economy that is able to sustain itself.

### Payment

All transactions on the Nunet network are expected to be conducted using the platform's utility token [NTX](https://docs.nunet.io/docs/v/getting-ntx). However, DMS is currently in development, and payment is not part of `v0.5.0` release. NTX payments are expected to be implemented in the Public Alpha Mainnet milestone within later release cycles.

**Note**: If you are a developer, please check out the [DMS specifications](#specification) and [Building from Source](#building-from-source) sections of this document.

## Installation

You can install Device Management Service (DMS) via [binary releases](#binary-releases) or [building it from source](#building-from-source).

### Binary releases

You can find all binary releases [here](https://gitlab.com/nunet/device-management-service/-/releases) and other builds in-between releases in the [package registry](https://gitlab.com/nunet/device-management-service/-/packages).
We currently support ARM and AMD64 architectures. You may check your architecture with appropriate command (`uname -p` for linux) and refer to the architecture name mapping e.g. [here](https://itsfoss.com/arm-aarch64-x86_64/) for figuring correct package to download.

**Note**: If you intalled the binary from a release and you would like to act as compute provider, you may need to check [permissions and features](#permissions-and-features) to enable some _required_ and optional features.

#### Ubuntu/Debian

1. Download the latest .deb package from the [package registry](https://gitlab.com/nunet/device-management-service/-/packages)

2. Install the Debian package with `apt` or `dpkg`:

```
sudo apt update
sudo apt install ./nunet-dms_<latest>.deb -y
```

3. Some dependencies such as `docker` and `libsystemd-dev` might be missing so it's recommended to fix install by running:

```
sudo apt -f install
```

### Building from source

We currently support Linux and MacOS (Darwin).

#### Dependencies

- iproute2 (linux only)
- build-essential (linux only)
- libsystemd-dev (linux only)
- go (v1.21.7 or later)
- [git-lfs](https://git-lfs.com/) (for downloading large files)

Clone the repository:

```
git clone https://gitlab.com/nunet/device-management-service.git
```

Configure git-lfs:

```
git lfs install && \
git lfs fetch && \
git lfs pull
```

Build the CLI:

```bash
cd device-management-service
make
```

This will result in a binary file in builds/ folder named as `dms_linux_amd64` or `dms_darwin_arm64` depending on the platform.

**Note**: If you built from source and would like to act as a compute provider,
you may need to check [permissions and features](#permissions-and-features) to enable some _required_ and optional features.

To cross compile to arm, cross compilers need to be installed. In particular arm-linux-gnueabihf and aarch64-linux-gnu.
For debian systems, install with:

```
apt install gcc-arm-linux-gnueabihf gcc-aarch64-linux-gnu
```

You can add the compiled binary to a directory in your `$PATH`. See the [Usage](#usage) section for more information.

### Permissions and features (for compute providers using Linux)

The following applies _only_ for **compute providers using Linux**. If you're running a client/orchestrator, you do _not_ need to set
any additional permissions.

> **Darwin users**: unfortunately, the DMS can't work with granular permissions on Mac.
> So, for now, if running a compute provider, you will have to run the nunet daemon (`nunet run`) as root.

For Linux users, granular permissions will have to be set to the binary (optionally, but _NOT_ recommended, you can run the binary as root).

#### Required: Net-admin permission and IP over libp2p

> **Note**: step **not** needed if you're using our **debian package**.
>
> It's needed for those building from source or downloading the binary releases.

**Note**: `cap_net_admin` is a **required** capability for **compute providers**.

Setting the permission enables IP over libp2p which is a feature that enhances the capabilities of compute providers, allowing them to participate in a wider
range of jobs. One capability enabled with this feature is to do port forwarding which it won't be possible without setting the right
unix permissions.

> One of the reasons for requiring this permission is because this feature depends on creating and managing tun interfaces.

To enable this feature, the `nunet` binary requires `network-admin` capabilities. These capabilities allow the application to perform
network configuration tasks without needing to run the entire application as root, which is a more secure approach.

To set the necessary capabilities, run the following command:

```shell
sudo setcap cap_net_admin+ep /usr/bin/nunet
```

The above command depends on: `libcap2-bin` (Debian/Ubuntu) or `libcap` (CentOS/RHEL/Arch...)

#### May be required: iptables upgrade

Some legacy versions of Linux `iptables` do not work with our IP over libp2p feature.

Check the version of yours by running:

```bash
iptables
```

If it's using the `nf_tables` version, you're fine. You can skip this step.

If it's using a legacy version, upgrade with:

```bash
sudo update-alternatives --config iptables
```

Then, select the number which corresponds to the `iptables-nft` option and press enter.

#### Optional: Firecracker (for compute providers)

To act as a compute provider capable of receiving and running jobs with Firecracker,
ensure your user is part of the `kvm` group. You can do this by running:

```shell
sudo usermod -aG kvm $USER
```

### Installation on VMs

- Skip doing an [unattended installation](https://www.virtualbox.org/manual/ch01.html#create-vm-wizard-unattended-install) for the new Ubuntu VM as it might not add the user with administrative privileges.
- Enable [Guest Additions](https://www.virtualbox.org/manual/ch04.html) when installing the VM (VirtualBox only).
- Always [change the default NAT network setting to Bridged](https://www.techrepublic.com/article/how-to-set-bridged-networking-in-a-virtualbox-virtual-machine) before booting the VM.
- [Install Extension Pack](https://phoenixnap.com/kb/install-virtualbox-extension-pack) if using VirtualBox (recommended).
- [Install VMware Tools](https://kb.vmware.com/s/article/1014294) if using VMware (recommended).
- ML on GPU jobs on VMs are not supported.

### Installation on WSL

- Install WSL through the Windows Store.
- Install the [Update KB5020030](https://www.catalog.update.microsoft.com/Search.aspx?q=KB5020030) (Windows 10 only).
- Install Ubuntu 20.04 through WSL.
- Enable [systemd on Ubuntu WSL](https://www.xda-developers.com/how-enable-systemd-in-wsl).
- ML Jobs deployed on Linux cannot be resumed on WSL.

Though it is possible to run ML jobs on Windows machines with WSL, using Ubuntu 20.04 natively is highly recommended to avoid unpredictability and performance losses.

If you are using a dual-boot machine, make sure to use the `wsl --shutdown` command before shutting down Windows and running Linux for ML jobs. Also, ensure your Windows machine is not in a hibernated state when you reboot into Linux.

### System Requirements

#### CPU-only machines

##### Minimum System Requirements

We require you to specify CPU (MHz x no. of cores) and RAM, but your system must meet at least the following requirements before you decide to onboard it:

- CPU: 2 GHz
- RAM: 4 GB
- Free Disk Space: 10 GB
- Internet Download/Upload Speed: 4 Mbps / 0.5 MBps

If the above CPU has 4 cores, your available CPU would be around 8000 MHz. So if you want to onboard half your CPU and RAM on NuNet, you can specify 4000 MHz CPU and 2000 MB RAM.

##### Recommended System Requirements

- CPU: 3.5 GHz
- RAM: 8-16 GB
- Free Disk Space: 20 GB
- Internet Download/Upload Speed: 10 Mbps / 1.25 MBps

#### GPU Machines

##### Minimum System Requirements

- CPU: 3 GHz
- RAM: 8 GB
- GPU: 4 GB VRAM (NVIDIA, AMD, or Intel discrete GPU with manually installed drivers)
- Free Disk Space: 50 GB
- Internet Download/Upload Speed: 50 Mbps

Note: For AMD64 platforms, we recommend using HiveOS as it comes with all necessary drivers pre-installed. For other setups, proper GPU drivers must be manually installed. See the [GPU Driver Installation](#gpu-driver-installation) section for instructions.

##### Recommended System Requirements

- CPU: 4 GHz
- RAM: 16-32 GB
- GPU: 8-12 GB VRAM (NVIDIA, AMD, or Intel discrete GPU with manually installed drivers)
- Free Disk Space: 100 GB
- Internet Download/Upload Speed: 100 Mbps

### GPU Driver Installation

NuNet DMS requires properly installed GPU drivers to function correctly. We do not automatically install drivers to ensure compatibility and flexibility across different user setups.

#### For AMD64 Platforms:

We recommend using the [Ubuntu](https://ubuntu.com/)-based [HiveOS](https://hiveon.com/install/) for the easiest setup.

If you prefer to use a different operating system or need to install drivers manually, please follow these steps:

#### NVIDIA GPUs:

1. Visit the [NVIDIA Official Driver Downloads](https://www.nvidia.com/en-us/drivers/) page.
2. Select your GPU model and operating system.
3. Download and install the recommended driver.
4. Install the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html).
5. Reboot your system after installation.

#### AMD GPUs:

1. Visit the [AMD Drivers and Support for Processors and Graphics](https://www.amd.com/en/support/download/drivers.html) page.
2. Select your GPU model and operating system.
3. Download and install the recommended driver.
4. Reboot your system after installation.

Along with the drivers, you will need to install amdgpu using ROCm for AMD GPUs. You can find the installation instructions [here](https://rocm.docs.amd.com/projects/install-on-linux/en/latest/install/amdgpu-install.html).

Make sure you select the rocm usecase when installing the amdgpu.

```bash
$ sudo amdgpu-install --usecase=rocm
```

#### Intel Discrete GPUs:

1. Visit the [Intel® software for general purpose GPU capabilities documentation](https://dgpu-docs.intel.com/driver/overview.html) page.
2. Select your GPU model and operating system.
3. Download and install the recommended driver.
4. Reboot your system after installation.

Along with the drivers, you will need to install XPU SMI for Intel GPUs. You can find the installation instructions [here](https://intel.github.io/xpumanager/smi_install_guide.html#).

For detailed instructions specific to your operating system, please refer to the documentation provided by NVIDIA, AMD, or Intel.

Note: Ensure that you have the correct permissions to install drivers on your system. On Linux systems, you may need to use `sudo` or log in as root to install drivers.

## Usage

### Quick Start

Before starting, ensure that you have properly installed GPU drivers if you're using a GPU-enabled machine. For AMD64 platforms, we recommend using HiveOS for the easiest setup. For other configurations, refer to the [GPU Driver Installation](#gpu-driver-installation) section for instructions.

This quick start guide will walk you through the process of setting up a Device Management Service (DMS) instance for the first time and getting it running. We'll cover creating identities, setting up capabilities, and running the DMS.

**The NuNet CLI**

The NuNet CLI is the command-line interface for interacting with the Nunet Device Management Service (DMS). It provides commands for managing keys, capabilities, configuration, running the DMS, and more. It's essential for setting up and administering your DMS instance.

**Key Concepts**

- **Actor:** An independent entity in the Nunet system capable of performing actions and communicating with other actors.
- **Capability:** Defines the permissions and restrictions granted to actors within the system.
- **Key:** A cryptographic key pair used for authentication and authorization within the DMS.

You can find a detailed documentation [here](./cmd/README.md).

#### Creating identities

The first step is to generate identities/keys and capability contexts. It is recommended that two keys are setup: one for the user (default name `user`) and another for the dms (default name `dms`)

A capability context is created with the `nunet cap new <context>` command and it is anchored on a key with the **same** context name.
The command automatically generates a key for the given context if not present. Keys can also be created manually with `nunet key new <key>` if you prefer.

> Note: If creating keys manually, make sure to use the same context name, otherwise it won't work.

In this example, we are going to set up two capability contexts:

First the user

```shell
$ nunet cap new user
```

then the dms instance.

```shell
$ nunet cap new dms
```

You can create as many identities as you want, specially if you want
to manage multiple DMS instances.

Each time a new identity is generated it will prompt the user
for a passphrase. The passphrase is associated with the created
identity, thus a different passphrase can be set up for each identity.
If you prefer, it's possible to set a `DMS_PASSPHRASE` environment variable
to avoid the command prompt.

The `key did` command returns a DID key for the specified identity.

Remember to secure your keys and capability contexts, as they control access to your NuNet resources.
They are encrypted and stored under `$HOME/.nunet` by default.

##### Using a Ledger Wallet

It is also possible to use a Ledger Wallet instead of creating a new
key; this is recommended for _user_ contexts, but you **should not** use it
for the _dms_ context as it needs the key to sign capability tokens.

To set up a _user_ context with a Ledger Wallet, you need the
`ledger-cli` script from [NuNet's ledger wallet tool](https://gitlab.com/nunet/ledger-wallet).
The tool uses the Eth application and specifically the first Eth
account with signing of personal messages. Everything that needs to be
signed (namely capability tokens) will be presented on your Nano's
screen in plaintext so that you can inspect it.

You can get your Ledger wallet's DID with:

```shell
$ nunet key did ledger
```

To create the capability context for the user

```shell
$ nunet cap new ledger:user
```

#### Setting up Capabilities

NuNet's network communication is powered by the [NuActor System](actor/README.md), a zero-trust system that utilizes fine-grained capabilities, anchored on [DIDs](https://www.w3.org/TR/did-core/), following the [UCAN model](https://github.com/ucan-wg/).

Once both identities are created, you'll need to set up capabilities. Specifically:

1. Create capability contexts for both the user and each of your DMS instances.
2. Add the user's DID as a _root anchor_ for the DMS capability context. This ensures that the DMS instance fully trusts the user, granting complete control over the DMS (the root capability).
3. If you want your DMS to participate in the public NuNet testnet (and eventually the mainnet), you'll need to set up capability anchors for your DMS:
   1. Create a capability anchor to allow your DMS to accept _public behavior invocations_ from authorized users and DMSs in the NuNet ecosystem.
   2. Add this token to your DMS as a _require anchor_.
   3. Request a capability token from NuNet to invoke public behaviors on the network.
   4. Add the token as a _provide anchor_ in your personal capability context.
   5. Delegate to your DMS the ability to make public invocations using your token.
   6. Add the delegation token as a _provide anchor_ in your DMS.

###### Add a root anchor for your DMS context

You can do this by invoking the `dms cap anchor` command:

```shell
$ nunet cap anchor --context dms --root <user-did>
```

Where `<user-did>` is the user did created above in [Creating identities](#creating-identities) and can be obtained by:

```shell
$ nunet key did <user>
```

or if you are using a Ledger Wallet

```shell
$ nunet key did ledger:<user>
```

##### Setup your DMS for the public testnet

0. **The NuNet DID**

```
did:key:zzCHUybNYmK8QsttZwXqUX8aDLoBGHnMCakDX2RpsGwmXmYHEW
```

1. **Create a capability anchor for public behaviors**

Create the grant

```shell
$ nunet cap grant --context user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <nunet-did>
```

or if you are using a Ledger Wallet

```shell
$ nunet cap grant --context ledger:user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <nunet-did>
```

And the granted token as a require anchor

```shell
$ nunet cap anchor --context dms --require <the-grant-output>
```

The first command grants nunet authorized users the capability to invoke public behaviors until December 31, 2024, and outputs a token.

The second command consumes the token and adds the require anchor for your DMS

2. **Ask NuNet for a public network capability token**

To request tokens for participating in the testnet, please go to [did.nunet.io](https://did.nunet.io) and submit the did you generated along with your gitlab username and an email address to receive the token. It's highly recommended that you use a Ledger hardware wallet for your keys.

3. **Use the NuNet granted token to authorize public behavior invocations in the public network**

3.1 **Add the provide anchor to your personal context**

```shell
$ nunet cap anchor --context user --provide <the-token-you-got-from-nunet>
```

or if you are using a Ledger Wallet

```shell
$ nunet cap anchor --context ledger:user --provide <the-token-you-got-from-nunet>
```

3.2 **Delegate to your DMS**

```shell
$ nunet cap delegate --context user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <your-dms-did>
```

or if you are using a Ledger Wallet

```shell
$ nunet cap delegate --context ledger:user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <your-dms-did>

```

3.3 **Add the delegation token as a provide anchor in your DMS**

```shell
$ nunet cap anchor --context dms --provide <the-delegate-output>
```

The first command ingests the NuNet provided token and the last two commands use this token to delegate the public behavior capabilities to your DMS.

#### Running DMS

If everything was setup properly, you should be able to run:

```shell
$ nunet run
```

> **Darwin users**: If you plan to onboard your computer power to the network, You may need to run with `sudo`.
> See the [optional features and permissions](#permissions-and-features) section for more information.

By default, DMS runs on port 9999.

### Provide Compute Power to the Network

If you want to contribute your computer's resources (CPU, RAM, GPU, storage) to the network, you have to onboard your machine.

Follow our [Compute Provider Guide](docs/onboarding.md) to get started.

### Deploy Jobs on the Network

Every node on the network can deploy workloads across available compute resources, given the necessary capabilities. Learn how deployments work by following our [Deployments Guide](docs/deployments.md).

### REST Endpoints

Refer to the `api` package [README](https://gitlab.com/nunet/device-management-service/-/blob/main/api/README.md) for the list of all endpoints. Head over to project's issue section and create an issue with your question.

## Configuration

### Config file

The DMS searches for a configuration file `dms_config.json` in the following locations, in order of priority whenever it's started:

1. The current directory (`.`)
2. `$HOME/.nunet`
3. `/etc/nunet`

The configuration file must be in JSON format and it does **not** support comments. It's recommended that only the parameters that need to be changed are included in the config file so that other parameters can retain their default values.

It's possible to manage configuration using the `config` subcommand as well. `nunet config set` allows setting each parameter individually and `nunet config edit` will open the config file in the default editor from `$EDITOR`

#### Run Two DMS Instances Side by Side

As a developer, you might find yourself needing to run two DMS instances, one acting as an SP (Service Provider) and the other as a CP (Compute Provider).

**Step 1**:

Clone the repository to two different directories. You might want to use descriptive directory names to avoid confusion.

**Step 2**:

You need to modify some configurations so that both DMS instances do not end up trying to listen on the same port and use the same path for storage. For example, ports on `p2p.listen_address`, `rest.port`, `general.user_dir` etc... neeed to be different for two instances on the same host.

The `dms_config.json` file can be used to modify these settings. Here is a sample config file that can be modified to your preference:

```json
{
  "p2p": {
    "listen_address": ["/ip4/0.0.0.0/tcp/9100", "/ip4/0.0.0.0/udp/9100/quic-v1"]
  },
  "general": {
    "user_dir": "/home/user/.config/nunet/dms/",
    "debug": true
  },
  "rest": {
    "port": 10000
  }
}
```

Prefer to use absolute paths and have a look at the [config structure](https://gitlab.com/nunet/device-management-service/-/blob/main/internal/config/config.go) for more info.

## Tests

Some packages contain tests, and it is always best to run them to ensure there are no broken tests before submitting any changes. Before running the tests, the Firecracker executor requires some test data, such as a kernel file, which can be downloaded with:

```bash
make testdata
```

After the download is complete, all unit tests can be run with the following command. It's necessary to include the `unit` tag due to the existence of files that contain functional and integration tests.

```bash
go test --tags unit ./...
```

Help in contributing tests is always appreciated :)

## Specification

### Description

NuNet is a computing platform that provides globally distributed and optimized computing power and storage for decentralized networks, by connecting data owners and computing resources with computational processes in demand of these resources. NuNet provides a layer of intelligent interoperability between computational processes and physical computing infrastructures in an ecosystem which intelligently harnesses latent computing resources of the community into the global network of computations.

Detailed information about the NuNet platform, concepts, architecture, models, stakeholders can be found in these two papers:

- [White Paper](https://docs.nunet.io/nunet-whitepaper)
- [Yellow Paper](https://docs.nunet.io/docs/nunet-yellow-paper)

DMS (Device Management Service) acts as the foundation of the NuNet platform, orchestrating the complex interactions between various computing resources and users. DMS implementation is structured into packages, creating a more maintainable, scalable, and robust codebase that is easier to understand, test, and collaborate on. Here are the existing packages in DMS and their purposes:

- **`actor`**: Contains the NuActor framework for secure actor oriented programming in decentralized systems.
- **`dms`**: Responsible for starting the whole application and core DMS functionality such as onboarding, job orchestration, job and resource management, etc.
- **`internal`**: Code that will not be imported by any other packages and is used only on the running instance of DMS. This includes all configuration-related code, background tasks, etc.
- **`db`**: Database used by the DMS.
- **`storage`**: Disk storage management on each DMS for data related to DMS and jobs deployed by DMS. It also acts as an adapter to external storage services.
- **`api`**: All API functionality to interact with the DMS.
- **`cmd`**: Command line functionality and tools.
- **`network`**: All network-related code such as p2p communication, IP over Libp2p, and other networks that might be needed in the future.
- **`executor`**: Responsible for executing the jobs received by the DMS. Interface to various executors such as Docker, Firecracker, etc.
- **`observability`**: Logs, traces, and everything related to observability.
- **`plugins`**: Defined entry points and specs for third-party plugins, registration, and execution of plugin code.
- **`types`**: Defines data structures and interfaces that are used across the whole DMS component by different packages.
- **`utils`**: Utility tools and functionalities used by other packages.
- **`lib`**: External libs being used in DMS.
- **`tokenomics`**: Interaction with blockchain for the crypto-micropayments layer of the platform (not yet implemented).
- **`test`**: Contains some automated tests, not including unit tests.
- **`maint-scripts`**: Utility scripts for building / development assistance and runtime.
- **`examples`**: Examples of ensembles to be used to deploy jobs on NuNet platform.
- **`docs`**: Documentation about main functionalities in DMS as onboarding, deployments, how to create a restricted network.
- **`specs`**: Platform components specifications.

### Design and Architecture

Main concepts of the architecture of DMS, the main component of the NuNet platform, can be found in the [Yellow Paper](https://gitlab.com/nunet/publisher/platform-yellow-paper/-/tree/main).

### Functionality

Current key functional areas of DMS:
* Actor-based system: NuNet's network communication is powered by the [NuActor System](https://gitlab.com/nunet/device-management-service/-/blob/main/actor/README.md), a zero-trust system that utilizes fine-grained capabilities, anchored on [DIDs](https://www.w3.org/TR/did-core/), following the [UCAN model](https://github.com/ucan-wg/).
* Node management: Supports [onboarding/offboarding](https://gitlab.com/nunet/device-management-service/-/blob/main/docs/onboarding.md) of nodes and manages peer connections.
* Compute ensembles: Defines ensembles as collections of logical nodes and allocations that represent compute workloads (as explained [here](https://gitlab.com/nunet/device-management-service/-/blob/deployment-docs/dms/jobs/README.md)). Each allocation is a compute job assigned to a node.
* Orchestration: [Deploys an ensemble](https://gitlab.com/nunet/device-management-service/-/blob/main/docs/deployments.md) across nodes by fulfilling the specified constraints. This is done using a constraint satisfaction process where bids are requested from nodes and evaluated based on the required resources and locations.
* Supervision: Once deployed, ensembles are continuously monitored.
* VM/container lifecycle management: Allows creation, customization, and management of containers and virtual machines on the network.
* Resource management: Controls different types of compute resources (VMs, CPUs, GPUs).
* API and CLI support: Offers both an [API](https://gitlab.com/nunet/device-management-service/-/blob/deployment-docs/api/README.md) and [CLI](https://gitlab.com/nunet/device-management-service/-/blob/deployment-docs/cmd/actor/README.md) for programmatic and manual interaction with the system.
* Observability: Collects information of events happening in the network allowing to perform real-time or post-mortem analysis and visualizations.

### Data Types

The global class diagram for the DMS is shown below.
[Global Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/specs/class_diagram.puml)

Find additional data models within specific packages.

### References

In addition to the relevant links added in the sections above, you can also find useful links here: [NuNet Links](https://www.nunet.io/links).


