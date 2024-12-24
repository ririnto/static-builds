#!/usr/bin/env sh
set -eu

. "$(dirname "$0")/.env"
. "$(dirname "$0")/../.github/scripts/download.sh"

# Downloads sources required for nginx build.
main() {
    download_source "https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz" "$(dirname "$0")/src/nginx-${NGINX_VERSION}.tar.gz"
    download_source "https://github.com/vozlt/nginx-module-vts/archive/refs/tags/v${NGINX_MODULE_VTS_VERSION}.tar.gz" "$(dirname "$0")/src/nginx-module-vts-${NGINX_MODULE_VTS_VERSION}.tar.gz"
}

case "${0}" in
    */pre-download.sh) main ;;
esac
