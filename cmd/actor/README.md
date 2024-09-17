## Nunet Actor System CLI

The Nunet Actor System CLI provides a set of commands for interacting with the Nunet actor system, enabling you to send messages to actors, invoke behaviors, and broadcast messages across the network.

### Basic Commands

* `nunet actor msg`: Constructs a message for an actor.
* `nunet actor send`: Sends a constructed message to an actor.
* `nunet actor invoke`: Invokes a behavior in an actor and returns the result.
* `nunet actor broadcast`: Broadcasts a message to a topic, potentially reaching multiple actors.
* `nunet actor cmd`: Invokes a predefined public behavior on an actor.

### `nunet actor msg`

This command is used to create a message that can be sent to an actor. It encapsulates the behavior to be invoked and the associated payload data.

**Usage**

```bash
nunet actor msg <behavior> <payload> [flags]
```

**Arguments**

* `<behavior>`: The specific behavior you want the actor to perform upon receiving the message
* `<payload>`: The data accompanying the message, providing context or input for the behavior

**Flags**

* `-b, --broadcast string`: Designates the topic for broadcasting the message.
* `-c, --context string`: Specifies the capability context name
* `-d, --dest string`: Identifies the destination handle for the message.
* `-e, --expiry time`: Sets an expiration time for the message.
* `-h, --help`: Displays help information for the `msg` command.
* `-i, --invoke`: Marks the message as an invocation, requesting a response from the actor.
* `-t, --timeout duration`: Sets a timeout for awaiting a response after invoking a behavior.

### `nunet actor send`

This command delivers a previously constructed message to an actor.

**Usage**

```bash
nunet actor send <msg> [flags]
```

**Arguments**

* `<msg>`: The message, created using the `nunet actor msg` command, to be sent.

**Flags**

* `-h, --help`: Displays help information for the `send` command.

### `nunet actor invoke`

This command directly invokes a specific behavior on an actor and expects a response.

**Usage**

```bash
nunet actor invoke <msg> [flags]
```

**Arguments**

* `<msg>`: The message, crafted with `nunet actor msg`, containing the behavior and payload

**Flags**

* `-h, --help`: Displays help information for the `invoke` command

### `nunet actor broadcast`

This command disseminates a message to a designated topic, potentially reaching multiple actors who have subscribed to that topic.

**Usage**

```bash
nunet actor broadcast <msg> [flags]
```

**Arguments**

* `<msg>`: The message to be broadcasted

**Flags**

* `-h, --help`: Displays help information for the `broadcast` command.

Please let me know if you have any other questions.

### `nunet actor cmd`

This command invokes a behavior on an actor.

**Usage**

```bash
nunet actor cmd [flags]
nunet actor cmd [command]
```

**Available Commands**

* `/broadcast/hello`: Invoke /broadcast/hello behavior on an actor.
* `/dms/node/onboarding/offboard`: Invoke /dms/node/onboarding/offboard behavior on an actor.
* `/dms/node/onboarding/onboard`: Invoke /dms/node/onboarding/onboard behavior on an actor.
* `/dms/node/onboarding/resource`: Invoke /dms/node/onboarding/resource behavior on an actor.
* `/dms/node/onboarding/status`: Invoke /dms/node/onboarding/status behavior on an actor.
* `/dms/node/peers/connect`: Invoke /dms/node/peers/connect behavior on an actor.
* `/dms/node/peers/dht`: Invoke /dms/node/peers/dht behavior on an actor.
* `/dms/node/peers/list`: Invoke /dms/node/peers/list behavior on an actor.
* `/dms/node/peers/ping`: Invoke /dms/node/peers/ping behavior on an actor.
* `/dms/node/peers/score`: Invoke /dms/node/peers/score behavior on an actor.
* `/dms/node/peers/self`: Invoke /dms/node/peers/self behavior on an actor.
* `/dms/node/vm/list`: Invoke /dms/node/vm/list behavior on an actor.
* `/dms/node/vm/start/custom`: Invoke /dms/node/vm/start/custom behavior on an actor.
* `/dms/node/vm/stop`: Invoke /dms/node/vm/stop behavior on an actor.
* `/public/hello`: Invoke /public/hello behavior on an actor
* `/public/status`: Invoke /public/status behavior on an actor

**Flags**

* `-c, --context string`: Capability context name.
* `-d, --dest string`: Destination DMS DID, peer ID or handle.
* `-e, --expiry time`: Expiration time.
* `-h, --help`: Help for the `cmd` command.
* `-t, --timeout duration`: Timeout duration.


### Broadcast Commands

* **`/broadcast/hello`**

   * **Description:** Invokes the `/broadcast/hello` behavior on an actor. This sends a "hello" message to a broadcast topic for polite introduction.

   * **Usage:** `nunet actor cmd /broadcast/hello [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/broadcast/hello` command

### DMS Node Commands

* **`/dms/node/onboarding/offboard`**

   * **Description:**  Invokes the `/dms/node/onboarding/offboard` behavior on an actor. This is used to offboard a node from the DMS (Decentralized Management System).

   * **Usage:** `nunet actor cmd /dms/node/onboarding/offboard [<param> ...] [flags]`

   * **Flags:**
     * `-f, --force`: Force the offboarding process overriding any safety checks and invalid states.
     * `-h, --help`: Display help information for the `/dms/node/onboarding/offboard` command.

