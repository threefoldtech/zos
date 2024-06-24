#!/bin/bash
set -e

CHV_VERSION="v39.0"
CHV_URL="https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/${CHV_VERSION}/cloud-hypervisor"
RUSTUP_URL="https://sh.rustup.rs"
VIRTIOFSD_REPO="https://gitlab.com/muhamad.azmy/virtiofsd.git"
RFS_VERSION="v1.1.1"
RFS_URL="https://github.com/threefoldtech/rfs/releases/download/${RFS_VERSION}/rfs"

install_chv() {
    echo "Installing cloud-hypervisor ${CHV_VERSION} ..."
    wget -q ${CHV_URL} -O /usr/local/bin/cloud-hypervisor
    chmod +x /usr/local/bin/cloud-hypervisor
}

install_rust() {
    echo "Installing rust ..."
    curl --proto '=https' --tlsv1.2 -sSf ${RUSTUP_URL} | sh -s -- -y
    . "$HOME/.cargo/env"
}

install_virtiofsd() {
    echo "Installing virtiofsd ..."
    [ ! -d 'virtiofsd' ] && git clone ${VIRTIOFSD_REPO}
    pushd virtiofsd

    # specially needed for virtiofsd bin    
    sudo apt -y update
    sudo apt -y install libseccomp-dev libcap-ng-dev

    cargo build --release
    sudo mv ./target/release/virtiofsd /usr/local/bin/virtiofsd
    popd
}

install_rfs() {
    echo "Installing rfs ${RFS_VERSION} ..."
    wget -q ${RFS_URL}
    chmod +x rfs
    sudo mv ./rfs /usr/local/bin/rfs1
}

main() {
    # must run as superuser
    if [ $(id -u) != "0" ]; then
    echo "You must be the superuser to run this script" >&2
    exit 1
    fi

    TEMP_DIR=$(mktemp -d)
    pushd $TEMP_DIR

    # install cloud-hypervisor
    if ! command -v cloud-hypervisor &>/dev/null; then
        install_chv
    fi

    # install virtiofsd
    if ! command -v virtiofsd &>/dev/null; then
        # install rustup and cargo
        if ! command -v cargo &>/dev/null; then
            install_rust
        fi
        
        install_virtiofsd
    fi

    # install rfs
    if ! command -v rfs1 &>/dev/null; then
        install_rfs
    fi

    # install mcopy/mkdosfs needed to create cidata image. comment if you a have the image
    sudo apt -y install dosfstools mtools

    # install screen for managing multiple servers
    # NOTE: rust bins like virtiofsd miss the logs, runs on a screen session to workaround that
    sudo apt -y install screen

    popd
    rm -rf $TEMP_DIR
}

main