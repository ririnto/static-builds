#!/usr/bin/env sh
set -eu

MODULE_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "${MODULE_DIR}/.." && pwd)"

. "${MODULE_DIR}/.env"
. "${ROOT_DIR}/.github/scripts/common.sh"

download_tarball "https://thekelleys.org.uk/dnsmasq/dnsmasq-${DNSMASQ_VERSION}.tar.gz" "${MODULE_DIR}/src/dnsmasq-${DNSMASQ_VERSION}.tar.gz" "${MODULE_DIR}/checksums/sha256sums.txt"
