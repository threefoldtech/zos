#!/bin/bash
set -ex

apt-get update
apt-get install -y curl git build-essential
apt-get install -y libgflags-dev # libzstd-dev

# install go 1.8 (needed by fuse)
curl https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz > /tmp/go1.12.9.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go1.12.9.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# zos
pushd /zosv2

pushd cmds
make
popd 

pushd bin
# reduce binary size
strip -s *
popd 

mkdir -p /tmp/root/{bin,etc/zinit}
cp -r bin/* /tmp/root/bin
cp -r zinit/* /tmp/root/etc

mkdir -p /tmp/archives/
tar -czf "/tmp/archives/zos.tar.gz" -C /tmp/root .