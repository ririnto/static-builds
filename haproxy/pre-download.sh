#!/usr/bin/env sh

. .env
curl -L -o "haproxy-${HAPROXY_VERSION}.tar.gz" "https://www.haproxy.org/download/3.0/src/haproxy-${HAPROXY_VERSION}.tar.gz"
