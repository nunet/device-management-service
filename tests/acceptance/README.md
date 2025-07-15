# NuNet Acceptance Tests

This document explains how to set up and run the NuNet acceptance tests, which currently use Incus, a Linux-native system for efficiently running and managing containers and virtual machines.

If you are a Mac users, see the section [Using a Remote Incus Server](#using-a-remote-incus-server).

## Running Incus Server Locally

### 1. Install Incus

Follow the official tutorial to install and initialize Incus: 
[Install and initialize Incus](https://linuxcontainers.org/incus/docs/main/tutorial/first_steps/#install-and-initialize-incus)

### 2. Initialize Incus Admin (after opening a new terminal session)

Until you restart, you need to run the following command every time you open a new terminal before running the tests:

```bash
incus admin init --minimal
```

### 3. Run the Acceptance Tests

From the root directory of the `device-management-service` project, execute:

```bash
make run-acceptance
```

Docker and Incus may conflict if both are installed on the same machine. To fix this, run the following commands:

```bash
iptables -I DOCKER-USER -i incusbr0 -j ACCEPT
iptables -I DOCKER-USER -o incusbr0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
```

More information can be found in the official documentation: 
[Prevent connectivity issues with Incus and Docker](https://linuxcontainers.org/incus/docs/main/howto/network_bridge_firewalld/#prevent-connectivity-issues-with-incus-and-docker)


## Using a Remote Incus Server

NuNet acceptance tests run on an Incus server, which is primarily designed for Linux distributions. For Mac users, one viable approach is to use a pre-configured NuNet Incus server. Although Mac users can initiate the acceptance tests locally using the command `make run-acceptance`, the tests will actually be executed on the remote Incus server.

Follow [these instructions](https://gitlab.com/nunet/test-suite/-/blob/main/infrastructure/acc-tests-config-maker/README.md) to configure access to the Incus Server. Upon completing the setup, a `config_out.yml` file will be generated with the following structure:

```yaml
incus_hosts:
  - host: host.example.com
    port: 8443
    client_cert: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
    client_key: |
      -----BEGIN EC PRIVATE KEY-----
      ...
      -----END EC PRIVATE KEY-----
    server_cert: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
```

Once you have the `config_out.yml` file, navigate to the `device-management-service` root directory and run the following commands to set the `DMS_ACC_TEST_CONFIG_FILE` variable and start the acceptance tests:

```bash
cd device-management-service
cp <path to config_out.yml> $PWD/acc-test-cfg.yml
export DMS_ACC_TEST_CONFIG_FILE=$PWD/acc-test-cfg.yml
make run-acceptance  
```

