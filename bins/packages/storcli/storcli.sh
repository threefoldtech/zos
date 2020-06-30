dependencies_storcli() {
    apt-get update
    apt-get install wget

    echo "[+] updating soruces list"
    wget -O - http://archive.thomas-krenn.com/tk-archive.gpg.pub | apt-key add -
    cd /etc/apt/sources.list.d
    wget http://archive.thomas-krenn.com/tk-main-xenial.list
    wget http://archive.thomas-krenn.com/tk-optional-xenial.list
    cd
    apt-get update

    echo "[+] installing storcli"
    apt-get install storcli
}

install_storcli() {
    echo "[+] copying storcli"

    mkdir -p "${ROOTDIR}/sbin"
    cp -av "/usr/sbin/storcli" "${ROOTDIR}/sbin/"
}

build_storcli() {
    pushd "${DISTDIR}"

    dependencies_storcli
    install_storcli

    popd
}

