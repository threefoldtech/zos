#!/bin/bash

# install dependencies for building
apt-get update

# toolchain dependencies
deps=(pkg-config make m4 autoconf libseccomp-dev)

# system tools and libs
deps+=(curl wget unzip)

# storage and filesystem
deps+=(btrfs-tools)

apt-get install -y ${deps[@]}

# install go
## curl -L https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz > /tmp/go1.13.1.linux-amd64.tar.gz
## tar -C /usr/local -xzf /tmp/go1.13.1.linux-amd64.tar.gz
## mkdir -p /gopath

# install rust
## curl https://sh.rustup.rs -sSf | sh -s -- -y
