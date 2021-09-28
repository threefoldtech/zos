MSGBUSD_VERSION="0.1.0"
MSGBUSD_CHECKSUM="99748f1d44f2a2e9ca432bcdd0afabe6"
MSGBUSD_LINK="https://github.com/threefoldtech/rmb-go/releases/download/v0.1.0/msgbusd"
MSGBUSD_RUST_TOOLCHAIN="1.50.0"

download_msgbusd() {
    echo "download msgbusd"
    download_file ${MSGBUSD_LINK} ${MSGBUSD_CHECKSUM} msgbusd
}

prepare_msgbusd() {
    echo "[+] prepare msgbusd"
    github_name "msgbusd-${MSGBUSD_VERSION}"
}

install_msgbusd() {
    echo "[+] install msgbusd"

    mkdir -p "${ROOTDIR}/bin"

    cp ${DISTDIR}/msgbusd ${ROOTDIR}/bin/
    chmod +x ${ROOTDIR}/bin/*
}

build_msgbusd() {
    pushd "${DISTDIR}"

    download_msgbusd
    popd

    prepare_msgbusd
    install_msgbusd

}
