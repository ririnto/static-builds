#!/usr/bin/env sh
set -eu

. "$(dirname "$0")/.env"
. "$(dirname "$0")/../.github/scripts/download.sh"

# Downloads sources required for haproxy build.
main() {
    download_source "https://www.haproxy.org/download/${HAPROXY_VERSION%.*}/src/haproxy-${HAPROXY_VERSION}.tar.gz" "$(dirname "$0")/src/haproxy-${HAPROXY_VERSION}.tar.gz"
}

case "${0}" in
    */pre-download.sh) main ;;
esac
