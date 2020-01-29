FIRECRACKER_VERSION="0.20.0"
FIRECRACKER_CHECKSUM="d5a823547543ba9e8a1ccbf87f8c74c1"
FIRECRACKER_LINK="https://github.com/firecracker-microvm/firecracker/releases/download/v${FIRECRACKER_VERSION}/firecracker-v${FIRECRACKER_VERSION}-x86_64"
JAILER_VERSION="0.20.0"
JAILER_CHECKSUM="3dfd76ac6209da4fc1f76d06f97972fb"
JAILER_LINK="https://github.com/firecracker-microvm/firecracker/releases/download/v${FIRECRACKER_VERSION}/jailer-v${FIRECRACKER_VERSION}-x86_64"

download_firecracker() {
    download_file ${FIRECRACKER_LINK} ${FIRECRACKER_CHECKSUM}
    download_file ${JAILER_LINK} ${JAILER_CHECKSUM}
}

extract_firecracker() {
    cp firecracker-v${FIRECRACKER_VERSION}-x86_64 ${WORKDIR}/
    cp jailer-v${FIRECRACKER_VERSION}-x86_64  ${WORKDIR}/
}

prepare_firecracker() {
    echo "[+] prepare firecracker"
}

compile_firecracker() {
    echo "[+] compile firecracker"
}

install_firecracker() {
    echo "[+] install firecracker"

    mkdir -p "${ROOTDIR}/usr/bin"
    cp ${WORKDIR}/firecracker-v${FIRECRACKER_VERSION}-x86_64 ${ROOTDIR}/usr/bin/firecracker
    cp ${WORKDIR}/jailer-v${FIRECRACKER_VERSION}-x86_64 ${ROOTDIR}/usr/bin/jailer

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_firecracker() {
    pushd "${DISTDIR}"

    download_firecracker
    extract_firecracker

    popd
    pushd ${WORKDIR}

    prepare_firecracker
    compile_firecracker
    install_firecracker

    popd
}

