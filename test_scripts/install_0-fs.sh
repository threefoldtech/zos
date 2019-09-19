#/bin/bash

set -ex

export GO11MODULE=off

go get -v github.com/tools/godep

mkdir -p /home/runner/work/zosv2/0-fs/
cd  /home/runner/work/zosv2/0-fs/
git clone --depth 1 https://github.com/threefoldtech/0-fs.git
cd 0-fs
godep restore
make install