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
    retries="${DOWNLOAD_RETRIES:-5}"
    connect_timeout="${DOWNLOAD_CONNECT_TIMEOUT:-15}"
    max_time="${DOWNLOAD_MAX_TIME:-300}"
    attempt=1

    mkdir -p "${dest_dir}"
    echo "Downloading ${url} -> ${dest_file}" >&2

    if is_valid_tarball "${dest_file}"; then
        return 0
    fi

    rm -f "${dest_file}"

    while [ "${attempt}" -le "${retries}" ]; do
        tmp_file="${dest_file}.tmp.$$"
        rm -f "${tmp_file}"

        if command -v curl >/dev/null 2>&1; then
            if curl --fail --location --silent --show-error \
                --connect-timeout "${connect_timeout}" \
                --max-time "${max_time}" \
                --output "${tmp_file}" "${url}"; then
                rc=0
            else
                rc=$?
            fi
        else
            if wget -q \
                --connect-timeout="${connect_timeout}" \
                --timeout="${max_time}" \
                -O "${tmp_file}" "${url}"; then
                rc=0
            else
                rc=$?
            fi
        fi

        if [ "${rc}" -eq 0 ]; then
            if is_valid_tarball "${tmp_file}"; then
                mv "${tmp_file}" "${dest_file}"
                return 0
            fi

            rm -f "${tmp_file}"
            echo "Error: Downloaded file is invalid: ${dest_file} (from ${url})" >&2
            return 1
        fi

        rm -f "${tmp_file}"
        echo "Error: Download failed (attempt ${attempt}/${retries}, exit=${rc}): ${url} -> ${dest_file}" >&2

        attempt=$((attempt + 1))
        if [ "${attempt}" -le "${retries}" ]; then
            sleep 1
        fi
    done

    return 1
}
