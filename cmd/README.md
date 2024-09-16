# Introduction to Nunet Command-Line Interface (CLI)

The Nunet Command-Line Interface (CLI) serves as a powerful tool for interacting with the Nunet ecosystem, enabling you to manage network configurations, control capabilities, and handle cryptographic keys. It provides a comprehensive set of commands to streamline various tasks and operations within the Nunet network.

## Top-Level Commands

## Nunet Actor System CLI Documentation

This command provides a suite of operations tailored for interacting with the Nunet Actor System. It enables you to communicate with actors within the network, facilitating actions like sending messages, invoking specific behaviors, and broadcasting information to multiple actors simultaneously.

Detailed documentation can be found [here](./actor/README.md).

## Nunet Capability Management CLI Documentation

This command focuses on capability management within the Nunet ecosystem. It allows you to define, delegate, and control the permissions and authorizations granted to different entities, ensuring secure and controlled interactions within the network.

Detailed documentation can be found [here](./cap/README.md).


## Nunet Configuration Management CLI Documentation

The `nunet config` command provides a streamlined way to interact with and manage your Nunet configuration file directly from the command line. This allows you to view existing settings, modify them as needed, and ensure your Nunet environment is tailored to your preferences.

### Usage

```bash
nunet config [command]
```

### Available Commands

* `edit`: Open the configuration file in your default text editor for manual adjustments.
* `get`: Retrieve and display the current value associated with a specific configuration key.
* `set`: Modify the configuration file by assigning a new value to a specified key.

### Flags

* `-h, --help`: Display help information for the `config` command and its subcommands.


## Nunet Key Management CLI Documentation 


The Nunet Key Management CLI provides commands to manage keys within the Nunet Device Management Service (DMS). It allows you to generate new keypairs and retrieve the Decentralized Identifier (DID) associated with a specific key.


### Main Command


* `nunet key` 


  * **Description:** The primary command to manage keys within the Nunet DMS


  * **Usage:** `nunet key [command]`


  * **Available Commands:**

     * `did`: Retrieve the DID for a specified key

     * `new`: Generate a new keypair


  * **Flags:**

    * `-h, --help`: Display help information for the main `key` command.


### Subcommands


* `nunet key did`


   * **Description:** Retrieves and displays the DID associated with a specified key. This DID uniquely identifies the key within the Nunet ecosystem.


   * **Usage:** `nunet key did <key-name> [flags]`


   * **Arguments:**

     * `<key-name>`: The name of the key whose DID you want to retrieve


   * **Flags:**

     * `-h, --help`: Display help information for the `did` command


* `nunet key new`


   * **Description:** Generates a new keypair and securely stores the private key in the user's local keystore. The corresponding public key can be used for various cryptographic operations within the Nunet DMS


   * **Usage:** `nunet key new <name> [flags]`


   * **Arguments:**

     * `<name>`:  A name to identify the newly generated keypair


   * **Flags:**

     * `-h, --help`: Display help information for the `new` command.


**Important Considerations:**


* Keep your private keys secure, as they provide access to your identity and associated capabilities within the Nunet DMS

* Choose descriptive names for your keypairs to easily identify their purpose or associated devices.


## Nunet Run Command Documentation

### `nunet run`

**Purpose:** Starts the Nunet Device Management Service (DMS), responsible for handling network operations and device management.

**Usage:**

```bash
nunet run [flags]
```

**Flags:**

* `-c, --context string`: Specifies the key and capability context to use (default: "dms").
* `-h, --help`: Displays help information for the `run` command.

**Example:**

```bash
nunet run
```

This starts the Nunet DMS with the default "dms" context.


## Nunet TAP Command Documentation


### `nunet tap`


**Purpose:** Creates a TAP (network tap) interface to bridge the host network with a virtual machine (VM) or container. It also configures essential network settings like IP forwarding and iptables rules to enable seamless communication.


**Key Points:**


* **Root Privileges Required:** This command necessitates root or administrator privileges for execution due to its manipulation of network interfaces and system-level settings.


**Usage:**


```bash
nunet tap [main_interface] [vm_interface] [CIDR] [flags]
```

**Arguments:**


* **`main_interface`:** (e.g., eth0) The name of the existing network interface on your host machine that you want to bridge with the TAP interface.

* **`vm_interface`:** (e.g., tap0) The name you want to assign to the newly created TAP interface.

* **`CIDR`:** (e.g., 172.16.0.1/24) The Classless Inter-Domain Routing (CIDR) notation specifying the IP address range and subnet mask for the TAP interface. This ensures that the VM or container connected to the TAP has its own IP address within the specified network.


**Flags:**


* `-h, --help`: Displays help information for the `tap` command.


**Example:**
```bash

sudo nunet tap eth0 tap0 172.16.0.1/24
```

This command will create a TAP interface named 'tap0' bridged to your host's 'eth0' interface. The 'tap0' interface will be assigned an IP address of '172.16.0.1' with a subnet mask of '/24'. This configuration allows a VM or container connected to 'tap0' to access the network through your host's 'eth0' interface.


**Important Notes:**


* Ensure you have the necessary permissions to execute this command.

* Be cautious when configuring network settings, as incorrect configurations can disrupt your network connectivity.


## Nunet Log Collection CLI Documentation

### `nunet log`

**Purpose:** Gathers all Nunet logs into a compressed archive (tarball) for troubleshooting and analysis.

**Important:** Requires root privileges (`sudo`).

**Usage:**

```bash
sudo nunet log [flags]
```

**Flags:**

* `-h, --help`: Displays help information.

