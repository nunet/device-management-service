# Device Management Service (DMS)

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
- [Secure coding guidelines](https://gitlab.com/nunet/documentation/-/wikis/secure-coding-guidelines)

## Table of Contents

Specification
1. [Description](#1-description)
2. [Design and Architecture](#2-design-and-architecture)
3. [Functionality](#3-functionality)
4. [Data Types](#4-data-types)
5. [Testing](#5-testing)
6. [References](#6-references)
7. [Class Diagram](#7-class-diagram)

Getting Started
1. [Installation](#1-installation)
2. [Usage](#2-usage)
3. [Configuration](#3-configuration)
4. [Tests](#4-tests)

_Note_: This `README` is intended for developers. For end users, please check the [README-USR](https://gitlab.com/nunet/device-management-service/-/blob/develop/README-USR.md).

## Specification

### 1. Description

NuNet is a computing platform that provides globally distributed and optimized computing power and storage for decentralized networks, by connecting data owners and computing resources with computational processes in demand of these resources. NuNet provides a layer of intelligent interoperability between computational processes and physical computing infrastructures in an ecosystem which intelligently harnesses latent computing resources of the community into the global network of computations.

Detailed information about NuNet platform, concepts, architecture, models, stakeholders can be found in these two papers: 
- [White Paper](https://docs.nunet.io/nunet-whitepaper)
- [Yellow Paper](https://gitlab.com/nunet/publisher/platform-yellow-paper/-/tree/main)

DMS (Device Management Service) acts as the foundation of the NuNet platform, orchestrating the complex interactions between various computing resources and users. DMS implementation is structured into packages creating a more maintainable, scalable, and robust codebase that is easier to understand, test, and collaborate on. These are the existing packages in DMS and the purpose of each one.

- `dms`: Responsible for starting the whole application while being dependent on other packages for most functionality.

- `internal`: Code which will not be imported by any other packages and used only on the running instance of DMS. This includes all configuration related code, background tasks etc.

- `db`: Database used by the DMS.

- `storage`: Disk storage management on each DMS for data related to DMS and jobs deployed by DMS. It also acts a an adapter to external storage services.

- `orchestrator`: Responsible for job scheduling workflow and settlement of the contract.

- `jobs`: Manage local jobs through whatever executor it's running (Container, VM, Direct_exe, Java etc...)

- `api`: All API functionality (including REST API etc) to interact with the DMS.

- `cmd`: Command line functionality and tools.

- `network`: All network related code such as p2p communication, IP over Libp2p and other networks that might be needed in the future.

- `executor`: Responsible for executing the jobs received by the DMS. Interface to various executors such as Docker, Firecracker etc.

- `telemetry`: Logs, traces and everything related to telemetry.

- `plugins`: Defined entry points and specs for third party plugins, registration and execution of plugin code

- `models`: Contains data models imported by various packages.

- `utils`: Utility tools and functionalities.

- `tokenomics`: Interaction with blockchain for the crypto-micropayments layer of the platform.


### 2. Design and Architecture

#### Conceptual Basis

Main concepts of the architecture of DMS, that is the main component of NuNet platform, can be found in this [Yellow Paper](https://gitlab.com/nunet/publisher/platform-yellow-paper/-/tree/main).

#### Ontology

The Nunet Ontology which forms the basis of the design is explained in the below articles:

- [NuNet Job Orchestration I: Ontology and Nomenclature](https://nunet.gitlab.io/research/blog/posts/ontology-and-nomenclature/)
- [NuNet Job Orchestration II: Scheduling](https://nunet.gitlab.io/research/blog/posts/scheduling-and-orchestration/)
- [NuNet Job Orchestration III: Mapping Ontology to Scheduling](https://nunet.gitlab.io/research/blog/posts/taxonomy-of-job-scheduling/)

#### Architecture

Refer to the below items for understanding **DMS architecture** at a high level.

- [DMS architecture -- Understanding I](https://nunet.gitlab.io/research/blog/posts/dms-architecture/)

- [Entity Diagram - DMS high level](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/entityDiagrams/New_DMS_Structure_Highlevel.drawio.svg)

#### Research

Relevent research work that has informed the design of DMS can be found below:

- [Detailed Job Orchestration Sequences I](https://nunet.gitlab.io/research/blog/posts/job-orchestration-details/)

- [Detailed job orchestration sequences II](https://nunet.gitlab.io/research/blog/posts/orchestration-discussion/)

- [Gossipsub, DHT and push/pull mechanisms](https://nunet.gitlab.io/research/blog/posts/gossipsub/)

- [Parent-child hierarchy, allocations starting allocations and failure tolerance](https://nunet.gitlab.io/research/blog/posts/parent-child-relations/)

- [Kubernetes Integration Specs](https://nunet.gitlab.io/research/blog/posts/kubernetes-integration/)


### 3. Functionality

`TBD`


### 4. Data Types

Refer to DMS global class diagram in [this](#7-class-diagram) section and various packages for data models.

### 5. Testing
Some packages contain tests and it's always best to run them and make sure there are no broken tests before submitting any changes. Before running the tests, the firecracker executor requires some test data such as a kernel file which can be downloaded with:

```
make testdata
```

After the download is done, all unit tests can be run with:

```
go test ./...
```

### 6. References
In addition to the relevant links added in the sections above, you can also find useful links here: https://www.nunet.io/links

### 7. Class Diagram

The global class diagram for the DMS is shown below.

#### Source file

[Global Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/class_diagram.puml?ref_type=heads)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = ""
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

## Getting Started

### 1. Installation

The cleanest way to setup development environment is to build a deb package out of this repository and let the installer do the work for you.

```
sudo apt install build-essential curl jq iproute2 libsystemd-dev
```

#### Prerequisites

To build the deb, you'd be required to install these two packages:

```
sudo snap install go
sudo apt install build-essential libsystemd-dev
```

#### Build, Install & Setup Dev Env

We provide a dev-setup shell script to ease the process of getting started. It does the following:

1. Setup `pre-commit` hook which runs test before every commit.
2. Build .deb file and installs it.
3. Stops the `nunet-dms` service so that we can run main.go directly.

Run the script as follows:

```
bash maint-scripts/dev-setup.sh
```

Once the env is setup, build the DMS as follows:
```
go build -o nunet
```

### 2. Usage

To run the DMS:

```
sudo ./nunet
```

Notice we're using `sudo` as the onboarding process writes some configuration files to `/etc/nunet` by default. It's possible to change this path using the configuration file which is explained later in this readme.

#### Onboarding

You don't necessarily need to onboard for development, but that depends which part you're working on. To onboard during development, `/etc/nunet` need to be manually created since it's created with the package during installation.

Onboarding instructions can be found at [Onboarding Wiki](https://gitlab.com/nunet/device-management-service/-/wikis/Onboarding)

#### REST Endpoints

A [Postman collection](https://gitlab.com/nunet/device-management-service/-/snippets/2507804) is there to help you get starting with REST endpoints exploration. Head over to project's issue section and create an issue with your question.


### 3. Configuration

#### Run two DMS side by side

As a developer, you might be in a situation where you have to both service provider and compute provider. For that you'll have to run 2 DMS, one acting as SP and another CP.

**Step 1**:

Clone the repo to 2 different directory. You might want to use descriptive directory names to avoid confusion.

**Step 2**:

You need to modify some configuration so that both the DMS does not create a deadlock. Those configuration include:

1. The port DMS is listening on.
2. The database file DMS uses.

dms_config.json can be used to modify those settings. Here is a sample config file which can be modified to your taste and used:

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

Please use absolute paths to keep yourself out of trouble. Moreover, have a look at [config structure](https://gitlab.com/nunet/device-management-service/-/blob/develop/internal/config/config.go).

3. You must also change the port number in the nunet shell script if you are planning to use nunet cli.

**Step 3**:

Onboard both DMSes.

**Step 4**:

Check if both can discover each other.


### 4. Tests

**Running Functional Tests with Python-Behave Framework**

**Pre-requisites**

- **Python Installation:** Ensure Python 3.8 or higher is installed on your system.

- **Environment Variable:** Confirm that the Python path is included in the system's PATH environment variable.

**Steps to Run Functional Tests**

**1. Clone the Test-Suite Repository**

- Open your terminal and execute the following command to clone the repository from GitLab:

  `git clone git@gitlab.com:nunet/test-suite.git
`

**2. Install the Dependencies**

- Navigate to the cloned test-suite directory:

  `cd test-suite
`


- Install the necessary dependencies by running:

  `pip install -r requirements.txt
`

**3. Executing the Tests**

- Navigate to the functional tests directory:

  `cd /test-suite/stages/functional_tests
`
- Run the tests using the behave command:

  `behave features/<dir or feature file path>
`
- Example

  `behave features/device-management-service/api-tests/p2p_api.feature
`
- Note: This command will execute all Gherkin feature files or scenarios under the specified directory. Standard output logs will be displayed in the terminal.

**4. Generate Allure Report (Optional)**

[Allure](https://allurereport.org/docs/behave/) is a HTML-based reporting framework. If you want to generate the report in a fancy way,Follow these steps to generate and view the report:

**Pre-requisites for Allure**
- Install Allure from [Allure Releases](https://github.com/allure-framework/allure2/releases).

**Steps to Generate and View Allure Report**

- Run the Tests with Allure Formatter:

  `behave features/device-management-service/<feature files to run> --junit -f allure_behave.formatter:AllureFormatter -o allure-results
`

- Example:

  `behave features/device-management-service/api-tests/p2p_api.feature --junit -f allure_behave.formatter:AllureFormatter -o allure-results
`
- Generate the HTML Report:

  `allure generate allure-results -o allure-report
`

- View the Report:

  `allure open allure-report
`
- Zip the Report for Sharing:

  `zip -r allure-report.zip allure-report
`


By following these steps, you can effectively run functional tests using the Python-Behave framework and generate comprehensive reports using Allure.


