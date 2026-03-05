#!/usr/bin/env sh
set -eu

download_tarball() {
    if [ $# -ne 2 ]; then
        echo "Error: download_tarball expects 2 arguments: <url> <dest_file>" >&2
        return 1
    fi
    url="${1}"
    dest_file="${2}"
    dest_dir="$(dirname "${dest_file}")"
    retries="${DOWNLOAD_RETRIES:-5}"
    connect_timeout="${DOWNLOAD_CONNECT_TIMEOUT:-15}"
    max_time="${DOWNLOAD_MAX_TIME:-300}"
    attempt=1

    mkdir -p "${dest_dir}"
    echo "Downloading ${url} -> ${dest_file}" >&2

    # If file exists and is non-empty, skip download
    if [ -f "${dest_file}" ] && [ -s "${dest_file}" ]; then
        echo "File exists: ${dest_file}" >&2
        return 0
    fi

    # Validate URL scheme
    case "${url}" in
        https://*)
            ;;
        http://*)
            ;;
        *)
            echo "Error: Unsupported URL scheme: ${url}" >&2
            return 1
            ;;
    esac

    # Create temporary file in destination directory for atomic move
    tmp_file=$(mktemp "${dest_file}.tmp.XXXXXX") || {
        echo "Error: Failed to create temporary file in ${dest_dir}" >&2
        return 1
    }

    while [ "${attempt}" -le "${retries}" ]; do
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
            # Check file is not empty
            if [ -s "${tmp_file}" ]; then
                # Atomic move: verified temp file replaces destination
                mv -f "${tmp_file}" "${dest_file}"
                return 0
            fi

            rm -f "${tmp_file}"
            echo "Error: Downloaded file is empty: ${dest_file} (from ${url})" >&2
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
