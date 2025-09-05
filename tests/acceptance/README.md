# NuNet Acceptance Tests

<!--toc:start-->
- [NuNet Acceptance Tests](#nunet-acceptance-tests)
  - [1. Install Incus](#1-install-incus)
  - [2. Initialize Incus Admin (after opening a new terminal session)](#2-initialize-incus-admin-after-opening-a-new-terminal-session)
    - [2.1 Install QEMU for VM Support](#21-install-qemu-for-vm-support)
    - [2.2 Optional: Check Incus Virtualization Support](#22-optional-check-incus-virtualization-support)
  - [3. Host Requirements for Running Incus Virtual Machines](#3-host-requirements-for-running-incus-virtual-machines)
    - [3.1 Check if Your CPU Supports KVM](#31-check-if-your-cpu-supports-kvm)
    - [3.2 Ensure KVM Modules Are Loaded](#32-ensure-kvm-modules-are-loaded)
    - [3.3 Check If Nested Virtualization Is Enabled](#33-check-if-nested-virtualization-is-enabled)
    - [3.4 Enable Nested Virtualization](#34-enable-nested-virtualization)
    - [3.5 If Running Inside a Virtual Machine (e.g., VirtualBox)](#35-if-running-inside-a-virtual-machine-eg-virtualbox)
    - [3.6 Verify Incus VM Support](#36-verify-incus-vm-support)
  - [4. Provision the required supporting infrastructure](#4-provision-the-required-supporting-infrastructure)
    - [4.1. Running infrastructure tests](#41-running-infrastructure-tests)
  - [5. Run the Acceptance Tests](#5-run-the-acceptance-tests)
  - [6. Using a Remote Incus Server](#6-using-a-remote-incus-server)
<!--toc:end-->

This document explains how to set up and run the NuNet acceptance tests, which currently use Incus, a Linux-native system for efficiently running and managing containers and virtual machines.

If you are a Mac users, see the section [Using a Remote Incus Server](#using-a-remote-incus-server).

## 1. Install Incus

Follow the official tutorial to install and initialize Incus:
[Install and initialize Incus](https://linuxcontainers.org/incus/docs/main/tutorial/first_steps/#install-and-initialize-incus)

## 2. Initialize Incus Admin (after opening a new terminal session)

Until you restart, you need to run the following command every time you open a
new terminal before running the tests:

```bash
incus admin init --minimal
```

### 2.1 Install QEMU for VM Support

To run Incus virtual machines, you must have QEMU installed.

For Ubuntu/Debian systems:

```bash
sudo apt install qemu-system-x86 qemu-utils
```

To verify QEMU is available:

```bash
which qemu-system-x86_64
```

If the above command returns a path (e.g., `/usr/bin/qemu-system-x86_64`), QEMU is properly installed.

Without QEMU, any attempt to launch Incus VMs will fail.

### 2.2 Optional: Check Incus Virtualization Support

To verify if Incus can manage VMs on your system, you can check:

```bash
incus info | grep driver
```

You should see:

```bash
driver: qemu
```

If you only see `driver: lxc`, then Incus is only supporting containers. This indicates missing VM support due to lack of QEMU or KVM.

## 3. Host Requirements for Running Incus Virtual Machines

To run virtual machines (VMs) with Incus, your system must support hardware virtualization (KVM). If you're running Incus inside another virtual machine (e.g., VirtualBox or a cloud VM), you must also enable nested virtualization.

### 3.1 Check if Your CPU Supports KVM

```bash
egrep -c '(vmx|svm)' /proc/cpuinfo
```

If the result is `>=1`, your processor supports hardware virtualization.
`vmx` indicates Intel VT-x; `svm` indicates AMD-V.

### 3.2 Ensure KVM Modules Are Loaded

```bash
lsmod | grep kvm
```

Expected output for Intel CPUs:

```ngix
kvm_intel             233472  0
kvm                   770048  1 kvm_intel
```

If the modules are not loaded, run:

```bash
sudo modprobe kvm
sudo modprobe kvm_intel   # Or sudo modprobe kvm_amd for AMD CPUs
```

### 3.3 Check If Nested Virtualization Is Enabled

For Intel:

```bash
cat /sys/module/kvm_intel/parameters/nested
```

For AMD:

```bash
cat /sys/module/kvm_amd/parameters/nested
```

If the output is `Y`, nested virtualization is enabled.
If `N`, it is disabled.

### 3.4 Enable Nested Virtualization

Temporary (until reboot):

```bash
sudo modprobe -r kvm_intel
sudo modprobe kvm_intel nested=1
```

Permanent:

```bash
echo "options kvm_intel nested=1" | sudo tee /etc/modprobe.d/kvm-intel.conf
sudo modprobe -r kvm_intel
sudo modprobe kvm_intel
```

(Use `kvm_amd` instead of `kvm_intel` if you're on an AMD CPU.)

### 3.5 If Running Inside a Virtual Machine (e.g., VirtualBox)

Nested virtualization must also be enabled in the hypervisor.

From the host machine:

```bash
VBoxManage modifyvm <VM_NAME> --nested-hw-virt on
```

Or via VirtualBox GUI:

1. Open VM settings
2. Go to "System" > "Processor"
3. Enable "Nested VT-x/AMD-V"

### 3.6 Verify Incus VM Support

To test launching a virtual machine:

```bash
incus launch images:ubuntu/22.04 vm-test --vm
```

## 4. Provision the required supporting infrastructure

The acceptance tests require Incus images to execute tests in Incus VMs or containers.
To create these images locally, run from the root directory of the `device-management-service` project:

```bash
cd tests/acceptance/scripts
./create-images-acc-test.sh
```

The acceptance tests require a GlusterFS Cluster to be provisioned in order to
execute some tests related to the storage package.

To provision the infrastructure locally, first set `DMS_ACC_TEST_CONFIG_FILE` and then:

```bash
make provision-acc-tests-infra
```

The script will rebuild DMS and launch the infrastructure using a dist file
from dms built with the current branch and add the required information to the
acceptance tests config file.

To reprovision the cluster without rebuilding DMS:

```bash
make reprovision-acc-tests-infra
```

To deprovision the cluster:

```bash
make deprovision-acc-tests-infra
```

### 4.1. Running infrastructure tests

There are independent tests for the infrastructure to assess basic
functionality, which can work as a sanity check in case the acceptance tests
fail.

To run these tests, change the `config.yml` file or copy it from
`confit.dist.yml` and change the number of clients to `1`, reprovision the
infrastructure with `make provision-acc-tests-infra`, then execute:

```
make test-acc-tests-infra
```

## 5. Run the Acceptance Tests

From the root directory of the `device-management-service` project, execute to run acceptance tests using VMs (default):

```bash
make run-acceptance
```

To run acceptance tests using containers execute:

```bash
make run-acceptance-container
```

Docker and Incus may conflict if both are installed on the same machine. To fix this, run the following commands:

```bash
iptables -I DOCKER-USER -i incusbr0 -j ACCEPT
iptables -I DOCKER-USER -o incusbr0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
```

More information can be found in the official documentation:
[Prevent connectivity issues with Incus and Docker](https://linuxcontainers.org/incus/docs/main/howto/network_bridge_firewalld/#prevent-connectivity-issues-with-incus-and-docker)

## 6. Using a Remote Incus Server

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
