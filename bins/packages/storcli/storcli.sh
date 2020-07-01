STORCLI_VERSION="1.21.06"

dependencies_storcli() {
    apt-get update

    echo "[+] updating soruces list"
    wget -O - http://archive.thomas-krenn.com/tk-archive.gpg.pub | apt-key add -

    wget -O /etc/apt/sources.list.d/tk-main-xenial.list http://archive.thomas-krenn.com/tk-main-xenial.list
    wget -O /etc/apt/sources.list.d/tk-optional-xenial.list http://archive.thomas-krenn.com/tk-optional-xenial.list
    cd
    apt-get update

    echo "[+] installing storcli"
    apt-get install storcli=${STORCLI_VERSION}
}

install_storcli() {
    echo "[+] copying storcli"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av "/usr/sbin/storcli" "${ROOTDIR}/sbin/"
}

prepare_storcli() {
    echo "[+] prepare storcli"
    github_name "storcli-${STORCLI_VERSION}"
}

build_storcli() {
    pushd "${DISTDIR}"

    dependencies_storcli
    prepare_storcli
    install_storcli

    popd
}