* **`/dms/node/onboarding/onboard`**

   * **Description:**  Invokes the `/dms/node/onboarding/onboard` behavior on an actor. This is used to onboard a node to the DMS, making its resources available for use.

   * **Usage:** `nunet actor cmd /dms/node/onboarding/onboard [<param> ...] [flags]`

   * **Flags:**
     * `-a, --available`: Set the node as unavailable for job deployment (default: false).
     * `-z, --cpu int`: Set the value for CPU usage.
     * `-h, --help`: Display help information for the `/dms/node/onboarding/onboard` command.
     * `-l, --local-enable`: Set server mode (enable for local) (default: true).
     * `-m, --memory uint`: Set the value for memory usage.
     * `-x, --ntx-price float`: Set the price in NTX per minute for the onboarded compute resource.
     * `-n, --nunet-channel string`: Set the channel.
     * `-w, --wallet string`: Set the wallet address.

* **`/dms/node/onboarding/resource`**

   * **Description:** Invokes the `/dms/node/onboarding/resource` behavior on an actor.  This retrieves or manages resource information related to the onboarding process.

   * **Usage:** `nunet actor cmd /dms/node/onboarding/resource [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/onboarding/resource` command.

* **`/dms/node/onboarding/status`**

   * **Description:**  Invokes the `/dms/node/onboarding/status` behavior on an actor. This is used to check the onboarding status of a node.

   * **Usage:** `nunet actor cmd /dms/node/onboarding/status [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/onboarding/status` command

* **`/dms/node/peers/connect`**

   * **Description:** Invokes the `/dms/node/peers/connect` behavior on an actor.  This initiates a connection to a specified peer.

   * **Usage:** `nunet actor cmd /dms/node/peers/connect [<param> ...] [flags]`

   * **Flags:**
     * `-a, --address string`:  The peer address to connect to
     * `-h, --help`: Display help information for the `/dms/node/peers/connect` command.

* **`/dms/node/peers/dht`**

   * **Description:**  Invokes the `/dms/node/peers/dht` behavior on an actor. This interacts with the Distributed Hash Table (DHT) used for peer discovery and content routing

   * **Usage:** `nunet actor cmd /dms/node/peers/dht [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/peers/dht` command.

* **`/dms/node/peers/list`**

   * **Description:**  Invokes the `/dms/node/peers/list` behavior on an actor. This retrieves a list of connected peers

   * **Usage:** `nunet actor cmd /dms/node/peers/list [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/peers/list` command.

* **`/dms/node/peers/ping`**

   * **Description:**  Invokes the `/dms/node/peers/ping` behavior on an actor. This sends a ping message to a specified host to check its reachability

   * **Usage:** `nunet actor cmd /dms/node/peers/ping [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/peers/ping` command
     * `-H, --host string`: The host address to ping

* **`/dms/node/peers/score`**

   * **Description:**  Invokes the `/dms/node/peers/score` behavior on an actor. This retrieves a snapshot of the peer's gossipsub broadcast score.

   * **Usage:** `nunet actor cmd /dms/node/peers/score [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/peers/score` command

* **`/dms/node/peers/self`**

   * **Description:** Invokes the `/dms/node/peers/self` behavior on an actor. This retrieves information about the node itself, such as its ID or addresses

   * **Usage:** `nunet actor cmd /dms/node/peers/self [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/peers/self` command.

* **`/dms/node/vm/list`**

   * **Description:**  Invokes the `/dms/node/vm/list` behavior on an actor. This retrieves a list of virtual machines (VMs) running on the node

   * **Usage:** `nunet actor cmd /dms/node/vm/list [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/vm/list` command.

* **`/dms/node/vm/start/custom`**

   * **Description:**  Invokes the `/dms/node/vm/start/custom` behavior on an actor. This starts a new VM with custom configurations.

   * **Usage:** `nunet actor cmd /dms/node/vm/start/custom [<param> ...] [flags]`

   * **Flags:**
     * `-a, --args string`: Arguments to pass to the kernel
     * `-z, --cpu float32`: CPU cores to allocate (default 1)
     * `-h, --help`: Display help information for the `/dms/node/vm/start/custom` command.
     * `-i, --initrd string`: Path to initial ram disk
     * `-k, --kernel string`: Path to kernel image file.
     * `-m, --memory uint`: Memory to allocate (default 1024)
     * `-r, --rootfs string`: Path to root fs image file

* **`/dms/node/vm/stop`**

   * **Description:**  Invokes the `/dms/node/vm/stop` behavior on an actor. This stops a running VM

   * **Usage:** `nunet actor cmd /dms/node/vm/stop [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/dms/node/vm/stop` command
     * `-i, --id string`: Execution id of the VM

### Public Commands

* **`/public/hello`**

   * **Description:** Invokes the `/public/hello` behavior on an actor. This broadcasts a "hello" for a polite introduction.

   * **Usage:** `nunet actor cmd /public/hello [<param> ...] [flags]`

   * **Flags:**
     * `-h, --help`: Display help information for the `/public/hello` command

* **`/public/status`**

   * **Description:**  Invokes the `/public/status` behavior on an actor. This retrieves the status or health information of the actor or system

   * **Flags:**
     * `-h, --help`: Display help information for the `/public/status` command

### Global Flags

These flags can be used with any of the above commands:

* `-c, --context string`: Specifies the capability context name. This is used for authorization or access control.
* `-d, --dest string`:  Specifies the destination for the command. This can be a DMS DID (Decentralized Identifier), a peer ID, or a handle.
* `-e, --expiry time`:  Sets an expiration time for the message or command.
* `-t, --timeout duration`: Sets a timeout duration for the command. If the command does not complete within the specified duration, it will time out.
* `-h, --help`: Display help information for the commands
