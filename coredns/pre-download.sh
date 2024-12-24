#!/usr/bin/env sh
set -eu

. "$(dirname "$0")/.env"
. "$(dirname "$0")/../.github/scripts/download.sh"

# Downloads sources required for coredns build.
main() {
    download_source "https://github.com/coredns/coredns/archive/refs/tags/v${COREDNS_VERSION}.tar.gz" "$(dirname "$0")/src/coredns-${COREDNS_VERSION}.tar.gz"
}

case "${0}" in
    */pre-download.sh) main ;;
esac
