# Device Management Service (DMS)

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

<!--toc:start-->
- [Device Management Service (DMS)](#device-management-service-dms)
  - [Table of Contents](#table-of-contents)
  - [About](#about)
    - [Payment](#payment)
  - [Installation](#installation)
    - [Binary releases](#binary-releases)
      - [Ubuntu/Debian](#ubuntudebian)
    - [Building from source](#building-from-source)
      - [Dependencies](#dependencies)
        - [Linux only](#linux-only)
        - [macOS (Apple Silicon - M1/M2) only](#macos-apple-silicon-m1m2-only)
      - [MacOS (ARM64 architecture)](#macos-arm64-architecture)
      - [Linux Installation](#linux-installation)
    - [Permissions and features (for compute providers using Linux)](#permissions-and-features-for-compute-providers-using-linux)
      - [Required: Net-admin permission and IP over libp2p](#required-net-admin-permission-and-ip-over-libp2p)
      - [Optional: containerd executor (Linux)](#optional-containerd-executor-linux)
      - [May be required: iptables upgrade](#may-be-required-iptables-upgrade)
      - [macOS ARM64 Installation & Debugging Guide (Apple Silicon)](#macos-arm64-installation--debugging-guide-apple-silicon)
        - [Troubleshooting Build Issues](#troubleshooting-build-issues)
          - [Output should include: Mach-O 64-bit executable arm64](#output-should-include-mach-o-64-bit-executable-arm64)
          - [MacOS Limitations](#macos-limitations)
          - [Optional: Add Binary to PATH](#optional-add-binary-to-path)
    - [Installation on VMs](#installation-on-vms)
    - [Installation on WSL](#installation-on-wsl)
    - [System Requirements](#system-requirements)
      - [CPU-only machines](#cpu-only-machines)
        - [Minimum System Requirements](#minimum-system-requirements)
        - [Recommended System Requirements](#recommended-system-requirements)
      - [GPU Machines](#gpu-machines)
        - [Minimum System Requirements](#minimum-system-requirements-1)
        - [Recommended System Requirements](#recommended-system-requirements-1)
    - [GPU Driver Installation](#gpu-driver-installation)
      - [For AMD64 Platforms](#for-amd64-platforms)
      - [NVIDIA GPUs](#nvidia-gpus)
      - [AMD GPUs](#amd-gpus)
      - [Intel Discrete GPUs](#intel-discrete-gpus)
  - [Usage](#usage)
    - [Quick Start](#quick-start)
      - [Creating identities](#creating-identities)
        - [Using a Ledger Wallet](#using-a-ledger-wallet)
      - [Setting up Capabilities](#setting-up-capabilities)
        - [Add a root anchor for your DMS context](#add-a-root-anchor-for-your-dms-context)
        - [Setup your DMS for the public testnet](#setup-your-dms-for-the-public-testnet)
      - [Running DMS](#running-dms)
    - [Provide Compute Resources to the Network](#provide-compute-resources-to-the-network)
    - [Deploy Jobs on the Network](#deploy-jobs-on-the-network)
    - [Contracts](#contracts)
    - [REST Endpoints](#rest-endpoints)
  - [Configuration](#configuration)
    - [Config file](#config-file)
  - [Tests](#tests)
    - [e2e Tests](#e2e-tests)
      - [Prerequisites](#prerequisites)
      - [Running the Tests](#running-the-tests)
  - [Specification](#specification)
    - [Description](#description)
    - [Design and Architecture](#design-and-architecture)
    - [Functionality](#functionality)
    - [References](#references)
<!--toc:end-->

## About

**Device Management Service** or **DMS** enables a machine to join the decentralized NuNet network both as a compute provider, offering its resources to the network, or to leverage the compute power of other machines in the network for running computational workloads. Eventually users with available hardware resources will get compensated whenever their machine is utilized for a computational job by other users in the network. The ultimate aim is to create a decentralized compute economy that is able to sustain itself.

### Payment

All transactions on the Nunet network are expected to be conducted using the native utility token [NTX](https://docs.nunet.io/docs/v/getting-ntx). However peer to peer payments is not part of current release. NTX payments are expected to be implemented in the [Public Network with Tokenomics](https://docs.nunet.io/docs/project-management-portal/platform-milestones/public-network-with-tokenomics) milestone within later release cycles.

**Note**: If you are a developer, please check out the [DMS specifications](#specification) and [Building from Source](#building-from-source) sections of this document.

## Installation

You can install Device Management Service (DMS) via [binary releases](#binary-releases) or [building it from source](#building-from-source).

### Binary releases

You can find all binary releases [here](https://gitlab.com/nunet/device-management-service/-/releases) and other builds in-between releases in the [package registry](https://gitlab.com/nunet/device-management-service/-/packages).
We currently support ARM and AMD64 architectures. You may check your architecture with appropriate command (`uname -p` for linux) and refer to the architecture name mapping e.g. [here](https://itsfoss.com/arm-aarch64-x86_64/) for figuring out the correct package to download.

**Note**: If you installed the binary from a release and you would like to act as compute provider, you may need to check [permissions and features](#permissions-and-features-for-compute-providers-using-linux) to enable some _required_ and optional features.

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

##### Common

- go (v1.22.7 or later)
- [git lfs](https://git-lfs.com/) for downloading large files

##### Linux only

- iproute2 (linux only)
- gcc
- build-essential (linux only)

##### macOS (Apple Silicon - M1/M2) only

- Install [Homebrew](https://brew.sh/) for package management
- Recommended: [iTerm2](https://iterm2.com/) for improved CLI experience.

#### MacOS (ARM64 architecture)

Before you begin, ensure that you have the following installed:

1. Homebrew: to manage dependencies easily.

    If you don't have Homebrew installed, run:

    ```bash
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    ```

2. Go (Golang): The Go programming language, which is used to build the project.

    Verify if Go is installed:

    ```bash
    go version
    ```

     Install if needed

    ```bash
    brew install go
    ```

3. Git: To clone the GitLab repository.
4. Make: Used for automating the build process.

       Verify if make is installed

    ```bash
    make --version
    ```

Install if needed

    ```bash
    xcode-select --install
    ```

#### Linux Installation

Install dependencies:

```shell
sudo apt update && sudo apt install -y iproute2 build-essential libsystemd-dev gcc-arm-linux-gnueabihf gcc-aarch64-linux-gnu
```

Clone the repository:

```shell
git clone https://gitlab.com/nunet/device-management-service.git
cd device-management-service
```

Configure git-lfs:

```shell
git lfs install && \
git lfs fetch && \
git lfs pull
```

Build the CLI:

```bash
make
```

This will produce a binary in `builds/` named `dms_linux_amd64`.

**Note**: If you built from source and would like to act as a compute provider,
you may need to check [permissions and features](#permissions-and-features-for-compute-providers-using-linux) to enable some _required_ and optional features.

To cross compile to arm, cross compilers need to be installed. In particular arm-linux-gnueabihf and aarch64-linux-gnu.
For debian systems, install with:

```shell
apt install gcc-arm-linux-gnueabihf gcc-aarch64-linux-gnu
```

You can add the compiled binary to a directory in your `$PATH`. See the [Usage](#usage) section for more information.

### Permissions and features (for compute providers using Linux)

The following applies _only_ for **compute providers using Linux**. If you're running a client/orchestrator, you do _not_ need to set any additional permissions except if you specify for the orchestrator to join the subnet in the ensemble config. If the ensemble contains the following config

```yml
subnet:
  join: true
```

then the orchestator will create its own tun interface and join the subnet. For that the `cap_net_admin` permission is required.

> **Darwin users**: unfortunately, the DMS can neither work with granular permissions nor
> with iptables and tun interfaces. Thus, on Darwin, for now, the DMS can only be an orchestrator.

For Linux users, granular permissions will have to be set to the binary (possible but _NOT_ recommended way is to run as root).

#### Required: Net-admin permission and IP over libp2p

> **Note**: step **not** needed if you're using our **debian package**.
>
> It's needed for those building from source or downloading the binary releases.

**Note**: `cap_net_admin` and `cap_sys_admin` are **required** capabilities for **compute providers**. `cap_net_admin` would be required for orchestrators if they are joining the subnet.

Setting the `cap_net_admin` permission enables IP over libp2p which is a feature that enhances the capabilities of compute providers, allowing them to participate in a wider
range of jobs. One capability enabled with this feature is to do port forwarding which it won't be possible without setting the right unix permissions.

Setting the `cap_sys_admin` permission allows the DMS to perform a mount for storage functionality. However, due to `cap_sys_admin` being too wide a permission, please make sure the machine you're setting up on can handle security risk until `cap_sys_admin` is narrowed down to specific actions or there is a better alternative.

To set the necessary capabilities, run the following command:

```shell
sudo setcap cap_net_admin,cap_sys_admin+ep /usr/bin/nunet
```

The above command depends on: `libcap2-bin` (Debian/Ubuntu) or `libcap` (CentOS/RHEL/Arch...)

#### Optional: containerd executor (Linux)

The **containerd** executor is an alternative to Docker for running allocations on Linux compute nodes. It requires containerd, CNI plugins, and the same host networking permissions as above. **Run DMS as root** on compute provider DMSs that intend to use it.

Minimal checklist:

1. Install **containerd** and **containerd-shim-runc-v2**; ensure `/run/containerd/containerd.sock` exists.
2. Install CNI plugins (`bridge`, `host-local`, `portmap`, `firewall`) into **`/opt/cni/bin`**.
3. Install the CNI conflist into **`/etc/cni/net.d`** (see `maint-scripts/nunet-dms/etc/cni/net.d/`).
4. Create **`/var/run/netns`** and enable **`net.ipv4.ip_forward=1`**.
5. Set allocation `execution.type` to **`containerd`** in your ensemble.

Full setup steps: [executor/containerd/README.md](./executor/containerd/README.md).

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

#### macOS ARM64 Installation & Debugging Guide (Apple Silicon)

Check your macOS version (important for compatibility):

```shell
sw_vers
```

Clone the repository:

```shell
git clone https://gitlab.com/nunet/device-management-service.git
cd device-management-service
```

Configure Git LFS:

```shell
git lfs install && \
git lfs fetch && \
git lfs pull
```

Build the CLI:

```shell
make
```

This will produce a binary in `builds/` named `dms_darwin_arm64`.

##### Troubleshooting Build Issues

Ensure Go is installed via Homebrew:

```shell
brew install go
```

If you see errors like:

```bash
zsh: permission denied: ./dms
```

Fix with:

```shell
chmod +x builds/dms_darwin_arm64
```

If macOS blocks execution:

```shell
sudo xattr -d com.apple.quarantine builds/dms_darwin_arm64
```

Confirm binary format:

```shell
file builds/dms_darwin_arm64
```

###### Output should include: Mach-O 64-bit executable arm64

Run the CLI:

```shell
./builds/dms_darwin_arm64
```

You should see:

```scss
The Device Management Service (DMS) Command Line Interface (CLI)
Usage:
  nunet [flags]
  nunet [command]
...
```

###### MacOS Limitations

- `cap_net_admin` and `cap_sys_admin` are not supported on macOS.
- DMS cannot act as a compute provider — only as an orchestrator.
- IP over libp2p, port forwarding, and TAP interfaces are unsupported.

###### Optional: Add Binary to PATH

You may copy the binary into a directory in your $PATH:

```shell
cp builds/dms_darwin_arm64 /usr/local/bin/nunet
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
- Internet Download/Upload Speed: 4 Mbps / 4 Mbps
If the above CPU has 4 cores, your available CPU would be around 8000 MHz. So if you want to onboard half your CPU and RAM on NuNet, you can specify 4000 MHz CPU and 2000 MB RAM.

##### Recommended System Requirements

- CPU: 3.5 GHz
- RAM: 8-16 GB
- Free Disk Space: 20 GB
- Internet Download/Upload Speed: 10 Mbps / 10 Mbps

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

#### For AMD64 Platforms

We recommend using the [Ubuntu](https://ubuntu.com/)-based [HiveOS](https://hiveon.com/install/) for the easiest setup.

If you prefer to use a different operating system or need to install drivers manually, please follow these steps:

#### NVIDIA GPUs

1. Visit the [NVIDIA Official Driver Downloads](https://www.nvidia.com/en-us/drivers/) page.
2. Select your GPU model and operating system.
3. Download and install the recommended driver.
4. Install the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html).
5. Reboot your system after installation.

#### AMD GPUs

1. Visit the [AMD Drivers and Support for Processors and Graphics](https://www.amd.com/en/support/download/drivers.html) page.
2. Select your GPU model and operating system.
3. Download and install the recommended driver.
4. Reboot your system after installation.

Along with the drivers, you will need to install amdgpu using ROCm for AMD GPUs. You can find the installation instructions [here](https://rocm.docs.amd.com/projects/install-on-linux/en/latest/install/amdgpu-install.html).

Make sure you select the rocm usecase when installing the amdgpu.

```bash
sudo amdgpu-install --usecase=rocm
```

#### Intel Discrete GPUs

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
nunet cap new user
```

then the dms instance.

```shell
nunet cap new dms
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

For a quick start, please take a look at the script [maint-scripts/quickstart.sh](./maint-scripts/quickstart.sh).

##### Using a Ledger Wallet

It is also possible to use a Ledger Wallet instead of creating a new
key; this is recommended for _user_ contexts, but you **should not** use it
for the _dms_ context as it needs the key to sign capability tokens.

To set up a _user_ context with a Ledger Wallet, you need the
`ledger-cli` script from [NuNet's ledger wallet tool](https://gitlab.com/nunet/dev-tools/ledger-wallet).
The tool uses the Eth application and specifically the first Eth
account with signing of personal messages. Everything that needs to be
signed (namely capability tokens) will be presented on your Nano's
screen in plaintext so that you can inspect it.

You can get your Ledger wallet's DID with:

```shell
nunet key did ledger
```

To create the capability context for the user

```shell
nunet cap new ledger:user
```

#### Setting up Capabilities

NuNet's network communication is powered by the [NuActor System](actor/README.md), a zero-trust system that utilizes fine-grained capabilities, anchored on [DIDs](https://www.w3.org/TR/did-core/), following the [UCAN model](https://github.com/ucan-wg/).

Once both identities are created, you'll need to set up capabilities. Specifically:

1. Create capability contexts for both the user and each of your DMS instances.
2. Add the user's DID as a _root anchor_ for the DMS capability context (see [here](#add-a-root-anchor-for-your-dms-context)). This ensures that the DMS instance fully trusts the user, granting complete control over the DMS (the root capability).
3. If you want your DMS to participate in the public NuNet testnet (and eventually the mainnet), you'll need to set up capability anchors for your DMS:
   1. Create a capability anchor to allow your DMS to accept _public behavior invocations_
      from authorized users and DMSs in the NuNet ecosystem.
   2. Add this token to your DMS as a _require anchor_.
   3. Request a capability token from NuNet to invoke public behaviors on the network.
   4. Add the token as a _provide anchor_ in your personal capability context.
   5. Delegate to your DMS the ability to make public invocations using your token.
   6. Add the delegation token as a _provide anchor_ in your DMS.

For more on capabilities and behaviors, see the [DMS Capabilities and Behaviors](dms/behaviors/README.md) document. Alternatively, if you installed the DMS using one of the debian packages, there is a man page with descriptions of the capabilities and behaviors you can access with `man nunet`.

###### Add a root anchor for your DMS context

You can do this by getting the did of the user first with:

```shell
nunet key did user
```

or if you are using a Ledger Wallet

```shell
nunet key did ledger:<user>
```

and then anchoring on the dms context's root with the `nunet cap anchor` command:

```shell
nunet cap anchor --context dms --root <user-did>
```

Alternatively, if you already have the _DMS_PASSPHRASE_ env var set, you can chain the commands together:

```shell
nunet cap anchor --context dms --root $(nunet key did user)
```

The examples below will mostly use this chaining so be sure to have the `DMS_PASSPHRASE` environment variable set to avoid prompts.

##### Setup your DMS for the public testnet

0. **The NuNet DID**

The NuNet public network is represented by the following DID:

```
did:key:zzCHUybNYmK8QsttZwXqUX8aDLoBGHnMCakDX2RpsGwmXmYHEW
```

To make it easier to use this DID with the grant and anchor commands below, you can set it as an environment variable:

```shell
NUNET_DID='did:key:zzCHUybNYmK8QsttZwXqUX8aDLoBGHnMCakDX2RpsGwmXmYHEW'
```

1. **Create a capability anchor to allow public behaviors to be invoked on your device**

Create the grant

```shell
nunet cap grant --context user --cap /public --cap /broadcast --topic /nunet --expiry 2025-12-31 $NUNET_DID | tee /tmp/grant-output
```

or if you are using a Ledger Wallet

```shell
nunet cap grant --context ledger:user --cap /public --cap /broadcast --topic /nunet --expiry 2025-12-31 $NUNET_DID | tee /tmp/grant-output
```

And the granted token as a require anchor

```shell
nunet cap anchor --context dms --require $(cat /tmp/grant-output)
```

The first command grants nunet authorized users the capability to invoke public behaviors until December 31, 2025, and outputs a token.

The second command consumes the token and adds the require anchor for your DMS.

2. **Ask NuNet for a public network capability token**

To request tokens for participating in the testnet, please go to [did.nunet.io](https://did.nunet.io) and submit the did you generated along with your gitlab username and an email address to receive the token. It's highly recommended that you use a Ledger hardware wallet for your keys.

3. **Use the NuNet granted token to get the capability to invoke public behaviors on other machines in the public network**

3.1 **Add the provide anchor to your personal context**

```shell
nunet cap anchor --context user --provide <the-token-you-got-from-nunet>
```

or if you are using a Ledger Wallet

```shell
nunet cap anchor --context ledger:user --provide <the-token-you-got-from-nunet>
```

3.2 **Delegate to your DMS**

```shell
nunet cap delegate --context user --cap /public --cap /broadcast --topic /nunet --expiry 2025-12-31 <your-dms-did>
```

or if you are using a Ledger Wallet

```shell
nunet cap delegate --context ledger:user --cap /public --cap /broadcast --topic /nunet --expiry 2025-12-31 <your-dms-did>

```

3.3 **Add the delegation token as a provide anchor in your DMS**

```shell
nunet cap anchor --context dms --provide <the-delegate-output>
```

The first command ingests the NuNet provided token and the last two commands use this token to delegate the public behavior capabilities to your DMS.

#### Running DMS

If everything was setup properly, you should be able to run:

```shell
nunet run
```

> **Darwin users**: If you plan to onboard your computer resources to the network, You may need to run with `sudo`.
> See the [optional features and permissions](#permissions-and-features-for-compute-providers-using-linux) section for more information.

By default, DMS runs on port 9999.

### Provide Compute Resources to the Network

If you want to contribute your computer's resources (CPU, RAM, GPU, storage) to the network, you have to onboard your machine.

Follow our [Compute Provider Guide](https://gitlab.com/nunet/device-management-service/-/blob/main/docs/onboarding/README.md) to get started.

### Deploy Jobs on the Network

Every node on the network, given the necessary capabilities, can deploy workloads across available compute resources. The first thing to complete before participating
in deployments is to delegate or be delegated deployment capabilities depending on
whether the machine is an orchestrator or compute provider.

Deployments require the `/dms/deployment` capability to invoke behaviors under it such
as the `/dms/deployment/request` behavior which allows orchestrator to request a bid from compute providers. The compute provider too will need to be
granted the capability to submit bids on `/dms/deployment/bid` by the orchestrator. The
latter can be achieved by simply granting the capability (using the `nunet cap grant` command) to specific compute providers that are allowed and anchoring the generated token on the require anchor (using the `nunet cap anchor` command) of the orchestrator without having to send the token. This is mainly because this release intends to provide a fine grained control to orchestrators on who they allow to run their jobs.

Learn how deployments work by following our [Deployments Guide](docs/deployments/README.md).

#### Managing Deployment History

DMS maintains a persistent history of all deployments across restarts. You can manage this deployment history using the following CLI commands:

**List all deployments:**
```shell
nunet actor cmd --context user /dms/node/deployment/list
```

**Prune old deployments:**
Remove deployments before a specified datetime or duration, or remove all deployments with status greater than Running:

```shell
# Remove deployments before a specific datetime (RFC3339 format)
nunet actor cmd --context user /dms/node/deployment/prune --before "2023-01-01T00:00:00Z"

# Remove deployments before a duration (days, hours, minutes, seconds)
nunet actor cmd --context user /dms/node/deployment/prune --before "7d"
nunet actor cmd --context user /dms/node/deployment/prune --before "2h"
nunet actor cmd --context user /dms/node/deployment/prune --before "30m"
nunet actor cmd --context user /dms/node/deployment/prune --before "1s"

# Remove all deployments with status greater than Running (Failed and Completed)
nunet actor cmd --context user /dms/node/deployment/prune --all
```

**Delete a specific deployment:**
Remove a specific deployment by its orchestrator ID:
```shell
nunet actor cmd --context user /dms/node/deployment/delete --orchestrator-id <deployment-id>
```

These commands help you manage storage space and maintain a clean deployment history. The prune command is particularly useful for removing old deployments based on time criteria or removing all deployments with terminal statuses (Failed and Completed) while keeping active deployments (Running). The delete command allows you to remove specific deployments that are no longer needed.

### Contracts

NuNet's contracts and tokenomics architecture integrates Agoric's Electronic Rights Transfer Protocol (ERTP) with an object capability model to enable secure, decentralized economic contracts for compute resource sharing. The system supports both simple bilateral contract management and more complex, multi-party workflows coordinated by a solution enabler, with a focus on capability-based security, deterministic contract logic, and blockchain-agnostic design. Core components include contract objects implemented as nuActors, a contract database for persistent storage, and capability management for secure asset access and delegation. The architecture is designed for extensibility, supporting both current trusted third-party execution and future blockchain-based enhancements.

For more on contracts, please refer to the [tokenomics package](./tokenomics/) and the [README in contracts](./tokenomics/contracts/README.md).


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

## Specification

### Description

NuNet is a protocol that facilitates globally distributed and optimized computing power and storage on a decentralized network, by connecting data owners and computing resources with computational processes in demand of these resources. NuNet provides a layer of intelligent interoperability between computational processes and physical computing infrastructures in an ecosystem which intelligently harnesses latent computing resources of the community into the global network of computations.

Detailed information about the NuNet vision, concepts, architecture, models, stakeholders can be found in these two papers:

- [White Paper](https://docs.nunet.io/nunet-whitepaper)
- [Yellow Paper](https://docs.nunet.io/docs/nunet-yellow-paper/readme/publisher/platform-yellow-paper/main)

DMS (Device Management Service) acts as the foundation of the NuNet ecosystem, orchestrating the complex interactions between various computing resources and users. DMS implementation is structured into packages, creating a more maintainable, scalable, and robust codebase that is easier to understand, test, and collaborate on. Here are the existing packages in DMS and their purposes:

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
- **`docs`**: Documentation about main functionalities in DMS as onboarding, deployments, how to create a private network.
- **`specs`**: Platform components specifications.

### Design and Architecture

Main concepts of the architecture of DMS, the main component of the NuNet ecosystem, can be found in the [Yellow Paper](https://docs.nunet.io/docs/nunet-yellow-paper/readme/publisher/platform-yellow-paper/main).

### Functionality

Current key functional areas of DMS:

- Actor-based system: NuNet's network communication is powered by the [NuActor System](https://gitlab.com/nunet/device-management-service/-/blob/main/actor/README.md), a zero-trust system that utilizes fine-grained capabilities, anchored on [DIDs](https://www.w3.org/TR/did-core/), following the [UCAN model](https://github.com/ucan-wg/).
- Node management: Supports [onboarding/offboarding](https://gitlab.com/nunet/device-management-service/-/blob/main/docs/onboarding/README.md) of nodes and manages peer connections.
- Compute ensembles: Defines ensembles as collections of logical nodes and allocations that represent compute workloads (as explained [here](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/jobs/README.md) and [here](./docs/ensemble/README.md)). Each allocation is a compute job assigned to a node.
- Orchestration: [Deploys an ensemble](https://gitlab.com/nunet/device-management-service/-/blob/main/docs/deployments/README.md) across nodes by fulfilling the specified constraints. This is done using a constraint satisfaction process where bids are requested from nodes and evaluated based on the required resources and locations.
- Supervision: Once deployed, ensembles are continuously monitored.
- VM/container lifecycle management: Allows creation, customization, and management of containers and virtual machines on the network.
- Resource management: Controls different types of compute resources (VMs, CPUs, GPUs).
- API and CLI support: Offers both an [API](https://gitlab.com/nunet/device-management-service/-/blob/main/api/README.md) and [CLI](https://gitlab.com/nunet/device-management-service/-/blob/main/cmd/actor/README.md) for programmatic and manual interaction with the system.
- Observability: Collects information of events happening in the network allowing to perform real-time or post-mortem analysis and visualizations.

### References

In addition to the relevant links added in the sections above, you can also find useful links here: [NuNet Links](https://www.nunet.io/links).
