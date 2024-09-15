# Device Management Service (DMS)

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

[[_TOC_]]

## About

**Device Management Service** or **DMS** is a program that allows users to run various computational workloads on a distributed set of machines. These machines are CPU/GPU-enabled devices that are part of the Nunet network. Think of this as a cloud service, but provided by multiple providers instead of a single entity like Amazon or Google.

For those with available hardware resources, they can earn rewards by onboarding their machines to NuNet. Each time their machine is used to run a computational job, they are eligible to receive compensation for their service.

To summarize, DMS connects users needing computational resources with those who can provide them, enabling the execution of computational work in a distributed setting.

### Payment

All transactions on the Nunet network are expected to be conducted using the platform's utility token [NTX](https://docs.nunet.io/infohub/tokenomics/ntx-utility-token-overview). However, DMS is currently in development, and NTX payments are expected to be implemented in the [Public Alpha Mainnet](https://gitlab.com/groups/nunet/-/milestones/46#tab-issues) milestone.

**Note**: If you are a developer, please check out the [DMS specifications](#specification) and [Getting Started](#getting-started-for-developers) sections of this document.

**Note**: Payment is not yet enabled in the `v0.5.0-boot` release, we will enable it later in the `v0.5.0` release cycle.

## Installation

You can install Device Management Service (DMS) via [binary releases](#binary-releases) or [building it from source](#building-from-source).

### Binary releases

You can find all binary releases [here](https://gitlab.com/nunet/device-management-service/-/releases).

#### Ubuntu

1. Download the latest binary:

```shell
wget https://d.nunet.io/nunet-dms-latest.deb -O nunet-dms-latest.deb
```

2. Install it:

```shell
sudo apt update
sudo apt install ./nunet-dms-latest.deb -y
```

### Building from source

We currently support Linux and MacOS (Darwin).

#### Dependencies

- iproute2 (linux only)
- libsystemd-dev (linux only)
- go (v1.21 or later)

Clone the repository:

```
git clone https://gitlab.com/nunet/device-management-service
```

Build the CLI:

```bash
cd device-management-service
make
```

You can add the compiled binary to a directory in your `$PATH`. See the [Usage](#usage) section for more information.

#### Development Builds

We also provide a `dev-setup` shell script to ease the process of getting started. It does the following:

1. Sets up `pre-commit` hook, which runs tests before every commit.
2. Builds the .deb file and installs it.
3. Stops the `nunet-dms` service so that we can run `main.go` directly.

Run the script:

```bash
bash maint-scripts/dev-setup.sh
```

Once the environment is set up, build the DMS as follows:

```bash
make
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

Though it is possible to run ML jobs on Windows machines with WSL, using Ubuntu 20.04 natively is highly recommended. The reason is that our development is centered around the Linux operating system. Additionally, the system requirements when using WSL would increase by at least around 25%.

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

The first necessary step is to generate identity keys and capability contexts using the command-line.
It's recommended to setup *at least* **two** identities, one for *personal* user (by default the `user` context) and one for use by the *dms* instance (by default the `dms` context)

To set up a new identity, run the command:

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

NuNet's network communications are implemented using the [NuActor System](actor/README.md). NuActor is a zero trust system that makes extensive use of fine grained capabilities anchored on [DIDs](https://www.w3.org/TR/did-core/), following the model of [UCANs](https://github.com/ucan-wg/).

With both identities are created, it's necessary to set up some capabilities.
Specifically:
1. You need to create capability contexts for both the user and each of your DMS instances.
2. The user's DID must be added as a _root anchor_ for the DMS capability context.
   Operationally this means that the DMS instance absolutely trusts the user, and
   confers complete control of the DMS (the root capability).
3. If you want your DMS to be part of the public NuNet testnet (and eventually the mainnet), you will need some capability anchors for your DMS:
  1. Create a capability anchor for your DMS to accept _public behavior invocations_
     from users and DMSs owned by users authorized by NuNet.
  2. Add this token to your DMS as a _require anchor_.
  3. Ask NuNet for a capability token that allows you to invoke public behaviors
     on the network.
  4. Add the token you got as a _provide anchor_ into your personal capability token.
  5. Delegate to your DMS the capability to make public invocations using your token.
  6. Add the delegation token to your DMS as a provide anchor.

Let's make things more concrete.

##### Create capability contexts

A capability context is created with the `dms cap new <context>`
command and it is anchored on a key with the context name.

So, if you have created `user` and `dms` identities, you can create the capability contexts as follows:
```
$ nunet cap new user
$ nunet cap new dms
```

If you use a ledger wallet for your personal key, you can create the user context as follows:
```
$ nunet cap new ledger:user
```

This will create a `user` context anchored on your ledger wallet.

###### Add a root anchor for your DMS context

You can do this by invoking the `dms cap anchor` command:
```
$ nunet cap anchor --context dms --root <user-did>
```

Where the user DID is the did of your personal key. You can get this with this command:
```
$ nunet key did user

## or if you are using a Ledger Wallet
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
The second command consumes the token and adds the require anchor for your DMS>

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

By default, DMS runs on port 9999. DMS looks for a configuration file on the following paths (in order):

1. `.` (current directory)
2. `$HOME/.nunet`
2. `/etc/nunet`

It's possible configure DMS by creating a `dms_config.json` in one of these locations or running `nunet config` to edit the configuration from the command-line.

### Onboarding

You don't necessarily need to onboard for development, but that depends on which part you're working on. To onboard during development, `/etc/nunet` needs to be manually created since it is created with the package during installation.

Refer to `dms/onboarding` package [README](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/README.md) for details of onboarding functionality for compute provider users.

### REST Endpoints

Refer to the `api` package [README](https://gitlab.com/nunet/device-management-service/-/blob/main/api/README.md) for the list of all endpoints. Head over to project's issue section and create an issue with your question.

## Configuration

### Run Two DMS Instances Side by Side

As a developer, you might find yourself needing to run two DMS instances, one acting as an SP (Service Provider) and the other as a CP (Compute Provider).

**Step 1**:

Clone the repository to two different directories. You might want to use descriptive directory names to avoid confusion.

**Step 2**:

You need to modify some configurations so that both DMS instances do not create a deadlock. These configurations include:

1. The port DMS is listening on.
2. The database file DMS uses.

The `dms_config.json` file can be used to modify these settings. Here is a sample config file that can be modified to your preference:

```json
{
  "p2p": {
    "listen_address": ["/ip4/0.0.0.0/tcp/9100", "/ip4/0.0.0.0/udp/9100/quic-v1"]
  },
  "general": {
    "metadata_path": "/home/user/.config/nunet/dms/",
    "debug": true
  },
  "rest": {
    "port": 10000
  }
}
```

Please use absolute paths to avoid trouble. Also, have a look at the [config structure](https://gitlab.com/nunet/device-management-service/-/blob/main/internal/config/config.go).

3. You must also change the port number in the `nunet` shell script if you plan to use the `nunet` CLI.

**Step 3**:

Onboard both DMS instances.

**Step 4**:

Check if both can discover each other.

## Tests

**Running Functional Tests with Python-Behave Framework**

**Pre-requisites**

- **Python Installation:** Ensure Python 3.8 or higher is installed on your system.
- **Environment Variable:** Confirm that the Python path is included in the system's PATH environment variable.

**Steps to Run Functional Tests**

1. **Clone the Test-Suite Repository**

   Open your terminal and execute the following command to clone the repository from GitLab:

   ```bash
   git clone git@gitlab.com:nunet/test-suite.git
   ```

2. **Install the Dependencies**

   Navigate to the cloned test-suite directory:

   ```bash
   cd test-suite
   ```

   Install the necessary dependencies by running:

   ```bash
   pip install -r requirements.txt
   ```

3. **Executing the Tests**

   Navigate to the functional tests directory:

   ```bash
   cd /test-suite/stages/functional_tests
   ```

   Run the tests using the `behave` command:

   ```bash
   behave features/<dir or feature file path>
   ```

   Example:

   ```bash
   behave features/device-management-service/api-tests/p2p_api.feature
   ```

   Note: This command will execute all Gherkin feature files or scenarios under the specified directory. Standard output logs will be displayed in the terminal.

4. **Generate Allure Report (Optional)**

   [Allure](https://allurereport.org/docs/behave/) is a HTML-based reporting framework. If you want to generate the report in a fancy way, follow these steps to generate and view the report:

   **Pre-requisites for Allure**

   - Install Allure from [Allure Releases](https://github.com/allure-framework/allure2/releases).

   **Steps to Generate and View Allure Report**

   - Run the Tests with Allure Formatter:

     ```bash
     behave features/device-management-service/<feature files to run> --junit -f allure_behave.formatter:AllureFormatter -o allure-results
     ```

     Example:

     ```bash
     behave features/device-management-service/api-tests/p2p_api.feature --junit -f allure_behave.formatter:AllureFormatter -o allure-results
     ```

   - Generate the HTML Report:

     ```bash
     allure generate allure-results -o allure-report
     ```

   - View the Report:

     ```bash
     allure open allure-report
     ```

   - Zip the Report for Sharing:

     ```bash
     zip -r allure-report.zip allure-report
     ```

By following these steps, you can effectively run functional tests using the Python-Behave framework and generate comprehensive reports using Allure.

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

### Testing

Some packages contain tests, and it is always best to run them to ensure there are no broken tests before submitting any changes. Before running the tests, the Firecracker executor requires some test data, such as a kernel file, which can be downloaded with:

```bash
make testdata
```

After the download is complete, all unit tests can be run with:

```bash
go test ./...
```

### References

In addition to the relevant links added in the sections above, you can also find useful links here: [NuNet Links](https://www.nunet.io/links).

### Class Diagram

The global class diagram for the DMS is shown below.

#### Source File

[Global Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/specs/class_diagram.puml)
