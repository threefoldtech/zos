#!/bin/bash
set -x
socket="/tmp/virtiofs.sock"

rootfs=""
kernel="$rootfs/boot/vmlinuz"
initram="$rootfs/boot/initrd.img"
cmdline="rw console=ttyS0 reboot=k panic=1 root=vroot rootfstype=virtiofs rootdelay=30"
overlayfs="/tmp/merged"
flist=""

user="user"
pass="pass"
name="cloud"

fspid=0
flpid=0

fail() {
    echo "$1" >&2
    exit 1
}

usage() {
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo "   --rootfs  Path to the root filesystem image (required)"
    echo "   --user     Username for the system (optional)"
    echo "   --pass     Password for the user (optional)"
    echo "   --name     Hostname for the system (optional)"
    echo "   --init     Entrypoint for the system (optional)"
    echo ""
    exit 1
}

handle_options() {

    while [[ "$#" -gt 0 ]]; do
        case $1 in
        --rootfs)
            rootfs="$2"
            kernel="$rootfs/boot/vmlinuz"
            initram="$rootfs/boot/initrd.img"
            shift
            ;;
        --flist)
            flist="$2"
            shift
            ;;
        --cidata)
            cidata="$2"
            shift
            ;;
        --init)
            if [ ! -z "$2" ]; then
                cmdline="$cmdline init=$2"
            fi
            shift
            ;;
        
        --user)
            user="$2"
            shift
            ;;
        --pass)
            pass="$2"
            shift
            ;;
        --name)
            name="$2"
            shift
            ;;
        
        *)
            usage
            ;;
        esac
        shift
    done

}

validate() {
    # if no rootfs or metadata, fail
    # check other binaries
    which virtiofsd &>/dev/null || fail "virtiofsd not found in PATH"
    which cloud-hypervisor &>/dev/null || fail "cloud-hypervior not found in path"

    if [ ! -f "${kernel}" ]; then
        fail "kernel file not found"
    fi

    if [ ! -f "${initram}" ]; then
        fail "kernel file not found"
    fi
}

cidata="/tmp/cidata.img"
create_cidata() {
    rm -f "${cidata}"
    mkdosfs -n CIDATA -C "${cidata}" 8192

    cat >user-data <<EOF
#cloud-config
users:
  - name: $user
    plain_text_passwd: $pass
    shell: /bin/bash
    lock_passwd: false
    sudo: ALL=(ALL) NOPASSWD:ALL
EOF
    mcopy -oi "${cidata}" -s user-data ::
    rm -f user-data

    cat >meta-data <<EOF
instance-id: $name
local-hostname: $name
EOF
    mcopy -oi "${cidata}" -s meta-data ::
    rm -f meta-data
}

start_fs() {
    sudo virtiofsd \
        --socket-path="${socket}" \
        --shared-dir="${overlayfs}" \
        --cache=never \
        &

    fspid=$!
}

cleanup() {
    sudo umount "$overlayfs"
    kill "$flpid" &>/dev/null
    kill "$fspid" &>/dev/null || true
}
trap cleanup EXIT

run_hypervisor() {
    sudo cloud-hypervisor \
        --cpus boot=1,max=1 \
        --memory size=1024M,shared=on \
        --kernel "${kernel}" \
        --initramfs "${initram}" \
        --fs tag=vroot,socket="${socket}" \
        --disk path="${cidata}" \
        --cmdline "${cmdline}" \
        --serial tty \
        --console off
}

create_rwlayer() {
    sudo mkdir -p /tmp/upper /tmp/workdir "$overlayfs"

    sudo mount \
        -t overlay \
        -o lowerdir="$rootfs",upperdir=/tmp/upper,workdir=/tmp/workdir \
        none \
        "$overlayfs"
}

mount_flist() {
    wget "$flist" -O /tmp/flist.flist

    rootfs=/home/omar/tmp
    mkdir -p "$rootfs"
    rfs1 --meta /tmp/flist.flist "$rootfs" &

    flpid=$!
}

# decompress_kernel() {
#     wget -O /tmp/extract-vmlinux https://raw.githubusercontent.com/torvalds/linux/master/scripts/extract-vmlinux
#     chmod +x /tmp/extract-vmlinux
    
# }

prepare_cloud_image() {
    # chroot and install ci
    # symlink to host vmlinuz/initrd
    # load virtiofs and update ramfs
}

boot() {
    start_fs
    run_hypervisor
}

handle_options "$@"

validate

# if no cidata
create_cidata

# if is metadata
mount_flist

create_rwlayer

boot
