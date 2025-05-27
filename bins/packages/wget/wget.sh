WGET_VERSION="1.24.5"
WGET_CHECKSUM="271bf949384d0858c2c3d419f6311365"
WGET_LINK="https://ftp.gnu.org/gnu/wget/wget-${WGET_VERSION}.tar.gz"

download_wget() {
	echo "downloading wget"
	download_file ${WGET_LINK} ${WGET_CHECKSUM} wget-${WGET_VERSION}
}

prepare_wget() {
	echo "[+] prepare wget"
	github_name "wget-${WGET_VERSION}"
}

install_wget() {
	echo "[+] install wget"

	mkdir -p "${ROOTDIR}/usr/bin"

	cp -f ${DISTDIR}/wget-${WGET_VERSION} wget
	chmod +x ${ROOTDIR}/usr/bin/*
}

build_wget() {
	pushd "${DISTDIR}"

	download_wget
	prepare_wget
	install_wget

	popd
}
