#!/bin/bash

systemctl stop systemd-resolved || true
systemctl disable systemd-resolved || true
apt remove -y systemd-resolved
rm -f /etc/resolv.conf
echo "nameserver 8.8.8.8" >/etc/resolv.conf
echo "nameserver 1.1.1.1" >>/etc/resolv.conf
chmod 0644 /etc/resolv.conf
cat /etc/resolv.conf
sleep 5
