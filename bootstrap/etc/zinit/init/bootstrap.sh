set -x

# TODO: change BOOTFLIST to release flist.
# In development, may be we also can specify the build branch from kernel params
# and use it to construct the full flist name.
BOOTFLIST=https://hub.grid.tf/azmy/zos.flist

echo "bootstraping"

# helper retry function
# the retry function never give up because the 
# bootstrap must succeed. otherwise the node 
# will not be functional. So no reason to give
# up
function retry() {
    until $@; do
        sleep 1s
        echo "retrying: $@"
    done
}

BS=/tmp/bootstrap
mkdir -p ${BS}

## Prepare a tmpfs for 0-fs cache
mount -t tmpfs -o size=512M tmpfs ${BS}

cd ${BS}
mkdir -p root
retry wget -O machine.flist ${BOOTFLIST}

g8ufs --backend ${BS}/backend --meta machine.flist root &

retry mountpoint root

## move to root
cd root
cp -a * /
### filesystem is ready

for file in $(ls etc/zinit/*.yaml); do 
    file=$(basename ${file})
    name="${file%.*}"
    zinit monitor ${name}
done

cd /tmp
umount -fl ${BS}/root
umount -fl ${BS}
