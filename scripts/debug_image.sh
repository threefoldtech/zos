#!/bin/bash
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
}

declare -A options
handle_options() {

    while [[ "$#" -gt 0 ]]; do
        case $1 in
        --image|--cidata|--kernel|--initramfs|--init|--user|--pass|--name)
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
    for tool in virtiofsd cloud-hypervisor mkdosfs rfs1; do
        which "$tool" &>/dev/null || fail "'$tool' not found in PATH"
    done

    [ -z "${options[--image]}" ] && fail "rootfs image not provided"
}

cidata="/tmp/cidata.img"
create_cidata() {
    [[ -f "${options[--cidata]}" ]] && return
    
    echo "Creating cidata"
    # todo: cloud-init clean

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

    # unmount and remove overlayfs
    sudo umount "$overlayfs" 2>/dev/null || true
    sudo rm -rf /tmp/upper /tmp/workdir "$overlayfs" 2>/dev/null || true

    # kill any running processes
    for pid in "${pids[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
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

prepare_rootfs() {
    local image_path="${options[--image]}"

    sudo mkdir -p /tmp/upper /tmp/workdir "$overlayfs"

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
        "$overlayfs"

    echo "Starting virtiofs"
    sudo virtiofsd \
        --socket-path="${socket}" \
        --shared-dir="${overlayfs}" \
        --cache=never \
        &
    fspid=$!
}

prepare_boot() {
    kernel="${options[--kernel]}"
    if [ -z "$kernel" ]; then
        kernel="${overlayfs}/boot/vmlinuz"
    fi

    initramfs="${options[--initramfs]}"
    if [ -z "$initramfs" ]; then
        initramfs="${overlayfs}/boot/initrd.img"
    fi

    init="${options[--init]}"
    if [ ! -z "$init" ]; then
        cmdline="$cmdline init=$init"
    fi

    if [ ! -f "$kernel" ] | [ ! -f "$initramfs" ]; then
        fail "kernel or initramfs not found"
        # in case no kernel or initramfs, it is a container image
        # chroot and install ci and add symlink to host vmlinuz/initrd and update initramfs
    fi
}

handle_options "$@"
validate
prepare_rootfs
prepare_boot
create_cidata

# run_hypervisor