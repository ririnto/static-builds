# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG NGINX_VERSION
ARG NGINX_VTS_MODULE_VERSION
ARG NJS_VERSION
ARG QUICKJS_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        cmake \
        pkgconf \
        libatomic_ops-dev \
        libatomic_ops-static \
        linux-headers \
        openssl-dev \
        openssl-libs-static \
        pcre2-dev \
        pcre2-static \
        zlib-dev \
        zlib-static
ADD ".tmp/nginx-${NGINX_VERSION}.tar.gz" /build/
ADD ".tmp/nginx-module-vts-${NGINX_VTS_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD ".tmp/njs-${NJS_VERSION}.tar.gz" /build/
ADD ".tmp/quickjs-${QUICKJS_VERSION}.tar.gz" /build/
RUN cmake -S "/build/quickjs-${QUICKJS_VERSION}" -B /build/quickjs-build \
    -DBUILD_SHARED_LIBS=OFF -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_BUILD_TYPE=Release
RUN cmake --build /build/quickjs-build "-j$(nproc)"
RUN cmake --install /build/quickjs-build
ENV NJS_QUICKJS=YES
ENV NJS_LIBXSLT=NO
COPY nginx-upstream-healthcheck/ngx_http_upstream_healthcheck_module "/build/nginx-${NGINX_VERSION}/ngx_http_upstream_healthcheck_module"
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN ./configure \
  --prefix="${TARGET_PREFIX}" \
  --with-threads \
  --with-file-aio \
  --with-http_ssl_module \
  --with-http_v2_module \
  --with-http_v3_module \
  --with-http_realip_module \
  --with-http_gzip_static_module \
  --with-http_stub_status_module \
  --without-http_ssi_module \
  --without-http_userid_module \
  --without-http_autoindex_module \
  --without-http_split_clients_module \
  --without-http_fastcgi_module \
  --without-http_uwsgi_module \
  --without-http_scgi_module \
  --without-http_memcached_module \
  --without-http_empty_gif_module \
  --without-http_browser_module \
  --with-stream \
  --with-stream_ssl_module \
  --with-stream_realip_module \
  --with-stream_ssl_preread_module \
  "--add-module=nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
  "--add-module=ngx_http_upstream_healthcheck_module" \
  "--add-module=/build/njs-${NJS_VERSION}/nginx" \
  --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
  --with-ld-opt="-L/usr/local/lib -static -static-pie -static-libgcc -Wl,-E -rdynamic -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed" \
  --with-libatomic
RUN make "-j$(nproc)"
RUN make install
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/conf/upstream-healthcheck.conf ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/README.md ${TARGET_PREFIX}/README.md

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install curl-minimal file openssl
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN file "${TARGET_PREFIX}/sbin/nginx" | grep -E 'statically linked|static-pie linked'

RUN test -x "${TARGET_PREFIX}/sbin/nginx"
RUN test -r "${TARGET_PREFIX}/conf/scripting.conf"
RUN test -r "${TARGET_PREFIX}/conf/scripting.js"
RUN test -r "${TARGET_PREFIX}/conf/upstream-healthcheck.conf"
RUN test -r "${TARGET_PREFIX}/README.md"

FROM scratch AS runtime

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} /
COPY nginx-upstream-healthcheck/passwd /etc/passwd
COPY nginx-upstream-healthcheck/group /etc/group
EXPOSE 80 443
ENTRYPOINT ["/sbin/nginx", "-p", "/", "-g", "daemon off;"]

FROM verify AS verify-runtime
COPY --from=runtime / /runtime/
RUN set -e; \
    trap 'rm -f /tmp/healthcheck-build-ca.key /runtime/conf/backend-ca.pem /runtime/etc/hosts' EXIT; \
    openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=healthcheck-build-ca \
    -keyout /tmp/healthcheck-build-ca.key -out /runtime/conf/backend-ca.pem && \
    echo "127.0.0.1 maintenance.internal" >/runtime/etc/hosts && \
    chroot /runtime /sbin/nginx -p / -t -c conf/upstream-healthcheck.conf
RUN <<'EOF'
set -eu
test_dir=$(mktemp -d /runtime/conf/healthcheck-verify.XXXXXX)
pid=
trap 'status=$?; trap - EXIT; if [ -n "$pid" ]; then if kill "$pid"; then wait "$pid" || status=1; else status=1; fi; fi; rm -rf "$test_dir" || status=1; exit "$status"' EXIT
chmod 755 "$test_dir"
cat >"$test_dir/nginx.conf" <<CONFIG
worker_processes 1;
pid /conf/${test_dir##*/}/nginx.pid;
error_log stderr notice;
events { worker_connections 64; }
http {
    access_log off;
    upstream backend {
        zone backend 64k;
        server 127.0.0.1:18091;
        healthcheck type=http interval=200ms timeout=500ms fall=2 rise=2 uri=/ready;
    }
    server {
        listen 127.0.0.1:18091;
        location = /ready { root /conf/${test_dir##*/}; }
        location / { return 200 'backend'; }
    }
    server {
        listen 127.0.0.1:18092;
        location / { proxy_pass http://backend; }
    }
}
CONFIG
touch "$test_dir/ready"
chroot /runtime /sbin/nginx -p / -c "conf/${test_dir##*/}/nginx.conf" -g 'daemon off;' &
pid=$!
wait_for_status() {
    for attempt in $(seq 1 15); do
        if [ "$(curl --connect-timeout 1 --max-time 2 -o /dev/null -w '%{http_code}' http://127.0.0.1:18092/)" = "$1" ]; then
            return 0
        fi
        sleep 1
    done
    echo "Timed out waiting for proxy status $1" >&2
    return 1
}
wait_for_status 200
rm "$test_dir/ready"
wait_for_status 502
touch "$test_dir/ready"
wait_for_status 200
EOF

FROM runtime
COPY --from=verify-runtime /runtime/ /
