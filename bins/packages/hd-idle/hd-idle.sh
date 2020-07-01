HD_IDLE_VERSION="d6667dec13349c207a63d74caf9e29b9def3d77b"
HD_IDLE_REPOSITORY="github.com/adelolmo/hd-idle"

dependencies_hd-idle() {
    . ${PKGDIR}/../golang/golang.sh
    build_golang
    HD_IDLE_HOME="${GOPATH}/src/github.com/adelolmo"
}

download_hd-idle() {
    echo "[+] downloading and installing hd-idle"
    go get ${HD_IDLE_REPOSITORY}@${HD_IDLE_VERSION}
}

prepare_hd-idle() {
    echo "[+] prepare hd-idle"
    github_name "hd-idle-${HD_IDLE_VERSION}"
}

install_hd-idle() {
    echo "[+] copying hd-idle"

    mkdir -p "${ROOTDIR}/sbin"
    cp -v "${GOPATH}/bin/hd-idle" "${ROOTDIR}/sbin/"
}

build_hd-idle() {
    pushd "${DISTDIR}"

    dependencies_hd-idle
    download_hd-idle

    popd
    pushd ${HD_IDLE_HOME}/hd-idle

    prepare_hd-idle
    install_hd-idle

    popd
}

