set -x

FLISTFILE=/tmp/flist.name
INFOFILE=/tmp/flist.info

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

function param() {
    local KEY=$1
    local KEY=$(echo $KEY=)
    for param in $(strings /proc/cmdline); do
        if [[ "${param:0:${#KEY}}" == "${KEY}" ]]
        then
            echo ${param#${KEY}}
            return 0
        fi
    done

    return 1
}

function default_param() {
    local KEY=$1
    local DEFAULT=$2

    if ! param "${KEY}"; then
        echo ${DEFAULT}
    fi
}

RUNMODE=$(default_param runmode prod)

# set default production flist
FLIST=azmy/zos:production:latest.flist

case "${RUNMODE}" in
    prod)
    ;;
    dev)
        FLIST=azmy/zos:development:latest.flist
    ;;
    test)
        FLIST=azmy/zos:testing:latest.flist
    ;;
    *)
        echo "Invalid run mode '${RUNMODE}'. fall back to production"
    ;;
esac

# track which flist used for booting
echo ${FLIST} > ${FLISTFILE}
chmod 0400 ${FLISTFILE}

BOOTFLIST=https://hub.grid.tf/${FLIST}
BOOTFLISTINFO=https://hub.grid.tf/api/flist/${FLIST}/light

echo "Bootstraping with: ${BOOTFLIST}"
retry wget -O ${INFOFILE} ${BOOTFLISTINFO}
chmod 0400 ${INFOFILE}

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
### filesystem now has all the binaries

for file in $(ls etc/zinit/*.yaml); do
    file=$(basename ${file})
    name="${file%.*}"
    zinit monitor ${name}
done

cd /tmp
umount -fl ${BS}/root
umount -fl ${BS}
