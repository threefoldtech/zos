CONTAINERD_VERSION="ae2f3fdfd1a435fe83fb083e4db9fa63a9e0a13e"
CONTAINERD_CHECKSUM="73e6d4bad082e69e530c8ab5a9ee5118"
CONTAINERD_LINK="https://github.com/containerd/containerd/archive/${CONTAINERD_VERSION}.tar.gz"

dependencies_containerd() {
    apt-get install -y btrfs-progs libbtrfs-dev libseccomp-dev build-essential pkg-config

    if [ -z $GOPATH ]; then
        if command -v go > /dev/null; then
            export GOPATH=$(go env GOPATH)
        else
            curl -L https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz > /tmp/go1.13.1.linux-amd64.tar.gz
            tar -C /usr/local -xzf /tmp/go1.13.1.linux-amd64.tar.gz
            mkdir -p /gopath

            export PATH=$PATH:/usr/local/go/bin
            export GOPATH=/gopath
        fi
    fi

    CONTAINERD_HOME="${GOPATH}/src/github.com/containerd"
}

download_containerd() {
    download_file ${CONTAINERD_LINK} ${CONTAINERD_CHECKSUM} containerd-${CONTAINERD_VERSION}.tar.gz
}

extract_containerd() {
    mkdir -p ${CONTAINERD_HOME}
    rm -rf ${CONTAINERD_HOME}/containerd

    pushd ${CONTAINERD_HOME}

    echo "[+] extracting: containerd-${CONTAINERD_VERSION}"
    tar -xf ${DISTDIR}/containerd-${CONTAINERD_VERSION}.tar.gz -C .
    mv containerd-${CONTAINERD_VERSION} containerd

    popd
}

prepare_containerd() {
    echo "[+] prepare containerd"
    github_name "containerd-${CONTAINERD_VERSION}"
}

compile_containerd() {
    echo "[+] compiling containerd"
    make CGO_CFLAGS=-I${ROOTDIR}/usr/include
}

install_containerd() {
    echo "[+] install containerd"

    mkdir -p "${ROOTDIR}/usr/bin"
    mkdir -p "${ROOTDIR}/etc/containerd"
    mkdir -p "${ROOTDIR}/etc/zinit"

    cp -av bin/* "${ROOTDIR}/usr/bin/"
    cp -av ${FILESDIR}/config.toml "${ROOTDIR}/etc/containerd/"
    cp -av ${FILESDIR}/containerd.yaml "${ROOTDIR}/etc/zinit/"
}

build_containerd() {
    pushd "${DISTDIR}"

    dependencies_containerd
    download_containerd
    extract_containerd

    popd
    pushd ${CONTAINERD_HOME}/containerd

    prepare_containerd
    compile_containerd
    install_containerd

    popd
}
