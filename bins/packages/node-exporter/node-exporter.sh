EXPORTER_VERSION="1.1.2"
EXPORTER_CHECKSUM="61e2b963f66f1e00649c8e4d1f4729e5"
EXPORTER_NAME="node_exporter-${EXPORTER_VERSION}.linux-amd64"
EXPORTER_LINK="https://github.com/prometheus/node_exporter/releases/download/v${EXPORTER_VERSION}/${EXPORTER_NAME}.tar.gz"


download_exporter() {
    download_file ${EXPORTER_LINK} ${EXPORTER_CHECKSUM}
}

extract_exporter() {
    tar -xf ${DISTDIR}/${EXPORTER_NAME}.tar.gz -C ${WORKDIR}
}

prepare_exporter() {
    echo "[+] prepare exporter"
    github_name "exporter-${EXPORTER_VERSION}"
}

compile_promtail() {
    echo "[+] compile exporter"
}

install_promtail() {
    echo "[+] install exporter"

    mkdir -p "${ROOTDIR}/usr/bin"
    mkdir -p "${ROOTDIR}/etc/zinit"

    cp ${WORKDIR}/${EXPORTER_NAME}/node_exporter ${ROOTDIR}/usr/bin/node_exporter

    cp ${FILESDIR}/node-exporter.yaml ${ROOTDIR}/etc/zinit/node-exporter.yaml

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_node-exporter() {
    pushd "${DISTDIR}"

    download_exporter
    extract_exporter

    popd
    pushd ${WORKDIR}

    prepare_exporter
    compile_promtail
    install_promtail

    popd
}
