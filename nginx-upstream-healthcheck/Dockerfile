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
ENV NJS_QUICKJS=YES NJS_LIBXSLT=NO
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
RUN make "-j$(nproc)" && make install
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/conf/upstream-healthcheck.conf ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/README.md ${TARGET_PREFIX}/README.md



FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG NJS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace openssl
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN set -eu; \
    readelf -h "${TARGET_PREFIX}/sbin/nginx" | grep -E 'Type:|Machine:'; \
    file "${TARGET_PREFIX}/sbin/nginx" | grep -E 'statically linked|static-pie linked'

RUN set -eu; \
    test -x "${TARGET_PREFIX}/sbin/nginx"; \
    test -r "${TARGET_PREFIX}/conf/scripting.conf"; \
    test -r "${TARGET_PREFIX}/conf/scripting.js"; \
    test -r "${TARGET_PREFIX}/conf/upstream-healthcheck.conf"; \
    test -r "${TARGET_PREFIX}/README.md"

RUN strace "${TARGET_PREFIX}/sbin/nginx" -V
RUN "${TARGET_PREFIX}/sbin/nginx" -t -c conf/scripting.conf

RUN set -eu; \
    nginx_pid=; \
    response_file=$(mktemp /tmp/njs-result.XXXXXX); \
    cleanup() { \
        status=$?; \
        trap - EXIT INT TERM; \
        if [ -n "$nginx_pid" ]; then \
            kill -QUIT "$nginx_pid"; \
            wait "$nginx_pid"; \
        fi; \
        rm -f "$response_file"; \
        exit "$status"; \
    }; \
    trap cleanup EXIT; \
    trap 'exit 130' INT; \
    trap 'exit 143' TERM; \
    timeout --signal=QUIT --kill-after=2s 40s "${TARGET_PREFIX}/sbin/nginx" -c conf/scripting.conf -g 'daemon off;' & \
    nginx_pid=$!; \
    timeout --signal=TERM --kill-after=1s 15s curl --retry 10 --retry-connrefused --retry-delay 1 --retry-max-time 10 --max-time 2 -fsS -o "$response_file" http://127.0.0.1:18081/njs; \
    grep -Fx "njs ${NJS_VERSION} engine QuickJS" "$response_file"; \
    timeout --signal=TERM --kill-after=1s 15s curl --max-time 5 -fsS -o "$response_file" http://127.0.0.1:18081/njs; \
    grep -Fx "njs ${NJS_VERSION} engine QuickJS" "$response_file"; \
    timeout 3 bash -c 'exec 3<>/dev/tcp/127.0.0.1/18082; IFS= read -r response <&3; printf "%s\n" "$response"' | grep -Fx "njs ${NJS_VERSION} engine QuickJS";



FROM scratch AS runtime

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} /
COPY nginx-upstream-healthcheck/passwd /etc/passwd
COPY nginx-upstream-healthcheck/group /etc/group
EXPOSE 80 443
ENTRYPOINT ["/sbin/nginx", "-p", "/", "-g", "daemon off;"]


FROM verify AS verify-runtime
COPY --from=runtime / /runtime/
RUN set -eu; \
    chroot /runtime /sbin/nginx -p / -t; \
    openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=healthcheck-build-ca \
    -keyout /tmp/healthcheck-build-ca.key -out /runtime/conf/backend-ca.pem; \
    printf '127.0.0.1 maintenance.internal\n' >/runtime/etc/hosts; \
    chroot /runtime /sbin/nginx -p / -t -c conf/upstream-healthcheck.conf; \
    rm -f /tmp/healthcheck-build-ca.key /runtime/conf/backend-ca.pem /runtime/etc/hosts;

RUN set -eu; \
    server_pid=; \
    response_file=$(mktemp /tmp/healthcheck-startup.XXXXXX); \
    cleanup() { \
        status=$?; \
        trap - EXIT INT TERM; \
        if [ -n "$server_pid" ]; then \
            kill -QUIT "$server_pid"; \
            wait "$server_pid"; \
        fi; \
        rm -f "$response_file"; \
        exit "$status"; \
    }; \
    trap cleanup EXIT; \
    trap 'exit 130' INT; \
    trap 'exit 143' TERM; \
    timeout --signal=QUIT --kill-after=2s 20s chroot /runtime /sbin/nginx -p / -g 'daemon off;' & \
    server_pid=$!; \
    timeout --signal=TERM --kill-after=1s 15s curl --retry 10 --retry-connrefused --retry-delay 1 --retry-max-time 10 --max-time 2 -fsS --output "$response_file" http://127.0.0.1/; \
    kill -0 "$(cat /runtime/logs/nginx.pid)"; \
    kill -QUIT "$server_pid"; \
    wait "$server_pid"; \
    server_pid=; \
    rm -f /runtime/logs/access.log /runtime/logs/error.log;


FROM runtime
COPY --from=verify-runtime /runtime/ /
