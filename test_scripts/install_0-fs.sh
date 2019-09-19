#/bin/bash

set -ex

wget https://github.com/threefoldtech/0-fs/releases/download/v2.0.1/g8ufs-v2.0.1-linux-amd64.tar.gz -O /tmp/g8ufs-v2.0.1-linux-amd64.tar.gz
cd /tmp
tar xvf g8ufs-v2.0.1-linux-amd64.tar.gz
sudo cp g8ufs /bin/g8ufs