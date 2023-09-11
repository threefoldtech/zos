CPU_BENCHMARK_VERSION="v0.1"
CPU_BENCHMARK_CHECKSUM="25891eb15ec0b1bb8d745a8af3907895"
CPU_BENCHMARK_LINK="https://github.com/threefoldtech/cpu-benchmark-simple/releases/download/${CPU_BENCHMARK_VERSION}/grid-cpubench-simple-0.1-linux-amd64-static"

download_cpu-benchmark() {
    echo "downloading cpu-benchmark"
    download_file ${CPU_BENCHMARK_LINK} ${CPU_BENCHMARK_CHECKSUM} cpu-benchmark
}


prepare_cpu-benchmark() {
    echo "[+] prepare cpu-benchmark"
    github_name "cpu-benchmark-${CPU_BENCHMARK_VERSION}"
}

install_cpu-benchmark() {
    echo "[+] install cpu-benchmark"
    mkdir -p "${ROOTDIR}/usr/bin"

    cp ${DISTDIR}/cpu-benchmark ${ROOTDIR}/usr/bin/cpu-benchmark
}

build_cpu-benchmark() {
    pushd "${DISTDIR}"

    download_cpu-benchmark
    prepare_cpu-benchmark
    install_cpu-benchmark

    popd
}
