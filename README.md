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
    - [System Requirements](#system-requirements)
      - [CPU-only machines](#cpu-only-machines)
        - [Minimum System Requirements](#minimum-system-requirements)
        - [Recommended System Requirements](#recommended-system-requirements)
      - [GPU Machines](#gpu-machines)
        - [Minimum System Requirements](#minimum-system-requirements-1)
        - [Recommended System Requirements](#recommended-system-requirements-1)
  - [Usage](#usage)
    - [Quick Start](#quick-start)
      - [Creating identities](#creating-identities)
      - [Using a Ledger Wallet](#using-a-ledger-wallet)
      - [Setting up Capabilities](#setting-up-capabilities)
        - [Create capability contexts](#create-capability-contexts)
          - [Add a root anchor for your DMS context](#add-a-root-anchor-for-your-dms-context)
        - [Setup your DMS for the public testnet](#setup-your-dms-for-the-public-testnet)
      - [Running DMS](#running-dms)
    - [Onboarding](#onboarding)
    - [REST Endpoints](#rest-endpoints)
  - [Configuration](#configuration)
    - [Config file](#config-file)
      - [Run Two DMS Instances Side by Side](#run-two-dms-instances-side-by-side)
  - [Tests](#tests)
  - [Specification](#specification)
    - [Description](#description)
    - [Design and Architecture](#design-and-architecture)
      - [Conceptual Basis](#conceptual-basis)
      - [Ontology](#ontology)
      - [Architecture](#architecture)
      - [Research](#research)
    - [Functionality](#functionality)
    - [Data Types](#data-types)
    - [References](#references)
    - [Class Diagram](#class-diagram)
      - [Source File](#source-file)

## About

**Device Management Service** or **DMS** enables a machine to join the decentralized NuNet network both as a compute provider, offering its resources to the network, or to leverage the compute power of other machines in the network for processing tasks. Users with available hardware resources can get compensated whenever their machine is utilized for a computational job by other users in the network. The ultimate goal is to create a decentralized compute economy that is able sustains itself.

### Payment

All transactions on the Nunet network are expected to be conducted using the platform's utility token [NTX](https://docs.nunet.io/docs/v/getting-ntx). However, DMS is currently in development, and payment isn't part of `v0.5.0-boot` release. NTX payments are expected to be implemented in the [Public Alpha Mainnet](https://gitlab.com/groups/nunet/-/milestones/46#tab-issues) milestone within later release cycles.

**Note**: If you are a developer, please check out the [DMS specifications](#specification) and [Building from Source](#building-from-source) sections of this document.

## Installation

You can install Device Management Service (DMS) via [binary releases](#binary-releases) or [building it from source](#building-from-source).

### Binary releases

You can find all binary releases [here](https://gitlab.com/nunet/device-management-service/-/releases) and other builds in-between releases in the [package registry](https://gitlab.com/nunet/device-management-service/-/packages)

#### Ubuntu/Debian

1. Download the latest .deb pacakge from the package registry:
2. Install the debian package with `apt` or `dpkg`:

```
sudo apt update
sudo apt install ./nunet-dms_0.5.0_amd64.deb -y
```
3. Some dependencies such as docker and libsystemd-dev might be missing so it's recommended to fix install by running:
```
sudo apt -f install
```

### Building from source

We currently support Linux and MacOS (Darwin).

#### Dependencies

- iproute2 (linux only)
- libsystemd-dev (linux only)
- go (v1.21 or later)

Clone the repository:

```
git clone https://gitlab.com/nunet/device-management-service.git
```

Build the CLI:

```bash
cd device-management-service
make
```

This will result in a binary file in builds/ folder named as `dms_linux_amd64` or `dms_darwin_arm64` depending on the platform.

You can add the compiled binary to a directory in your `$PATH`. See the [Usage](#usage) section for more information.

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
- NVIDIA GPU: 4 GB VRAM
- Free Disk Space: 50 GB
- Internet Download/Upload Speed: 50 Mbps

##### Recommended System Requirements

- CPU: 4 GHz
- RAM: 16-32 GB
- NVIDIA GPU: 8-12 GB VRAM
- Free Disk Space: 100 GB
- Internet Download/Upload Speed: 100 Mbps

## Usage

### Quick Start

This quick start guide will walk you through the process of setting up a Device Management Service (DMS) instance for the first time and getting it running. We'll cover creating identities, setting up capabilities, and running the DMS.

#### Creating identities

The first step is to generate identities/keys and capability contexts. It is recommended that two keys are setup: one for the user (default name `user`) and another for the dms (default name `dms`)

A capability context is created with the `dms cap new <context>` command and it is anchored on a key with the context name.

To set up a new identity/create a new key, run the command:

```
$ nunet key new <identity>
```

Then, to initialize its capability context:

```
$ nunet cap new <identity>
```

In this example, we are going to set up two identities:

```
$ nunet key new user // returns did
$ nunet cap new user

$ nunet key new dms // returns did
$ nunet cap new dms
```

If you use a ledger wallet for your personal key, you can create the user context as follows:
```
$ nunet key did ledger
$ nunet cap new ledger:user
```

You can create as many identities as you want, specially if you want
to manage multiple DMS instances.

The `key new` command returns a DID key for the specified identity.
Remember to secure your keys and capability contexts, as they control access to your NuNet resources.
They are encrypted and stored under `$HOME/.nunet` by default.

Each time a new identity is generated it will prompt the user
for a passphrase. The passphrase is associated with the created
identity, thus a different passphrase can be set up for each identity.
If you prefer, it's possible to set a `DMS_PASSPHRASE` environment variable
to avoid the command prompt.

#### Using a Ledger Wallet

It is also possible to use a Ledger Wallet instead of creating a new
key; this is recommended for user contexts, but you should not use it
for the dms context as it needs the key to sign capability tokens.

To set up a user context with a Ledger Wallet, you need the
`ledger-cli` script from [NuNet's ledger wallet tool](https://gitlab.com/nunet/ledger-wallet).
The tool uses the Eth application and specifically the first Eth
account with signing of personal messages. Everything that needs to be
signed (namely capability tokens) will be presented on your Nano's
screen in plaintext so that you can inspect it.

You can get your Ledger wallet's DID with:
```
$ nunet key did ledger
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
```
$ nunet cap anchor --context dms --root <user-did>
```

Where `<user-did>` is the user did created above in [Creating identities](#creating-identities) and can be obtained by:
```
$ nunet key did <user>

## or if you are using a Ledger Wallet:
$ nunet key did ledger
```

##### Setup your DMS for the public testnet
0. **The NuNet DID**

```
did:key:zzCHUybNYmK8QsttZwXqUX8aDLoBGHnMCakDX2RpsGwmXmYHEW
```

1. **Create a capability anchor for public behaviors**

```
# Create the grant
$ nunet cap grant --context user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <nunet-dir>

# or if you are using a Ledger Wallet
$ nunet cap grant --context ledger:user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <nunet-dir>

# And the granted token as a require anchor
$ nunet cap anchor --context dms --require <the-grant-output>

```

The first command grants nunet authorized users the capability to invoke public behaviors until December 31, 2024, and outputs a token.

The second command consumes the token and adds the require anchor for your DMS


2. **Ask NuNet for a public network capability token**

TODO

3. **Use the NuNet granted token to authorize public behavior invocations in the public network**

```
# Add the provide anchor to your personal context
$ nunet cap anchor --context user --provide <the-token-you-got-from-nunet>

# or if you are using a Ledger Wallet
$ nunet cap anchor --context ledger:user --provide <the-token-you-got-from-nunet>

# Delegate to your DMS
$ nunet cap delegate --context user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <your-dms-dir>

# or if you are using a Ledger Wallet
$ nunet cap delegate --context ledger:user --cap /public --cap /broadcast --topic /nunet --expiry 2024-12-31 <your-dms-dir>

$ nunet cap anchor --context dms <the-delegate-output>
```

The first command ingests the NuNet provided token and the last two commands use this token to delegate the public behavior capabilities to your DMS.

#### Running DMS

If everything was setup properly, you should be able to run:

```
$ nunet run
```

By default, DMS runs on port 9999.

### Onboarding

You don't necessarily need to onboard for development, but that depends on which part you're working on. To onboard during development, `/etc/nunet` needs to be manually created since it is created with the package during installation.

Refer to `dms/onboarding` package [README](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/README.md) for details of onboarding functionality for compute provider users.

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
- [Yellow Paper](https://gitlab.com/nunet/publisher/platform-yellow-paper/-/tree/main)

DMS (Device Management Service) acts as the foundation of the NuNet platform, orchestrating the complex interactions between various computing resources and users. DMS implementation is structured into packages, creating a more maintainable, scalable, and robust codebase that is easier to understand, test, and collaborate on. Here are the existing packages in DMS and their purposes:

- **`dms`**: Responsible for starting the whole application and core DMS functionality such as onboarding, job orchestration, job and resource management, etc.
- **`internal`**: Code that will not be imported by any other packages and is used only on the running instance of DMS. This includes all configuration-related code, background tasks, etc.
- **`db`**: Database used by the DMS.
- **`storage`**: Disk storage management on each DMS for data related to DMS and jobs deployed by DMS. It also acts as an adapter to external storage services.
- **`api`**: All API functionality (including REST API, etc.) to interact with the DMS.
- **`cmd`**: Command line functionality and tools.
- **`network`**: All network-related code such as p2p communication, IP over Libp2p, and other networks that might be needed in the future.
- **`executor`**: Responsible for executing the jobs received by the DMS. Interface to various executors such as Docker, Firecracker, etc.
- **`telemetry`**: Logs, traces, and everything related to telemetry.
- **`plugins`**: Defined entry points and specs for third-party plugins, registration, and execution of plugin code.
- **`types`**: Contains data models imported by various packages.
- **`utils`**: Utility tools and functionalities.
- **`tokenomics`**: Interaction with blockchain for the crypto-micropayments layer of the platform.

### Design and Architecture

#### Conceptual Basis

Main concepts of the architecture of DMS, the main component of the NuNet platform, can be found in the [Yellow Paper](https://gitlab.com/nunet/publisher/platform-yellow-paper/-/tree/main).

#### Ontology

The Nunet Ontology, which forms the basis of the design, is explained in the articles below:

- [NuNet Job Orchestration I: Ontology and Nomenclature](https://nunet.gitlab.io/research/blog/posts/ontology-and-nomenclature/)
- [NuNet Job Orchestration II: Scheduling](https://nunet.gitlab.io/research/blog/posts/scheduling-and-orchestration/)
- [NuNet Job Orchestration III: Mapping Ontology to Scheduling](https://nunet.gitlab.io/research/blog/posts/taxonomy-of-job-scheduling/)

#### Architecture

Refer to the following items to understand **DMS architecture** at a high level.

- [DMS Architecture -- Understanding I](https://nunet.gitlab.io/research/blog/posts/dms-architecture/)
- [Entity Diagram - DMS High Level](https://gitlab.com/nunet/device-management-service/-/blob/main/specs/entityDiagrams/New_DMS_Structure_Highlevel.drawio.svg)

#### Research

Relevant research work that has informed the design of DMS can be found below:

- [Detailed Job Orchestration Sequences I](https://nunet.gitlab.io/research/blog/posts/job-orchestration-details/)
- [Detailed Job Orchestration Sequences II](https://nunet.gitlab.io/research/blog/posts/orchestration-discussion/)
- [Gossipsub, DHT, and Push/Pull Mechanisms](https://nunet.gitlab.io/research/blog/posts/gossipsub/)
- [Parent-Child Hierarchy, Allocations, and Failure Tolerance](https://nunet.gitlab.io/research/blog/posts/parent-child-relations/)
- [Kubernetes Integration Specs](https://nunet.gitlab.io/research/blog/posts/kubernetes-integration/)

### Functionality

DMS is currently being refactored and new functionality will be added.

### Data Types

Refer to the DMS global class diagram in [this](#class-diagram) section and various packages for data models.


### References

In addition to the relevant links added in the sections above, you can also find useful links here: [NuNet Links](https://www.nunet.io/links).

### Class Diagram

The global class diagram for the DMS is shown below.

#### Source File

[Global Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/specs/class_diagram.puml)
