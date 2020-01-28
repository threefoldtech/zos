CONTAINERD_REPOSITORY="https://github.com/containerd/containerd"
CONTAINERD_BRANCH="v1.2.7"
CONTAINERD_HOME="${GOPATH}/src/github.com/containerd"

download_containerd() {
    download_git ${CONTAINERD_REPOSITORY} ${CONTAINERD_BRANCH}
}

extract_containerd() {
    event "refreshing" "containerd-${CONTAINERD_BRANCH}"
    mkdir -p ${CONTAINERD_HOME}
    rm -rf ${CONTAINERD_HOME}/containerd
    cp -a ${WORKDIR}/containerd ${CONTAINERD_HOME}/
}

prepare_containerd() {
    echo "[+] prepare containerd"
}

compile_containerd() {
    echo "[+] compiling containerd"
    make CGO_CFLAGS=-I${ROOTDIR}/usr/include
}

install_containerd() {
    echo "[+] copying binaries"

    mkdir -p "${ROOTDIR}/containerd"
    pushd "${ROOTDIR}/containerd"

    mkdir -p usr/bin
    mkdir -p etc/containerd
    mkdir -p etc/zinit

    cp -av ${CONTAINERD_HOME}/containerd/bin/* "usr/bin/"
}

build_containerd() {
    pushd "${WORKDIR}"

    download_containerd
    extract_containerd

    popd
    pushd ${CONTAINERD_HOME}/containerd

    prepare_containerd
    compile_containerd
    install_containerd

    popd
}

