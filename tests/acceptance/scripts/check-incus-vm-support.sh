#!/bin/bash

echo "Checking Incus VM Support..."
echo "-----------------------------"

# Check CPU virtualization support
echo -n "Checking CPU virtualization support (vmx/svm): "
if egrep -q '(vmx|svm)' /proc/cpuinfo; then
    echo "Supported"
else
    echo "Not supported"
    echo "Your CPU does not support hardware virtualization (KVM)."
    exit 1
fi

# Check if KVM modules are loaded
echo -n "Checking if KVM module is loaded: "
if lsmod | grep -q '^kvm'; then
    echo "Loaded"
else
    echo "Not loaded"
    echo "Try loading it with: sudo modprobe kvm"
    exit 1
fi

# Check if QEMU is installed
echo -n "Checking if QEMU is installed: "
if which qemu-system-x86_64 > /dev/null 2>&1; then
    echo "Found"
else
    echo "Not found"
    echo "Install it with: sudo apt install qemu-system-x86 qemu-utils"
    exit 1
fi

# Check Incus driver support for VMs
echo -n "Checking if Incus supports VM driver (qemu): "
if incus info | grep -q "driver:.*qemu"; then
    echo "qemu driver is active"
else
    echo "qemu driver not active"
    echo "Ensure QEMU is installed and Incus is properly initialized with VM support."
    exit 1
fi

echo "-----------------------------"
echo "All checks passed. Incus is ready to run virtual machines."
