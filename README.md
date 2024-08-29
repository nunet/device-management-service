![latest release version](https://gitlab.com/nunet/device-management-service/-/badges/main/coverage.svg)

# Device Management Service (DMS)

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

**Usage**
1. [About](#1-about)
2. [Installation](#2-installation)

**Specification**
1. [Description](#1-description)
2. [Design and Architecture](#2-design-and-architecture)
3. [Functionality](#3-functionality)
4. [Data Types](#4-data-types)
5. [Testing](#5-testing)
6. [References](#6-references)
7. [Class Diagram](#7-class-diagram)

**Getting Started for Developers**
1. [Installation](#1-installation)
2. [Usage](#2-usage)
3. [Configuration](#3-configuration)
4. [Tests](#4-tests)


## Usage

### 1. About

**Device Management Service** or **DMS** is a program that allows users to run various computational workloads on a distributed set of machines. These machines are CPU/GPU-enabled devices that are part of the Nunet network. Think of this as a cloud service, but provided by multiple providers instead of a single entity like Amazon or Google.

For those with available hardware resources, they can earn rewards by onboarding their machines to NuNet. Each time their machine is used to run a computational job, they are eligible to receive compensation for their service.

To summarize, DMS connects users needing computational resources with those who can provide them, enabling the execution of computational work in a distributed setting.

#### Payment

All transactions on the Nunet network are expected to be conducted using the platform's utility token [NTX](https://docs.nunet.io/infohub/tokenomics/ntx-utility-token-overview). However, DMS is currently in development, and NTX payments are expected to be implemented in the [Public Alpha Mainnet](https://gitlab.com/groups/nunet/-/milestones/46#tab-issues) milestone.

**Note**: If you are a developer, please check out the [DMS specifications](#specification-for-developers) and [Getting Started](#getting-started-for-developers) sections of this document.

### 2. Installation

Before diving into the installation process, let’s take a quick look at the system requirements and a few things to keep in mind.

#### Installing via Virtual Machines or Windows Subsystem for Linux (WSL)

For VM or WSL installations, using Ubuntu 20.04 is highly recommended.

##### Things to Keep in Mind for VMs

- Skip doing an [unattended installation](https://www.virtualbox.org/manual/ch01.html#create-vm-wizard-unattended-install) for the new Ubuntu VM as it might not add the user with administrative privileges.
- Enable [Guest Additions](https://www.virtualbox.org/manual/ch04.html) when installing the VM (VirtualBox only).
- Always [change the default NAT network setting to Bridged](https://www.techrepublic.com/article/how-to-set-bridged-networking-in-a-virtualbox-virtual-machine) before booting the VM.
- [Install Extension Pack](https://phoenixnap.com/kb/install-virtualbox-extension-pack) if using VirtualBox (recommended).
- [Install VMware Tools](https://kb.vmware.com/s/article/1014294) if using VMware (recommended).
- ML on GPU jobs on VMs are not supported.

##### Things to Keep in Mind for WSLs

- Install WSL through the Windows Store.
- Install the [Update KB5020030](https://www.catalog.update.microsoft.com/Search.aspx?q=KB5020030) (Windows 10 only).
- Install Ubuntu 20.04 through WSL.
- Enable [systemd on Ubuntu WSL](https://www.xda-developers.com/how-enable-systemd-in-wsl).
- ML Jobs deployed on Linux cannot be resumed on WSL.

Though it is possible to run ML jobs on Windows machines with WSL, using Ubuntu 20.04 natively is highly recommended. The reason is that our development is centered around the Linux operating system. Additionally, the system requirements when using WSL would increase by at least around 25%.

If you are using a dual-boot machine, make sure to use the `wsl --shutdown` command before shutting down Windows and running Linux for ML jobs. Also, ensure your Windows machine is not in a hibernated state when you reboot into Linux.

#### CPU-only Machines

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

### Step-by-Step Installation Process

Here is a step-by-step process to install the Device Management Service (DMS) on a machine:

1. **Download the DMS package**:

   Download the latest version with this command:

   ```bash
   wget https://d.nunet.io/nunet-dms-latest.deb -O nunet-dms-latest.deb
   ```

2. **Install DMS**:

   Navigate to the directory where you downloaded the DMS package and install the DMS with this command:

   ```bash
   sudo apt update && sudo apt install ./nunet-dms-latest.deb -y
   ```

   If the above fails, try using `dpkg` instead:

   ```bash
   sudo dpkg -i nunet-dms-latest.deb
   sudo apt -f install -y
   ```

   Check if DMS is running. Either look for the _nunet_ process with:

   ```bash
   ps aux | grep nunet
   ```

   Or use systemd:

   ```bash
   sudo systemctl status nunet-dms.service
   ```

   If it is not running and you notice errors, submit a bug report by following [these](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/contributing_guidelines/README.md#how-to-report-a-bug) guidelines.

3. **Uninstall DMS** (if needed):

   To remove DMS, use this command:

   ```bash
   sudo apt remove nunet-dms
   ```

   To download and install a new DMS package, repeat steps 1 and 2.

4. **Completely Remove DMS** (if needed):

   To fully uninstall and stop DMS, use either of these commands:

   ```bash
   sudo apt purge nunet-dms
   ```

   Or

   ```bash
   sudo dpkg --purge nunet-dms
   ```

5. **Update DMS**: 

   To update the DMS to the latest version, follow these steps in sequence:
   - a. Download the latest DMS package (Step 1)
   - b. Install the new DMS package (Step 2)

## Specification

### 1. Description

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

### 2. Design and Architecture

#### Conceptual Basis

Main concepts of the architecture of DMS, the main component of the NuNet platform, can be found in this [Yellow Paper](https://gitlab.com/nunet/publisher/platform-yellow-paper/-/tree/main).

#### Ontology

The Nunet Ontology, which forms the basis of the design, is explained in the articles below:

- [NuNet Job Orchestration I: Ontology and Nomenclature](https://nunet.gitlab.io/research/blog/posts/ontology-and-nomenclature/)
- [NuNet Job Orchestration II: Scheduling](https://nunet.gitlab.io/research/blog/posts/scheduling-and-orchestration/)
- [NuNet Job Orchestration III: Mapping Ontology to Scheduling](https://nunet.gitlab.io/research/blog/posts/taxonomy-of-job-scheduling/)

#### Architecture

Refer to the following items to understand **DMS architecture** at a high level.

- [DMS Architecture -- Understanding I](https://nunet.gitlab.io/research/blog/posts/dms-architecture/)
- [Entity Diagram - DMS High Level](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/entityDiagrams/New_DMS_Structure_Highlevel.drawio.svg)

#### Research

Relevant research work that has informed the design of DMS can be found below:

- [Detailed Job Orchestration Sequences I](https://nunet.gitlab.io/research/blog/posts/job-orchestration-details/)
- [Detailed Job Orchestration Sequences II](https://nunet.gitlab.io/research/blog/posts/orchestration-discussion/)
- [Gossipsub, DHT, and Push/Pull Mechanisms](https://nunet.gitlab.io/research/blog/posts/gossipsub/)
- [Parent-Child Hierarchy, Allocations, and Failure Tolerance](https://nunet.gitlab.io/research/blog/posts/parent-child-relations/)
- [Kubernetes Integration Specs](https://nunet.gitlab.io/research/blog/posts/kubernetes-integration/)

### 3. Functionality

`TBD`

### 4. Data Types

Refer to the DMS global class diagram in [this](#7-class-diagram) section and various packages for data models.

### 5. Testing

Some packages contain tests, and it is always best to run them to ensure there are no broken tests before submitting any changes. Before running the tests, the Firecracker executor requires some test data, such as a kernel file, which can be downloaded with:

```bash
make testdata
```

After the download is complete, all unit tests can be run with:

```bash
go test ./...
```

### 6. References

In addition to the relevant links added in the sections above, you can also find useful links here: [NuNet Links](https://www.nunet.io/links).

### 7. Class Diagram

The global class diagram for the DMS is shown below.

#### Source File

[Global Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/class_diagram.puml?ref_type=heads)

#### Rendered from Source File

```plantuml
!include https://gitlab.com/nunet/device-management-service/-/raw/develop/specs/class_diagram.puml
```


## Getting Started for Developers

### 1. Installation

The cleanest way to set up a development environment is to build a .deb package out of this repository and let the installer do the work for you.

```bash
sudo apt install build-essential curl jq iproute2 libsystemd-dev
```

#### Prerequisites

To build the .deb package, you'd need to install these two packages:

```bash
sudo snap install go
sudo apt install build-essential libsystemd-dev
```

#### Build, Install & Setup Dev Environment

We provide a `dev-setup` shell script to ease the process of getting started. It does the following:

1. Sets up `pre-commit` hook, which runs tests before every commit.
2. Builds the .deb file and installs it.
3. Stops the `nunet-dms` service so that we can run `main.go` directly.

Run the script as follows:

```bash
bash maint-scripts/dev-setup.sh
```

Once the environment is set up, build the DMS as follows:

```bash
go build -o nunet
```

### 2. Usage

To run the DMS:

```bash
sudo ./nunet
```

Note: We're using `sudo` because the onboarding process writes some configuration files to `/etc/nunet` by default. It is possible to change this path using the configuration file, which is explained later in this README.

#### Onboarding

You don't necessarily need to onboard for development, but that depends on which part you're working on. To onboard during development, `/etc/nunet` needs to be manually created since it is created with the package during installation.

Onboarding instructions can be found here: [Onboarding Instructions](https://gitlab.com/nunet/team-processes-and-guidelines/-/tree/main/onboarding_instructions).

#### REST Endpoints

A [Postman collection](https://gitlab.com/nunet/device-management-service/-/snippets/2507804) is available to help you get started with REST endpoints exploration. Head over to project's issue section and create an issue with your question.

### 3. Configuration

#### Run Two DMS Instances Side by Side

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
    "listen_address": [
      "/ip4/0.0.0.0/tcp/9100",
      "/ip4/0.0.0.0/udp/9100/quic"
    ]
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

Please use absolute paths to avoid trouble. Also, have a look at the [config structure](https://gitlab.com/nunet/device-management-service/-/blob/develop/internal/config/config.go).

3. You must also change the port number in the `nunet` shell script if you plan to use the `nunet` CLI.

**Step 3**:

Onboard both DMS instances.

**Step 4**:

Check if both can discover each other.

### 4. Tests

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

