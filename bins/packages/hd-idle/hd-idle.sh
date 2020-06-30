HDIDLE_VERSION="master"
HDIDLE_REPOSITORY="https://github.com/adelolmo/hd-idle.git"

dependencies_hd-idle() {
    . ${PKGDIR}/../golang/golang.sh
    build_golang

    HD_IDLE_HOME="${GOPATH}/src/github.com/adelolmo"
}

download_hd-idle() {
    download_git ${HDIDLE_REPOSITORY} ${HDIDLE_VERSION}
}

extract_hd-idle() {
    event "refreshing" "hd-idle-${HDIDLE_VERSION}"
    mkdir -p ${HD_IDLE_HOME}
    rm -rf ${HD_IDLE_HOME}/hd-idle
    cp -a ${DISTDIR}/hd-idle ${HD_IDLE_HOME}/
}

prepare_hd-idle() {
    echo "[+] prepare hd-idle"
    github_name "hd-idle-${HDIDLE_VERSION}"
}

compile_hd-idle() {
    echo "[+] compiling hd-idle"
    dpkg-buildpackage -a armhf -us -uc -b
}

install_hd-idle() {
    echo "[+] install hd-idle"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av hd-idle "${ROOTDIR}/sbin/"
}

build_hd-idle() {
    pushd "${DISTDIR}"

    dependencies_hd-idle
    download_hd-idle
    extract_hd-idle

    popd
    pushd ${HD_IDLE_HOME}/hd-idle

    prepare_hd-idle
    compile_hd-idle
    install_hd-idle

    popd
}

