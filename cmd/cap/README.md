## Nunet Capability Management CLI Documentation

The Nunet Capability Management CLI provides commands to manage capabilities within the Nunet ecosystem. Capabilities define the permissions and authorizations granted to different entities within the system. These commands allow you to create, modify, delegate, and list capabilities.

### Commands

* `nunet cap anchor`:  Add or modify capability anchors in a capability context.
* `nunet cap delegate`: Delegate capabilities for a subject.
* `nunet cap grant`: Grant (delegate) capabilities as anchors and side chains from a capability context.
* `nunet cap list`: List all capability anchors in a capability context
* `nunet cap new`: Create a new persistent capability context for DMS or personal usage
* `nunet cap remove`: Remove capability anchors in a capability context.

### `nunet cap anchor`

This command is used to add new or modify existing capability anchors within a specific capability context. Anchors serve as the foundation for defining capabilities, establishing the core permissions and restrictions

**Usage**

```bash
nunet cap anchor [flags]
```

**Flags**

* `-c, --context string`: Specifies the operation context, defining the key and capability context to use (defaults to "user").
* `-h, --help`: Displays help information for the `anchor` command
* `--provide string`: Adds tokens as a "provide" anchor in JSON format, defining capabilities the context can offer
* `--require string`: Adds tokens as a "require" anchor in JSON format, defining capabilities the context demands
* `--root string`: Adds a DID as a "root" anchor, establishing the root authority for the context

### `nunet cap delegate`

This command delegates specific capabilities to a subject, granting them permissions within the system

**Usage**

```bash
nunet cap delegate <subjectDID> [flags]
```

**Arguments**

* `<subjectDID>`: The Decentralized Identifier (DID) of the entity receiving the delegated capabilities

**Flags**

* `-a, --audience string`:  (Optional) Specifies the audience DID, restricting the delegation to a specific recipient
* `--cap strings`:  Defines the capabilities to be granted or delegated (can be specified multiple times)
* `-c, --context string`: Specifies the operation context (defaults to "user")
* `-d, --depth uint`:  (Optional) Sets the delegation depth, controlling how many times the capabilities can be further delegated (default 0)
* `--duration duration`:  Sets the duration for which the delegation is valid
* `-e, --expiry time`:  Sets an expiration time for the delegation
* `-h, --help`: Displays help information for the `delegate` command
* `--self-sign string`:  Specifies self-signing options: 'no' (default), 'also', or 'only'
* `-t, --topic strings`:  Defines the topics for which capabilities are granted or delegated (can be specified multiple times)

### `nunet cap grant`

This command grants (delegates) capabilities as anchors and side chains from a specified capability context

**Usage**

```bash
nunet cap grant <subjectDID> [flags]
```

**Arguments**

* `<subjectDID>`: The Decentralized Identifier (DID) of the entity receiving the granted capabilities

**Flags**

* `-a, --audience string`:  (Optional) Specifies the audience DID
* `--cap strings`:  Defines the capabilities to be granted or delegated
* `-c, --context string`: Specifies the operation context (defaults to "user")
* `-d, --depth uint`: (Optional) Sets the delegation depth
* `--duration duration`: Sets the duration for which the grant is valid
* `-e, --expiry time`: Sets an expiration time for the grant
* `-h, --help`: Displays help information for the `grant` command
* `-t, --topic strings`:  Defines the topics for which capabilities are granted

### `nunet cap list`

This command lists all capability anchors within a specified capability context

**Usage**

```bash
nunet cap list [flags]
```

**Flags**

* `-c, --context string`: Specifies the operation context (defaults to "user")
* `-h, --help`: Displays help information for the `list` command

### `nunet cap new`

This command creates a new persistent capability context, which can be used for DMS or personal purposes

**Usage**

```bash
nunet cap new <name> [flags]
```

**Arguments**

* `<name>`: The name for the new capability context

**Flags**

* `-h, --help`:  Displays help information for the `new` command

### `nunet cap remove`

This command removes capability anchors from a specified capability context

**Usage**

```bash
nunet cap remove [flags]
```

**Flags**

* `-c, --context string`: Specifies the operation context (defaults to "user")
* `-h, --help`:  Displays help information for the `remove` command
* `--provide string`:  Removes tokens from the "provide" anchor in JSON format
* `--require string`: Removes tokens from the "require" anchor in JSON format
* `--root string`:  Removes a DID from the "root" anchor
