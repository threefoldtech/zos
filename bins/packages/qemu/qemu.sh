QEMU_VERSION="4.2.0"
QEMU_CHECKSUM="278eeb294e4b497e79af7a57e660cb9a"
QEMU_LINK="https://download.qemu.org/qemu-${QEMU_VERSION}.tar.xz"

dependencies_qemu() {
    apt-get install -y build-essential python3 pkg-config git libglib2.0-dev libfdt-dev libpixman-1-dev zlib1g-dev
}

download_qemu() {
    download_file ${QEMU_LINK} ${QEMU_CHECKSUM}
}

extract_qemu() {
    tar -xvJf ${DISTDIR}/qemu-${QEMU_VERSION}.tar.xz -C ${WORKDIR}
}

prepare_qemu() {
    echo "[+] prepare qemu"
    github_name "qemu-${QEMU_VERSION}"

    ./configure --target-list="x86_64-softmmu" --enable-kvm --disable-gtk --disable-sdl --prefix=/usr
}

compile_qemu() {
    echo "[+] compile qemu"
    make ${MAKEOPTS}
}

install_qemu() {
    echo "[+] install qemu"

    make DESTDIR=${ROOTDIR} install

    rm -rf ${ROOTDIR}/usr/share/icons
    rm -rf ${ROOTDIR}/usr/share/applications
}

build_qemu() {
    pushd "${DISTDIR}"

    dependencies_qemu
    download_qemu
    extract_qemu

    popd
    pushd ${WORKDIR}/qemu-${QEMU_VERSION}

    prepare_qemu
    compile_qemu
    install_qemu

    popd
}