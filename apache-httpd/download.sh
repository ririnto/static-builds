#!/usr/bin/env sh
set -eu

MODULE_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "${MODULE_DIR}/.." && pwd)"

. "${MODULE_DIR}/.env"
. "${ROOT_DIR}/.github/scripts/common.sh"

download_tarball "https://downloads.apache.org/httpd/httpd-${HTTPD_VERSION}.tar.gz" "${MODULE_DIR}/src/httpd-${HTTPD_VERSION}.tar.gz" "https://downloads.apache.org/httpd/httpd-${HTTPD_VERSION}.tar.gz.sha256"
download_tarball "https://downloads.apache.org/apr/apr-${APR_VERSION}.tar.gz" "${MODULE_DIR}/src/apr-${APR_VERSION}.tar.gz" "https://downloads.apache.org/apr/apr-${APR_VERSION}.tar.gz.sha256"
download_tarball "https://downloads.apache.org/apr/apr-util-${APR_UTIL_VERSION}.tar.gz" "${MODULE_DIR}/src/apr-util-${APR_UTIL_VERSION}.tar.gz" "https://downloads.apache.org/apr/apr-util-${APR_UTIL_VERSION}.tar.gz.sha256"
