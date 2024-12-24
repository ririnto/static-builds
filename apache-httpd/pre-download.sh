#!/usr/bin/env sh
set -eu

. "$(dirname "$0")/.env"
. "$(dirname "$0")/../.github/scripts/download.sh"

# Downloads sources required for apache-httpd build.
main() {
    download_source "https://downloads.apache.org/httpd/httpd-${HTTPD_VERSION}.tar.gz" "$(dirname "$0")/src/httpd-${HTTPD_VERSION}.tar.gz"
    download_source "https://downloads.apache.org/apr/apr-${APR_VERSION}.tar.gz" "$(dirname "$0")/src/apr-${APR_VERSION}.tar.gz"
    download_source "https://downloads.apache.org/apr/apr-util-${APR_UTIL_VERSION}.tar.gz" "$(dirname "$0")/src/apr-util-${APR_UTIL_VERSION}.tar.gz"
}

case "${0}" in
    */pre-download.sh) main ;;
esac
