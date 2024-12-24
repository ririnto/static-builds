#!/usr/bin/env sh
set -eu

. "$(dirname "$0")/.env"
. "$(dirname "$0")/../.github/scripts/download.sh"

# Downloads sources required for dnsmasq build.
main() {
    download_source "https://thekelleys.org.uk/dnsmasq/dnsmasq-${DNSMASQ_VERSION}.tar.gz" "$(dirname "$0")/src/dnsmasq-${DNSMASQ_VERSION}.tar.gz"
}

case "${0}" in
    */pre-download.sh) main ;;
esac
