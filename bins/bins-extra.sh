#!/bin/bash
set -e

# You need to use absolutes path
WORKDIR="${PWD}/staging"
TMPDIR="${PWD}/tmp"
PKGDIR="${PWD}/packages"
ROOTDIR="${PWD}/releases"

# By default, we compile with (number of cpu threads + 1)
# you can changes this to reduce computer load
JOBS=$(($(grep -c 'bogomips' /proc/cpuinfo) + 1))
MAKEOPTS="-j ${JOBS}"
USE_MIRROR=0

#
# auto loading packages directory
#
packages=0
DOWNLOADERS=()
EXTRACTORS=()

for package in "${PKGDIR}"/*; do
    # loading submodule
    pkgname=$(basename $package)
    # echo "[+] loading package: $pkgname"

    . "${package}/${pkgname}.sh"

    modules=$(($modules + 1))
done

unset package

#
# argument processing
#
OPTS=$(getopt -o p:h --long package:,help -n 'parse-options' -- "$@")
if [ $? != 0 ]; then
    echo "Failed parsing options." >&2
    exit 1
fi

if [ "$OPTS" != " --" ] && [ "$OPTS" != " --release --" ]; then
    while true; do
        case "$1" in
            -p | --package)
                shift;
                package=$1
                shift ;;

            -h | --help)
                echo "Usage:"
                echo " -p --package <pkg>   process specific package"
                echo " -h --help            display this help message"
                exit 1
            shift ;;

            -- ) shift; break ;;
            * ) break ;;
        esac
    done
fi

#
# Utilities
#
pushd() {
    command pushd "$@" > /dev/null
}

popd() {
    command popd "$@" > /dev/null
}

#
# interface tools
#
green="\033[32;1m"
orange="\033[33;1m"
blue="\033[34;1m"
cyan="\033[36;1m"
white="\033[37;1m"
clean="\033[0m"

success() {
    echo -e "${green}$1${clean}"
}

info() {
    echo -e "${blue}$1${clean}"
}

warning() {
    echo -e "${orange}$1${clean}"
}

event() {
    echo -e "[+] ${blue}$1: ${clean}$2 ${orange}$3${clean}"
}

#
# Check the md5 hash from a file ($1) and compare with $2
#
checksum() {
    checksum=$(md5sum "$1" | awk '{ print $1 }')

    if [ "${checksum}" == "$2" ]; then
        # echo "[+] checksum match"
        return 0
    else
        echo "[-] checksum mismatch"
        return 1
    fi
}

#
# Sanity check
#
prepare() {
    if [ -z $GOPATH ]; then
        echo "[-] gopath not defined"
        exit 1
    fi

    echo "[+] ${modules} submodules loaded"

    mkdir -p "${WORKDIR}"
}

#
# Download a file and check the hash
# First argument needs to be the url, second is the md5 hash
#
download_file() {
    # set extra filename output or default output
    if [ ! -z $3 ]; then
        output=$3
    else
        output=$(basename "$1")
    fi

    # set default url
    fileurl=$1

    # if we use a mirror we rewrite the url
    if [ $USE_MIRROR == 1 ]; then
        # if we use a custom filename, that
        # filename will be used on the mirror site
        if [ ! -z $3 ]; then
            fileurl=$3
        fi

        fileurl="$MIRRORSRC/$(basename $fileurl)"
    fi

    event "downloading" "${output}"

    if [ -f "${output}" ]; then
        # Check for md5 before downloading the file
        checksum ${output} $2 && return
    fi

    # Download the file
    if [ "${INTERACTIVE}" == "false" ]; then
        curl -L -k -o "${output}" $fileurl
    else
        curl -L -k --progress-bar -C - -o "${output}" $fileurl
    fi

    # Checksum the downloaded file
    checksum ${output} $2 || false
}

download_git() {
    repository="$1"
    branch="$2"
    tag="$3"

    target=$(basename "${repository}")
    logfile="${target}-git.log"

    echo "Loading ${repository}" > "${logfile}"

    if [ -d "${target}" ]; then
        event "updating" "${repository}" "[${branch}]"

        # Ensure branch is up-to-date
        pushd "${target}"

        git fetch

        git checkout "${branch}" >> "${logfile}" 2>&1
        git pull origin "${branch}" >> "${logfile}" 2>&1

        [[ ! -z "$tag" ]] && git reset --hard "$tag" >> "${logfile}"

        popd
        return
    fi

    event "cloning" "${repository}" "[${branch}]"
    git clone -b "${branch}" "${repository}"
}

#
# Dynamic libraries management
#
ensure_libs() {
    echo "[+] verifing libraries dependancies"
    pushd "${ROOTDIR}"

    if [ ! -e lib64 ]; then ln -s usr/lib lib64; fi
    if [ ! -e lib ]; then ln -s lib64 lib; fi

    export LD_LIBRARY_PATH=${ROOTDIR}/lib:${ROOTDIR}/usr/lib

    # Copiyng ld-dependancy
    ld=$(ldd /bin/bash | grep ld-linux | awk '{ print $1 }')
    cp -aL $ld lib/

    for file in $(find -type f -executable); do
        # Looking for dynamic libraries shared
        libs=$(ldd $file 2>&1 | grep '=>' | grep -v '=>  (' | awk '{ print $3 }' || true)

        # Checking each libraries
        for lib in $libs; do
            libname=$(basename $lib)

            # Library found and not the already installed one
            if [ -e lib/$libname ] || [ "$lib" == "${PWD}/usr/lib/$libname" ]; then
                continue
            fi

            # Grabbing library from host
            cp -avL $lib lib/
        done
    done

    popd
}

clean_staging() {
    echo "[+] cleaning staging files"

    # cleaning staging files
    rm -rf "${WORKDIR}"/*
}

#
# helpers
#
get_size() {
    du -shc --apparent-size $1 | tail -1 | awk '{ print $1 }'
}

main() {
    info "===================================="
    info "=  Zero-OS Extra Binaries Builder  ="
    info "===================================="
    echo ""

    prepare

    if [ ! -z ${package} ]; then
        if [[ -d "${PKGDIR}/${package}" ]]; then
            echo "[+] building package: ${package}"
            build_${package}

        else
            echo "[-] unknown package: ${package}"
            exit 1
        fi
    fi
}

main
