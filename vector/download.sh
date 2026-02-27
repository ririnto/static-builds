#!/usr/bin/env sh
set -eu

MODULE_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "${MODULE_DIR}/.." && pwd)"

. "${MODULE_DIR}/.env"
. "${ROOT_DIR}/.github/scripts/common.sh"

download_tarball "https://github.com/vectordotdev/vector/releases/download/v${VECTOR_VERSION}/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" "${MODULE_DIR}/src/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" "${MODULE_DIR}/checksums/sha256sums.txt"
