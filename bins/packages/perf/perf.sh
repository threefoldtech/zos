KERNEL_VERSION="v5.10"
KERNEL_LINK="https://github.com/torvalds/linux.git"

download_kernel() {
    echo "downloading kernel"
    download_git ${KERNEL_LINK} ${KERNEL_VERSION} ${KERNEL_VERSION}
}


prepare_perf() {
    echo "[+] prepare perf"
    # only build-essential, bison and flex are needed the rest is for extra perf features.
    apt update && \
    apt install -y \
    build-essential \
    libdwarf-dev \
    libdw-dev \
    binutils-dev \
    libcap-dev \
    libelf-dev \
    libnuma-dev \
    python-setuptools \
    python3 \
    python3-dev \
    libssl-dev \
    libunwind-dev \
    libdwarf-dev  \
    zlib1g-dev \
    liblzma-dev \
    libaio-dev \
    flex \
    bison        
}

install_perf() {
    echo "[+] install perf"

    pushd linux/tools/perf
    make LDFLAGS=-static
    popd

    mkdir -p "${ROOTDIR}/usr/bin"

    cp ${DISTDIR}/linux/tools/perf/perf ${ROOTDIR}/usr/bin/perf
}

build_perf() {
    pushd "${DISTDIR}"

    download_kernel
    prepare_perf
    install_perf

    popd

}
