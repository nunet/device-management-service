
# backend

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#description)
2. [Structure and Organisation](#structure-and-organisation)
3. [Class Diagram](#class-diagram)
4. [Functionality](#functionality)
5. [Data Types](#data-types)
6. [Testing](#testing)
7. [Proposed Functionality/Requirements](#proposed-functionality--requirements)
8. [References](#references)


## Specification

### Description

The backend sub package contains the actual implementation of the Nunet CLI commands.

### Structure and Organisation

Here is quick overview of the contents of this directory:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/README.md): Current file which is aimed towards developers who wish to use and modify the cmd functionality.

* [backend](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/backend.go): This file defines various interfaces DMS backend service. These interfaces provide abstractions for functionalities like resource management, peer management, network management, logging, and file system access.

* [filesystem](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/filesystem.go): This file implements FileSystem functionality using the standard `os` package. It provides functions for basic file operations like creating, opening, reading, writing, and deleting files and directories.

* [journal](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/journal.go): This file implements a wrapper to `go-systemd/sdjournal` functionality. It wraps the `sdjournal` functionality providing access to systemd journal entries like adding filters, retrieving entries, and iterating through them.

* [libp2p](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/libp2p.go): This file implements functionalites to clear incoming chat requests and decode a peer id.

* [network](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/network.go): This file implements a method to get network connections data.

* [resources](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/resources.go): This file implements a method to get total capacity of the machine.

* [utils](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/utils.go): This file implements utitlity functions for backend functionality.

* [wallet](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/wallet.go): This file contains methods to get wallet address for the user.

* [websocket](https://gitlab.com/nunet/device-management-service/-/tree/main/cmd/backend/websocket.go): This file implements a WebSocket Client for establishing and managing WebSocket connections. It provides functions to initialize the connection, send/receive messages, and handle pings for maintaining the connection.

### Class Diagram

The class diagram for the `backend` package is shown below.

#### Source file

[backend Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/cmd/backend/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/cmd/backend"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath

!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

#### ResourceManager interface

`ResourceManager` interface consists of following methods:

##### GetTotalProvisioned

- signature: `GetTotalProvisioned() *types.Provisioned`

- input: None

- output: `types.onboarding.Provisioned`

`GetTotalProvisioned` returns the total capacity of the machine.

#### PeerManager interface

`PeerManager` interface abstracts libp2p functionality. It consists of following methods:

##### ClearIncomingChatRequests

- signature: `ClearIncomingChatRequests() error`

- input: None

- output (error): `error message`

`ClearIncomingChatRequests` deletes all the incoming chat requests.

##### Decode

- signature: `Decode(s string) (peer.ID, error)`

- input: `peerID string`

- output: `decoded ID` of type string

- output (error): `error message`

`Decode` receives an encoded peerID string and returns the decoded ID.

#### WalletManager interface

`WalletManager` interface consists of following methods:

##### GetCardanoAddressAndMnemonic

- signature: `GetCardanoAddressAndMnemonic() (*types.BlockchainAddressPrivKey, error)`

- input: None

- output: `types.onboarding.BlockchainAddressPrivKey`

- output (error): `error message`

`GetCardanoAddressAndMnemonic` generates wallet on Cardano blockchain.

##### GetEthereumAddressAndPrivateKey

- signature: `GetEthereumAddressAndPrivateKey() (*types.BlockchainAddressPrivKey, error)`

- input: None

- output: `types.onboarding.BlockchainAddressPrivKey`

- output (error): `error message`

`GetEthereumAddressAndPrivateKey` generates wallet on Ethereum blockchain.


#### NetworkManager interface

`NetworkManager` interface abstracts connection on ports. It consists of following methods:

##### GetConnections

- signature: `GetConnections(kind string) ([]gonet.ConnectionStat, error)`

- input: `type of connection`

- output: list of [Connection Stats](https://pkg.go.dev/github.com/shirou/gopsutil/net#ConnectionStat)

- output (error): `error message`

`GetConnections` returns the list of network connections opened on the machine.

#### Utility interface

`Utility` interface abstracts helper functions. It consists of following methods:

##### IsOnboarded

- signature: `IsOnboarded() (bool, error)`

- input: None

- output: `bool` value showing whether the machine is onboarded or not.

- output (error): `error message`

`IsOnboarded` checks whether machine is onboarded to Nunet.


##### ReadMetadataFile

- signature: `ReadMetadataFile() (*types.Metadata, error)`

- input: None

- output: `types.onboarding.Metadata`

- output (error): `error message`

`ReadMetadataFile` reads metadata file from the machine.

##### ResponseBody

- signature: `ResponseBody(c *gin.Context, method, endpoint, query string, body []byte) ([]byte, error)`

- input #1 : `Context object`

- input #2 : `HTTP method`

- input #3 : `Endpoint URL`

- input #4 : `Query parameters string`

- input #5 : `Payload`

- output: `byte value of the API Response`

- output (error): `error message`

`ResponseBody` executes an internal call to the DMS API endpoint and returns the response.

#### WebSocketClient interface

`WebSocketClient` interface provides functionality to chat commands. It consists of following methods:

##### Initialize

- signature: `Initialize(url string) error`

- input: `target url`

- output: none

- output (error): `error message`

`Initialize` initiates the dialing procedure to connect to the provided endpoint URL.

##### Close

- signature: `Close() error`

- input: none

- output: none

- output (error): `error message`

`Close` closes the active WebSocket connection.

##### ReadMessage

- signature: `ReadMessage(ctx context.Context, w io.Writer) error`

- input #1: `Context object`

- input #1: `io writer`

- output: none

- output (error): `error message`

`ReadMessage` reads the incoming messages over a websocket connection.

##### WriteMessage

- signature: `WriteMessage(ctx context.Context, r io.Reader) error`

- input #1: `Context object`

- input #1: `io reader`

- output: none

- output (error): `error message`

`WriteMessage` sends a message over a websocket connection.

##### Ping

- signature: `Ping(ctx context.Context, w io.Writer) error`

- input #1: `Context object`

- input #1: `io writer`

- output: none

- output (error): `error message`

`Ping` sends WebSocket pings at 30 second intervals until the context is canceled or an error prevents sending the ping message.


#### Logger interface

`Logger` interface abstracts systemd journal entries. It consists of following methods:

##### AddMatch

- signature: `AddMatch(match string) error`

- input: `filter pattern string`

- output: none

- output (error): `error message`

`AddMatch` utilizes the input pattern string to filter the Systemd journal.

##### Close

- signature: `Close() error`

- input: none

- output: none

- output (error): `error message`

`Close` closes an open systemd journal.

##### GetEntry

- signature: `GetEntry() (*sdjournal.JournalEntry, error)`

- input: none

- output: `systemd journal entry`

- output (error): `error message`

`GetEntry` retrieves the next journal entry from the Systemd journal.

##### Next

- signature: `Next() (uint64, error)`

- input: none

- output: `byte offset for next entry`

- output (error): `error message`

`Next` advances the read pointer into the journal by one entry. This prepares the stage for the subsequent `GetEntry` method call.


#### FileSystem interface

`FileSystem` interface abstracts abstracts Afero/os calls. It consists of following methods:

##### Create

- signature: `Create(name string) (FileHandler, error)`

- input: `file name`

- output: `file object`

- output (error): `error message`

`Create` function creates a new file using the provided name.

##### MkdirAll

- signature: `MkdirAll(path string, perm os.FileMode) error`

- input #1: `directory path`

- input #2: `file mode and permission`

- output: none

- output (error): `error message`

`MkdirAll` creates a new directory at the specified path.

##### OpenFile

- signature: `OpenFile(name string, flag int, perm os.FileMode) (FileHandler, error)`

- input #1: `file name including the path`

- input #2: `access mode`

- input #2: `file permissions`

- output: `file object`

- output (error): `error message`

`OpenFile` opens the file with the provided name.

##### ReadFile

- signature: `ReadFile(filename string) ([]byte, error)`

- input: `file name`

- output: `byte data of the file`

- output (error): `error message`

`ReadFile` reads the file with the provided name.

##### RemoveAll

- signature: `RemoveAll(path string) error`

- input: `path`

- output: none

- output (error): `error message`

`RemoveAll` deletes all the data at the provided path.


##### Walk

- signature: `Walk(root string, walkFn filepath.WalkFunc) error`

- - input #1: `root directory path`

- input #2: `callback function`

- output: none

- output (error): `error message`

`Walk` navigates a file hierarchy, instigating a callback function at each discovery.

#### FileHandler interface

`FileSystem` interface abstracts file interfaces shared between `os` and `afero` so that both can be used interchangeably. It consists of following methods:

##### io methods (Go standard library)

- `io.Closer`
- `io.Reader`
- `io.ReaderAt`
- `io.Seeker`
- `io.Writer`
- `io.WriterAt`

##### Stat

- signature: `Stat() (os.FileInfo, error)`

- input: none

- output: [file information](https://pkg.go.dev/io/fs#FileInfo)

- output (error): `error message`

`Stat` returns information about the file.

##### WriteString

- signature: `WriteString(s string) (int, error)`

- input: `string to be written`

- output: `number of bytes written`

- output (error): `error message`

`WriteString` writes the content of the provided string to the file.

### Data Types

`ConnectionStat`: This is a data type defined in `Go` `net` package. It consists of network connection data. See [here](https://pkg.go.dev/github.com/shirou/gopsutil/net#ConnectionStat) for more details.

```
type ConnectionStat struct {
	Fd     uint32  `json:"fd"`
	Family uint32  `json:"family"`
	Type   uint32  `json:"type"`
	Laddr  Addr    `json:"localaddr"`
	Raddr  Addr    `json:"remoteaddr"`
	Status string  `json:"status"`
	Uids   []int32 `json:"uids"`
	Pid    int32   `json:"pid"`
}
```

`FileInfo`: data type defined in `Go` `fs` pacakge. It consists of file information. See [here](https://pkg.go.dev/io/fs#FileInfo) for more details.

```
type FileInfo interface {
	Name() string       // base name of the file
	Size() int64        // length in bytes for regular files; system-dependent for others
	Mode() FileMode     // file mode bits
	ModTime() time.Time // modification time
	IsDir() bool        // abbreviation for Mode().IsDir()
	Sys() any           // underlying data source (can return nil)
}
```


Refer to `cmd` package for all other data types applicable.

### Testing

The methods and interfaces in the `backend` subpacakge can be used to test the functionality of the `cmd` pacakge commands. Currently no unit test are defined since the implementaton is mostly wrappers around functions that should be tested somewhere else.

### Proposed Functionality / Requirements

#### List of issues

All issues that are related to the design of `cmd` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [cmd package design]() `TBD`

### References



