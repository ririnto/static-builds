#!/usr/bin/env sh

. .env
curl -L -o "nginx-${NGINX_VERSION}.tar.gz" "https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz"
curl -L -o "nginx-module-vts-${NGINX_MODULE_VTS_VERSION}.tar.gz" "https://github.com/vozlt/nginx-module-vts/archive/refs/tags/v${NGINX_MODULE_VTS_VERSION}.tar.gz"
