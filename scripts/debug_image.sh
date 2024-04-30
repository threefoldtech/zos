#!/bin/bash

socket="/tmp/virtiofs.sock"

rootfs=""
kernel="$rootfs/boot/vmlinuz"
initram="$rootfs/boot/initrd.img"

user="user"
pass="pass"
name="cloud"
init="/sbin/init"

fspid=0

fail() {
    echo "$1" >&2
    exit 1
}

usage() {
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo "  -r, --rootfs  Path to the root filesystem image (required)"
    echo "  -u, --user     Username for the system (optional)"
    echo "  -p, --pass     Password for the user (optional)"
    echo "  -n, --name     Hostname for the system (optional)"
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

            if [[ -z "$rootfs" ]]; then
                echo "Error: -r (rootfs) flag is required."
                usage
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
        --init)
            init="$2"
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
        --shared-dir="${rootfs}" \
        --cache=never \
        &

    fspid=$!
}

cleanup() {
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
        --cmdline "rw console=ttyS0 reboot=k panic=1 root=vroot rootfstype=virtiofs rootdelay=30 init=${init}" \
        --serial tty \
        --console off
}

handle_options "$@"
validate
create_cidata
start_fs
run_hypervisor
