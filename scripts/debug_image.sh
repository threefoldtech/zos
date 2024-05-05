#!/bin/bash

set -euo pipefail

# constants
readonly SOCKET="/tmp/virtiofs.sock"
readonly OVERLAYFS="/tmp/merged"

# defaults
cidata="/tmp/cidata.img"
user="user"
pass="pass"
name="cloud"
cmdline="rw console=ttyS0 reboot=k panic=1 root=vroot rootfstype=virtiofs rootdelay=30"

# globals
init=""
kernel=""
initramfs=""
fspid=0
flpid=0

fail() {
    echo "Error: $*" >&2
    exit 1
}

usage() {
    echo ""
}

handle_options() { 
    while [[ "$#" -gt 0 ]]; do
        case $1 in
            --image|--cidata|--kernel|--initramfs|--init|--user|--pass|--name|--debug)
                options[$1]="$2"
                shift 2
                ;;
            *)
                usage
                exit 1
                ;;
        esac
    done
}


validate() {
    for tool in virtiofsd cloud-hypervisor rfs1; do
        command -v "$tool" >/dev/null 2>&1 || fail "'$tool' not found in PATH"
    done

    if [ ! -z "${options[--cidata]}" ]; then
        command -v mkdosfs >/dev/null 2>&1 || fail "mkdosfs not found in PATH"
    fi

    if [ -z "${options[--image]}" ]; then
        fail "rootfs image not provided"
    fi
}

create_cidata() {
    sudo chroot "${OVERLAYFS}" cloud-init clean

    if [[ -f "${options[--cidata]}" ]]; then
        cidata="${options[--cidata]}"
        return
    fi

    [ ! -z "${options[--user]}" ] && user="${options[--user]}"
    [ ! -z "${options[--pass]}" ] && pass="${options[--pass]}"
    [ ! -z "${options[--name]}" ] && name="${options[--name]}"
    
    echo "Creating cidata"

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


cleanup() {
    local pids=( "$flpid" "$fspid" )

    sudo umount "${OVERLAYFS}" 2>/dev/null || true
    sudo rm -rf /tmp/upper /tmp/workdir "${OVERLAYFS}" 2>/dev/null || true

    for pid in "${pids[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
}

boot() {
    sudo cloud-hypervisor \
        --cpus boot=1,max=1 \
        --memory size=1024M,shared=on \
        --kernel "${kernel}" \
        --initramfs "${initramfs}" \
        --fs tag=vroot,socket="${SOCKET}" \
        --disk path="${cidata}" \
        --cmdline "${cmdline}" \
        --serial tty \
        --console off
}

prepare_rootfs() {
    local image_path="${options[--image]}"

    sudo mkdir -p /tmp/upper /tmp/workdir "${OVERLAYFS}"

    if [[ ${image_path} == *.flist ]]; then
        echo "${image_path} is flist, mounting"

        rootfs=/tmp/lower
        sudo mkdir -p "$rootfs" && sudo chmod 777 "$rootfs" 

        rfs1 --meta "$image_path" "$rootfs" &
        flpid=$!
    else 
        rootfs="${image_path}"
    fi

    echo "Mounting overlay"
    sudo mount \
        -t overlay \
        -o lowerdir="$rootfs",upperdir=/tmp/upper,workdir=/tmp/workdir \
        none \
        "${OVERLAYFS}"

    echo "Starting virtiofs"
    sudo virtiofsd \
        --socket-path="${SOCKET}" \
        --shared-dir="${OVERLAYFS}" \
        --cache=never \
        &
    fspid=$!
}

prepare_boot() {
    kernel="${options[--kernel]-}"
    kernel="${kernel:-"${OVERLAYFS}/boot/vmlinuz"}"

    initramfs="${options[--initramfs]-}"
    initramfs="${initramfs:-"${OVERLAYFS}/boot/initrd.img"}"

    init="${options[--init]-}"
    if [ -n "$init" ]; then
        cmdline="$cmdline init=$init"
    fi

    if [ ! -f "$kernel" ] | [ ! -f "$initramfs" ]; then
        fail "kernel or initramfs not found"
        # in case no kernel or initramfs, it is a container image
        # chroot and install ci and add symlink to host vmlinuz/initrd and update initramfs
    fi
}

# main 

# move defaults here
declare -A options=(
    [--debug]="false"
    [--cidata]=""
    [--kernel]=""
    [--initramfs]=""
    [--init]=""
    [--user]=""
    [--pass]=""
    [--name]=""
)

handle_options "$@"

if [[ "${options[--debug]}" == true ]]; then
    set -x
fi

echo "validating requirements"
validate

echo "preparing rootfs"
prepare_rootfs

echo "preparing boot"
prepare_boot

echo "creating cidata"
create_cidata
trap cleanup EXIT

echo "booting"
boot