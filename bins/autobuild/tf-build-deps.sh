#!/bin/bash

# install dependencies for building
apt-get update

# toolchain dependencies
deps=(pkg-config make m4 autoconf libseccomp-dev)

# system tools and libs
deps+=(libssl-dev dnsmasq git curl bc wget unzip)

# storage and filesystem
deps+=(e2fslibs-dev libblkid-dev uuid-dev libattr1-dev btrfs-tools)

apt-get install -y ${deps[@]}

# install go
## curl -L https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz > /tmp/go1.13.1.linux-amd64.tar.gz
## tar -C /usr/local -xzf /tmp/go1.13.1.linux-amd64.tar.gz
## mkdir -p /gopath

# install rust
## curl https://sh.rustup.rs -sSf | sh -s -- -y
