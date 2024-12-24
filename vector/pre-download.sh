#!/usr/bin/env sh
set -eu

. "$(dirname "$0")/.env"
. "$(dirname "$0")/../.github/scripts/download.sh"

# Downloads pre-built Vector musl binary from GitHub releases.
# Vector is complex to build from source (Rust + many dependencies),
# so we use the official pre-built musl binaries which are statically linked.
main() {
    download_source "https://github.com/vectordotdev/vector/releases/download/v${VECTOR_VERSION}/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" "$(dirname "$0")/src/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz"
}

case "${0}" in
    */pre-download.sh) main ;;
esac
