#!/bin/bash

modprobe fuse
modprobe btrfs
modprobe tun
modprobe br_netfilter

echo never > /sys/kernel/mm/transparent_hugepage/enabled

ulimit -n 524288