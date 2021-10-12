ZUFS_VERSION="0.2.2"
ZUFS_CHECKSUM="d30d3cc6773f9fbfc538eb4e6560e1ec"
ZUFS_LINK="https://github.com/threefoldtech/rfs/releases/download/v${ZUFS_VERSION}/rfs"

download_zufs() {
    download_file ${ZUFS_LINK} ${ZUFS_CHECKSUM} rfs-${ZUFS_VERSION}
}


prepare_zufs() {
    echo "[+] prepare 0-fs"
    github_name "0-fs-${ZUFS_VERSION}"
}

install_zufs() {
    echo "[+] install 0-fs"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av rfs-${ZUFS_VERSION} "${ROOTDIR}/sbin/g8ufs"
}

build_zufs() {
    pushd "${DISTDIR}"

    download_zufs
    prepare_zufs
    install_zufs

    popd
}
