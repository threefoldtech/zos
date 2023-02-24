QSFS_VERSION="0.2.0-rc1"
QSFS_LINK="ghcr.io/threefoldtech/qsfs"

download_qsfs() {
    echo "download qsfs"
    download_docker_image ${QSFS_LINK}:${QSFS_VERSION} qsfs-${QSFS_VERSION}.tar
}

prepare_qsfs() {
    echo "[+] prepare qsfs"
    github_name "qsfs-${QSFS_VERSION}"
}

install_qsfs() {
    echo "[+] install qsfs"

    mkdir -p "${ROOTDIR}"

    extract_docker_image "${DISTDIR}/qsfs-${QSFS_VERSION}.tar" "${ROOTDIR}"
}

build_rmb() {
    pushd "${DISTDIR}"

    download_qsfs
    popd

    prepare_qsfs
    install_qsfs
}
