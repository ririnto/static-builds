#!/usr/bin/env sh
set -eu

is_valid_tarball() {
    target_file="${1}"

    if [ ! -s "${target_file}" ]; then
        return 1
    fi

    if gzip -t "${target_file}" >/dev/null 2>&1; then
        tar -tzf "${target_file}" >/dev/null 2>&1
        return $?
    fi

    tar -tf "${target_file}" >/dev/null 2>&1
}

download_tarball() {
    url="${1}"
    dest_file="${2}"
    dest_dir="$(dirname "${dest_file}")"

    mkdir -p "${dest_dir}"
    if is_valid_tarball "${dest_file}"; then
        return 0
    fi

    rm -f "${dest_file}"
    wget -q -O "${dest_file}" "${url}"
    if ! is_valid_tarball "${dest_file}"; then
        rm -f "${dest_file}"
        echo "Error: Downloaded file is invalid: ${dest_file}" >&2
        return 1
    fi
}
