LVM_VERSION="2_03_22"
LVM_CHECKSUM="012884748396dffce84b67a5e159e272"
LVM_LINK="https://github.com/lvmteam/lvm2/archive/refs/tags/v${LVM_VERSION}.tar.gz"

download_lvm() {
    download_file $LVM_LINK $LVM_CHECKSUM
}

extract_lvm() {
    if [ ! -d "lvm-${LVM_VERSION}" ]; then
        echo "[+] extracting: lvm-${LVM_VERSION}"
        tar -xf ${DISTDIR}/v${LVM_VERSION}.tar.gz -C ${WORKDIR}
    fi
}

prepare_lvm() {
    echo "[+] preparing lvm"
    github_name "lvm-v${LVM_VERSION}"
    ./configure --prefix=/usr --enable-static_link
}

compile_lvm() {
    echo "[+] compiling tpm"
    make ${MAKEOPTS}
}

install_lvm() {
    echo "[+] installing tpm"
    mkdir -p "${ROOTDIR}/bin"
    cp tools/lvm.static "${ROOTDIR}/bin"
}

build_lvm() {
    apt-get install -y \
        build-essential \
        thin-provisioning-tools \
        libaio-dev

    pushd "${DISTDIR}"
    download_lvm
    extract_lvm
    popd

    pushd "${WORKDIR}/lvm2-${LVM_VERSION}"

    export PKG_CONFIG_PATH="${ROOTDIR}/usr/lib/pkgconfig/"
    export CFLAGS="-I${ROOTDIR}/usr/include"
    export LDFLAGS="-L${ROOTDIR}/usr/lib"

    prepare_lvm
    compile_lvm
    install_lvm

    unset PKG_CONFIG_PATH
    unset CFLAGS
    unset LDFLAGS

    popd
}
