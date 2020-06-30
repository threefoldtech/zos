HDIDLE_VERSION="master"
HDIDLE_REPOSITORY="https://github.com/adelolmo/hd-idle.git"

dependencies_hd-idle() {
    . ${PKGDIR}/../golang/golang.sh
    build_golang
    HD_IDLE_HOME="${GOPATH}/src/github.com/adelolmo"
}

download_hd-idle() {
    echo "[+] downloading and installing hd-idle"
    go get github.com/adelolmo/hd-idle
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

    install_hd-idle

    popd
}

