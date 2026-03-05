#!/usr/bin/env sh
set -eu

# Download a tarball from a URL to a destination file with retry support
#
# Outputs progress and error messages to stderr.
#
# :param str url: URL to download from
# :param str dest_file: Destination file path
# :returns: 0 on success, 1 on failure
# :rtype: int
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
  if [ -f "${dest_file}" ] && [ -s "${dest_file}" ]; then
    echo "File exists: ${dest_file}" >&2
    return 0
  fi
  case "${url}" in
  https://*) ;;
  http://*) ;;
  *)
    echo "Error: Unsupported URL scheme: ${url}" >&2
    return 1
    ;;
  esac

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
      if [ -s "${tmp_file}" ]; then
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
