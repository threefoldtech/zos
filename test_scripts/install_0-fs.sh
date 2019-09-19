#/bin/bash

set -ex

mkdir -p /home/runner/work/zosv2/0-fs/
cd  /home/runner/work/zosv2/0-fs/
git clone --depth 1 https://github.com/threefoldtech/0-fs.git
cd 0-fs
make install