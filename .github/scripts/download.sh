#!/usr/bin/env sh
set -eu

# Downloads a tarball and saves it to the specified path.
# @param $1 url - Download URL
# @param $2 dest_file - Destination file path
download_tarball() {
    url="${1}"
    dest_file="${2}"
    dest_dir="$(dirname "${dest_file}")"

    mkdir -p "${dest_dir}"
    wget -q -O "${dest_file}" "${url}"
}

# Downloads source tarball from official distribution site.
# @param $1 url - Download URL
# @param $2 dest_file - Destination file path
download_source() {
    url="${1}"
    dest_file="${2}"

    download_tarball "${url}" "${dest_file}"
}
