LSHW_VERSION="B.02.20"
LSHW_CHECKSUM="5805eba5f31886582fff673c5dccdb3b"
LSHW_LINK="https://github.com/lyonel/lshw/archive/refs/tags/${LSHW_VERSION}.tar.gz"

download_lshw() {
    download_file ${LSHW_LINK} ${LSHW_CHECKSUM} lshw-${LSHW_VERSION}.tar.gz
}

dependencies_lshw() {
    apt-get install -y build-essential gcc g++ make
}

extract_lshw() {
    if [ ! -d "lshw-${LSHW_VERSION}" ]; then
        echo "[+] extracting: lshw-${LSHW_VERSION}"
        tar -xf ${DISTDIR}/lshw-${LSHW_VERSION}.tar.gz -C ${WORKDIR} 
    fi
}

prepare_lshw() {
    echo "[+] configuring lshw"
    github_name "lshw-${LSHW_VERSION}"
}

compile_lshw() {
    pushd src
    # Build static binary
    make static
    popd
}

install_lshw() {
    echo "[+] installing lshw to initramfs"
    
    mkdir -p "${ROOTDIR}/usr/sbin"
    mkdir -p "${ROOTDIR}/usr/share/man/man1"
    mkdir -p "${ROOTDIR}/usr/share/lshw"
    
    # Install the static binary
    cp src/lshw-static "${ROOTDIR}/usr/sbin/lshw"
    chmod +x "${ROOTDIR}/usr/sbin/lshw"
    
    cp src/lshw.1 "${ROOTDIR}/usr/share/man/man1/"
    cp src/pci.ids src/usb.ids src/oui.txt src/manuf.txt src/pnp.ids src/pnpid.txt "${ROOTDIR}/usr/share/lshw/"
}

build_lshw() {
    dependencies_lshw

    pushd "${DISTDIR}"
    download_lshw
    popd

    pushd "${WORKDIR}"
    extract_lshw
    
    pushd "lshw-${LSHW_VERSION}"
    prepare_lshw
    compile_lshw
    install_lshw
    popd

    popd
}
