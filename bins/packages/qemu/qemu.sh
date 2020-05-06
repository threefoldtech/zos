QEMU_VERSION="4.2.0"
QEMU_LINK="https://download.qemu.org/qemu-${QEMU_VERSION}.tar.xz"

download_qemu() {
    download_file ${QEMU_LINK}
}

extract_qemu() {
    tar -xvJf ${DISTDIR}/qemu-${QEMU_VERSION}.tar.xz -C ${WORKDIR}
}

prepare_qemu() {
    echo "[+] prepare qemu"
    github_name "qemu"
}

compile_qemu() {
    echo "[+] compile qemu"
    pushd ${WORKDIR}
    ./configure --target-list="x86_64-softmmu" --enable-kvm --disable-gtk --disable-sdl
    make
    make install
    popd
}

install_qemu() {
    echo "[+] install qemu"

    mkdir -p "${ROOTDIR}/usr/bin"


    cp ${WORKDIR}/qemu-${QEMU_VERSION} ${ROOTDIR}/usr/bin/qemu

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_qemu() {
    pushd "${DISTDIR}"

    download_qemu
    extract_qemu

    popd
    pushd ${WORKDIR}

    prepare_qemu
    compile_qemu
    install_qemu

    popd
}