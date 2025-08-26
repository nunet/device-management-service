# Device Management Service (DMS)

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

# Development guidelines

<!--toc:start-->
- [Device Management Service (DMS)](#device-management-service-dms)
- [Development guidelines](#development-guidelines)
  - [First things first](#first-things-first)
  - [Testing](#testing)
    - [e2e Tests](#e2e-tests)
      - [Prerequisites](#prerequisites)
      - [Running the e2e tests](#running-the-e2e-tests)
    - [Acceptance tests](#acceptance-tests)
  - [Manual Testing](#manual-testing)
    - [Running DMS and config file](#running-dms-and-config-file)
      - [Config file](#config-file)
    - [Actor behaviors](#actor-behaviors)
    - [Running multiple DMS instances](#running-multiple-dms-instances)
      - [Configuration file](#configuration-file)
      - [Setting up capabilities](#setting-up-capabilities)
    - [Deployment and Onboarding](#deployment-and-onboarding)
    - [Logs](#logs)
    - [Debugging tips](#debugging-tips)
      - [Calling behaviors](#calling-behaviors)
      - [Check logs](#check-logs)
      - [Working with subnets](#working-with-subnets)
    - [Local execution of unit tests](#local-execution-of-unit-tests)
    - [Local execution of acceptance tests](#local-execution-of-acceptance-tests)
<!--toc:end-->

## First things first

Before anything, you probably want to read:

- [NuActor](./actor/README.md): it gives a general context on NuActor communication and security model.
- [Deployment](./dms/jobs/README.md): explains how orchestration of ensembles works.
- **Onboarding**: for now, you just have to be aware that hardware resources are **not** automatically available
  to the network. Nodes looking forward to making their hardware available should follow the [onboarding steps](./docs/onboarding/README.md)
- Before running a DMS, you probably want to know about the installation process (dependencies, linux permissions..),
  read the [main README](./README.md) for that.

## Testing

The repository contains unit tests, end-to-end (e2e) tests, and acceptance tests.

Most packages contain unit tests, and it is always best to run them to ensure there is nothing broken before submitting changes.

All unit tests can be run with the following command. It's necessary to include the `unit` tag to exclude other tests such as e2e tests.

```bash
go test --tags unit ./...
```

### e2e Tests

#### Prerequisites

Before running the e2e tests, make sure that the following commands are run:

```bash
sudo modprobe fuse
docker pull ghcr.io/gluster/gluster-containers:fedora
docker pull nginxdemos/hello:plain-text
docker pull ubuntu:22.04
docker pull hello-world
sudo chmod 777 "/etc/glusterfs" "/var/lib/glusterd" "/var/log/glusterfs" "/glusterfs_data"
sudo sed -i 's/#user_allow_other/user_allow_other/g' /etc/fuse.sh
```

#### Running the e2e tests

To run the e2e tests, use the following command:

```bash
make e2e
```

Help in contributing tests is always appreciated :)

### Acceptance tests

Acceptance tests are located in the `tests/acceptance` directory. They are designed to test the DMS functionality in a more integrated manner, simulating real-world scenarios.
It's recommended to first read the [Acceptance Tests README](./tests/acceptance/README.md) for detailed instructions on how to set up and run the tests.

To run the acceptance tests, use the following command:

```bash
make run-acceptance
```

## Manual Testing

When manually testing DMS, you usually want to setup multiple DMSes either under the same
machine or across different machines (or VMs).

Your DMSes should also use their own NuNet private network so that deployments
get limited for the peers you're controlling.

Let's explore how to do exactly this:

### Running DMS and config file

When running a DMS daemon, you probably want to use a specific capability context as in:

```bash
nunet run -c <cap-context>
# if context is not specified, defaults to:
nunet run -c dms
```

What that does is: for this DMS instance, all the actor authorization procedures
will rely on the specified capability context.

#### Config file

All `nunet` commands rely on a configuration file `dms_config.json` that
defines certain parameters of the DMS.

Use the following to open, and possibly edit, your configuration file.

> Make sure you have set EDITOR environment variable first

```bash
nunet config edit
```

> **IMPORTANT**: you don't have to create a `dms_config.json` file to run a
> dms. DMS will use an in-memory configuration with default values.
>
> `dms_config.json` will only explictly be written to your disk if you either
> create it manually or if you use `nunet config edit`

Everytime you use `nunet` command, it will look for the `dms_config.json` file
on the following order:

1. Current directory
2. `~/.nunet`
3. `/etc/nunet`

> _Be careful_: if you executed a `nunet run` command on one directory containing
> a specific `dms_config.json` file and execute `nunet actor cmd ...` on another directory,
> it may find a different configuration (or simply initialize a default config in-memory).
>
> What is the problem with that?
>
> `nunet actor cmd` might try to contact DMS daemon using the wrong REST port
> since they might be defined differently from both configuration files used by each command.

### Actor behaviors

As you have seen from the _NuActor_ documentation, most functionalities requested
and processed between actors happen through actor behaviors.

Some behaviors, not all, can be invoked using the `nunet actor cmd` command.

> Of course, a DMS daemon must be running.

See all available behaviors invokable with cmd running:

```bash
nunet actor cmd
```

To know information (e.g.: available flags) about a cmd-behavior, run:

```bash
nunet actor cmd <cmd-behavior> --help
# e.g.:
nunet actor cmd /dms/node/peers/self --help
```

### Running multiple DMS instances

If you're running multiple DMSes on different machines or VMs, you might
skip the following [Configuration file](#configuration-file) section.

Otherwise, if you're running all on the same machine, be sure to change
the configuration of each first.

#### Configuration file

> **Note**: this step is only necessary if you're running all instances on the _same_ machine

1. Create a directory for each DMS
2. Run `nunet config edit` inside each directory to explictly multiple `dms_config.json` files

> Recommended to use meaningful names for each DMS

Example using a DMS named `bob` (`bob` would be used for the capability
context name too):

```bash
mkdir -p ~/nunet/bob && cd ~/nunet/bob && nunet config edit
```

For each DMS, you have to change the following:

1. `p2p.listen_address` (just change both ports being used)
2. `rest.port`
3. `profiler.port`
4. `observability.log_file` (avoid overwriting log file)
5. `general.work_dir` (recommended to use same previous dir as `~/nunet/bob`)
6. `general.data_dir` (recommended to use same previous dir as `~/nunet/bob/data`)

**IMPORTANT**: recommended to run both `nunet run` and other nunet commands (such as
`nunet actor cmd`) from the _same_ directory **for each DMS instance**. Then, commands will use the same
configuration as you wish depending on the directory you're located.

#### Setting up capabilities

The repository contains two interactive scripts to make the capability setup easier:

1. `./maint-scripts/quickstart.sh`
2. `./maint-scripts/private_network.sh`

`quickstart.sh` goes through the process of creating keys and capability contexts for your identities
(one for user, another for node). It also anchors your user as root on your node by default.

`private_network.sh` will enable you to create or join an existing private network where you will able to
make deployments. This script deals with granting, setting anchors and delegating capabilities between the
parties.

It is recommended to run `quickstart.sh` before `private_network.sh`

For manually setting up capabilities or additional information, please refer to [Private Network Guide](./docs/private_network/README.md).

### Deployment and Onboarding

After having setup all your DMSes (including granting capabilities between them),
you may want to onboard some specific DMSes (or all) that will make their hardware
resources available to your capability pool.

For this, just follow the [onboarding guide](./docs/onboarding/README.md).

After having onboarded the DMSes, see the [deployment guide](./docs/deployments/README.md) to
actually deploy an ensemble in one of your onboarded nodes.

### Logs

Change the logging level either through the configuration file or exporting the following env var:

```bash
export GOLOG_LOG_LEVEL=DEBUG
```

Some `libp2p` and networking logs are silenced by default. To enable them, export:

```bash
export DMS_CONN_LOGS=true
```

### Debugging tips

It is assumed for some tips that you have access to all machines running DMS.

It's possible to export `DMS_PASSPHRASE` variable to avoid CLI prompt for passphrase. For now, make sure all your keys share
the same passphrase.

```bash
export DMS_PASSPHRASE=1234
```

#### Calling behaviors

You can call actor behaviors with `actor cmd` command to help debugging ensembles. A general workflow to test deployment looks like:

```bash
# Deploy ensemble
nunet actor cmd -c dms /dms/node/deployment/new -f examples/docker_hello.yaml -t 5m  # returns ensemble ID

# List deployments
nunet actor cmd -c dms /dms/node/deployment/list

# Check status of ensemble
# (if all allocations are of type 'task', then when they all get finished,
# the status will be set to 'Completed')
nunet actor cmd -c dms /dms/node/deployment/status -i <ensemble_id>

# Detailed information of deployment
nunet actor cmd -c dms /dms/node/deployment/manifest -i <ensemble_id>

# Get logs from running ensemble
nunet actor cmd -c dms /dms/node/deployment/logs -i <ensemble_id>

# Check how much resources were allocated (compute provider)
nunet actor cmd -c dms /dms/node/resources/allocated
```

> Refer to `nunet actor cmd --help` for all available behaviors

Currently, there are two types of ensembles:

- **task** for short-lived processes, e.g. a simple machine learning job
- **service** for long-running jobs, e.g. running a web server

You can run `docker ps` on the compute provider machines to check if allocations are effectively running.
While this can work well for _service_ ensembles, when running a _task_ ensemble the container may exit before having a chance to fetch its status. In that case,
prefer `deployment/status` or `deployment/list` behaviors.

#### Check logs

Always check logs of **all** DMS instances you have access to. DMS logs can be found
at the specified work directory in configuration file, under `jobs/` (if compute provider)
or `deployments/` for orchestrators.

**IMPORTANT**: You can retrieve logs both from DMS instances and allocations. For the latter, call `/dms/node/deployment/logs` behavior.
While allocations of type 'task' return logs automatically when it gets completed.

#### Working with subnets

To debug subnet conns, you can either opt-in the orchestrator to enter the subnet
or you try to communicate with other containers from within one of the allocations's containers.

```bash
docker exec -it <container-id> /bin/bash
```

This enables you to check, for example:

- Run `dig <name>` for debugging DNS names

- Try to `curl <alloc_name>:<alloc_port>` using information from
  another allocation running on the same ensemble.

**Note**: not always tools like `dig` and `curl` will be available on the container.
You can proceed to test with other containers or extend the available ones.

### Local execution of unit tests

To execute the unit tests locally, just run `make unit-docker`. It will build a
docker image and run the unit tests in a container reproducing the same
conditions as in the CI pipeline.

Another command to run it outside a dockerized environment exists. It's `make
unit`, but it will expect the host environment to be correctly configured for
the unit tests. It can be useful in some situations, but the command is there
to be used by the pipeline which already runs in docker, so it's recommended to
stick with `make unit-docker` for consistent results.

### Local execution of acceptance tests

The [acceptance tests README file](/tests/acceptance/README.md) describe the
prerequisites that needs to be installed in the system in order to run these
tests.

After these dependencies are installed, running the tests can be executed using
make targets.

Please refer to the README file mentioned above for detailed instructions.
