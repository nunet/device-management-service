# api

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

The api package contains all API functionality of Device Management Service (DMS). DMS exposes various endpoints through which its different functionalities can be accessed.

### Structure and Organisation

Here is quick overview of the contents of this directory:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/api/README.md?ref_type=heads): Current file which is aimed towards developers who wish to use and modify the api functionality.

* [api.go](./api.go): This file contains router setup using Gin framework. It also applies Cross-Origin Resource Sharing (CORS) middleware and OpenTelemetry middleware for tracing. Further it lists down the endpoint URLs and the associated handler functions.

* [actor.go](./actor.go): Contains endpoints for actor interaction.


### Functionality

#### Configuration
The REST server by default binds to `127.0.0.1` on port `9999`. The configuration file `dms_config.json` can be used to change to a different address and port.

The parameters `rest.port` and `rest.addr` define the port and the address respectively.

You can use the following format to construct the URL for accessing API endpoints

```
http://localhost:<port>/api/v1/<endpoint>
```

Currently, all endpoints are under the `/actor` path


##### `/actor/handle`

Retrieve actor handle with ID, DID, and inbox address

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         |`/actor/handle`   |
| **method**:           | `HTTP GET`        |
| **output**:           | `Actor Handle`   |

Response:
```
{
    "id": <actor_id>,
    "did" : <actor_did>,
    "addr" : <inbox_address>
}
```

##### `/actor/send`

Send a message to actor

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         |`/actor/handle`   |
| **method**:           | `HTTP POST`        |
| **output**:           | `{"message": "message sent"}`   |

The request should be an enveloped message
```
{
    "to": {
        "id": <actor_id>,
        "did" : <actor_did>,
        "addr" : <inbox_address>
    },
    "be": <behavior>,
    "from": {
        "id": <actor_id>,
        "did" : <actor_did>,
        "addr" : <inbox_address>
    },
    "nonce": <nonce>,
    "opt": {
        "cont": <topic_to_publish_to>,
        "exp": <time_to_expire>
    },
    "msg": <message>,
    "cap": <capability>,
    "sig": <signature>
}
```

##### `/actor/invoke`

Invoke actor with message

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         |`/actor/invoke`   |
| **method**:           | `HTTP POST`        |
| **output**:           | Enveloped Response or if error `{"error": "<error message>"}`  |

The request should be an enveloped message
```
{
    "to": {
        "id": <actor_id>,
        "did" : <actor_did>,
        "addr" : <inbox_address>
    },
    "be": <behavior>,
    "from": {
        "id": <actor_id>,
        "did" : <actor_did>,
        "addr" : <inbox_address>
    },
    "nonce": <nonce>,
    "opt": {
        "cont": <topic_to_publish_to>,     // needs to be specified for invoke
        "exp": <time_to_expire>
    },
    "msg": <message>,
    "cap": <capability>,
    "sig": <signature>
}
```

##### `/actor/broadcast`

Broadcast message to actors

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         |`/actor/broadcast`   |
| **method**:           | `HTTP POST`        |
| **output**:           | Enveloped Response or if error `{"error": "<error message>"}`  |


The request should be an enveloped message
```
{
    "to": {},                                 // should be empty for broadcast
    "be": <behavior>,
    "from": {
        "id": <actor_id>,
        "did" : <actor_did>,
        "addr" : <inbox_address>
    },
    "nonce": <nonce>,
    "opt": {
        "topic": <topic_to_publish_to>,
        "exp": <time_to_expire>
    },
    "msg": <message>,
    "cap": <capability>,
    "sig": <signature>
}
```



For more details on these Actor API endpoints, refer to the [cmd/actor package](../cmd/actor) on how the they are used.