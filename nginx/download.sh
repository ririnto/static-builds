#!/usr/bin/env sh
set -eu

MODULE_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "${MODULE_DIR}/.." && pwd)"

. "${MODULE_DIR}/.env"
. "${ROOT_DIR}/.github/scripts/common.sh"

download_tarball "https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz" "${MODULE_DIR}/src/nginx-${NGINX_VERSION}.tar.gz"
download_tarball "https://github.com/vozlt/nginx-module-vts/archive/refs/tags/v${NGINX_VTS_MODULE_VERSION}.tar.gz" "${MODULE_DIR}/src/nginx-module-vts-${NGINX_VTS_MODULE_VERSION}.tar.gz"
download_tarball "https://github.com/simpl/ngx_devel_kit/archive/refs/tags/v${NGINX_DEVEL_KIT_VERSION}.tar.gz" "${MODULE_DIR}/src/ngx_devel_kit-${NGINX_DEVEL_KIT_VERSION}.tar.gz"
download_tarball "https://github.com/openresty/lua-nginx-module/archive/refs/tags/v${NGINX_LUA_MODULE_VERSION}.tar.gz" "${MODULE_DIR}/src/lua-nginx-module-${NGINX_LUA_MODULE_VERSION}.tar.gz"
download_tarball "https://github.com/openresty/luajit2/archive/refs/tags/v${OPENRESTY_LUAJIT2_VERSION}.tar.gz" "${MODULE_DIR}/src/luajit2-${OPENRESTY_LUAJIT2_VERSION}.tar.gz"
download_tarball "https://github.com/openresty/lua-upstream-nginx-module/archive/refs/tags/v${NGINX_LUA_UPSTREAM_MODULE_VERSION}.tar.gz" "${MODULE_DIR}/src/lua-upstream-nginx-module-${NGINX_LUA_UPSTREAM_MODULE_VERSION}.tar.gz"
download_tarball "https://github.com/openresty/lua-resty-upstream-healthcheck/archive/refs/tags/v${NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION}.tar.gz" "${MODULE_DIR}/src/lua-resty-upstream-healthcheck-${NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION}.tar.gz"
