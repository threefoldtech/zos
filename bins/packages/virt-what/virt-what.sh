VIRTWHAT_VERSION="1.20"
VIRTWHAT_CHECKSUM="881bf446bc922eb1c702cdd2302b4459"
VIRTWHAT_LINK="https://people.redhat.com/~rjones/virt-what/files/virt-what-${VIRT-WHAT_VERSION}.tar.gz"

download_virtwhat() {
    download_file ${VIRTWHAT_LINK}
}

extract_virtwhat() {
    tar -xf ${DISTDIR}/virt-what-${VIRTWHAT_VERSION}.tar.gz -C ${WORKDIR}
}

prepare_virtwhat() {
    echo "[+] prepare virt-what"
    github_name "virtwhat-${VIRTWHAT_VERSION}"dis
}

compile_virtwhat() {
    echo "[+] compile virt-what"
    pushd ${WORKDIR}
    ./configure
    make
    popd
}

install_virtwhat() {
    echo "[+] install virt-what"

    mkdir -p "${ROOTDIR}/usr/bin"


    cp ${WORKDIR}/virt-what-${VIRTWHAT_VERSION} ${ROOTDIR}/usr/bin/virt-what

    chmod +x ${ROOTDIR}/usr/bin/*
}

build_virtwhat() {
    pushd "${DISTDIR}"

    download_virtwhat
    extract_virtwhat

    popd
    pushd ${WORKDIR}

    prepare_virtwhat
    compile_virtwhat
    install_virtwhat

    popd
}

