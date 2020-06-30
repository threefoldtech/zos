HDIDLE_VERSION="master"
HDIDLE_REPOSITORY="https://github.com/adelolmo/hd-idle.git"

dependencies_hd_idle() {
    . ${PKGDIR}/../golang/golang.sh
    build_golang

    HD_IDLE_HOME="${GOPATH}/src/github.com/adelolmo"
}

download_hd_idle() {
    download_git ${HDIDLE_REPOSITORY} ${HDIDLE_VERSION}
}

extract_hd_idle() {
    event "refreshing" "hd-idle-${HDIDLE_VERSION}"
    mkdir -p ${HD_IDLE_HOME}
    rm -rf ${HD_IDLE_HOME}/hd-idle
    cp -a ${DISTDIR}/hd-idle ${HD_IDLE_HOME}/
}

prepare_hd_idle() {
    echo "[+] prepare hd-idle"
    github_name "hd-idle-${HDIDLE_VERSION}"
}

compile_hd_idle() {
    echo "[+] compiling hd-idle"
    dpkg-buildpackage -a armhf -us -uc -b
}

install_hd_idle() {
    echo "[+] install hd-idle"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av hd-idle "${ROOTDIR}/sbin/"
}

build_hd_idle() {
    pushd "${DISTDIR}"

    dependencies_hd_idle
    download_hd_idle
    extract_hd_idle

    popd
    pushd ${HD_IDLE_HOME}/hd-idle

    prepare_hd_idle
    compile_hd_idle
    install_hd_idle

    popd
}

