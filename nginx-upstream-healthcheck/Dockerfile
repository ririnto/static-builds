# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG TARGETARCH
ARG BUILD_TARGET=nginx-upstream-healthcheck
ARG NGINX_VERSION
ARG NGINX_VTS_MODULE_VERSION
ARG NJS_VERSION
ARG QUICKJS_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,id=nginx-upstream-healthcheck-apk-${TARGETARCH},sharing=locked,target=/var/cache/apk \
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
ADD ".cache/downloads/nginx-upstream-healthcheck/nginx-${NGINX_VERSION}.tar.gz" /build/
ADD ".cache/downloads/nginx-upstream-healthcheck/nginx-module-vts-${NGINX_VTS_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD ".cache/downloads/nginx-upstream-healthcheck/njs-${NJS_VERSION}.tar.gz" /build/
ADD ".cache/downloads/nginx-upstream-healthcheck/quickjs-${QUICKJS_VERSION}.tar.gz" /build/
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
  --conf-path=config/nginx.conf \
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
RUN mkdir -p "${TARGET_PREFIX}/config/conf.d"
COPY nginx-upstream-healthcheck/config/nginx.conf ${TARGET_PREFIX}/config/nginx.conf

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGETARCH
ARG BUILD_TARGET=nginx-upstream-healthcheck
ARG TARGET_PREFIX
RUN --mount=type=cache,id=nginx-upstream-healthcheck-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install binutils curl-minimal file openssl strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN set -eu; \
    case "$TARGETARCH" in \
        amd64) machine='Advanced Micro Devices X86-64' ;; \
        arm64) machine='AArch64' ;; \
        arm) machine='ARM' ;; \
        386) machine='Intel 80386' ;; \
        ppc64le) machine='PowerPC64' ;; \
        s390x) machine='IBM S/390' ;; \
        riscv64) machine='RISC-V' ;; \
        *) machine= ;; \
    esac; \
    for binary in "${TARGET_PREFIX}/sbin/nginx"; do \
        test -f "$binary"; \
        test -x "$binary"; \
        header=$(LC_ALL=C readelf -h "$binary"); \
        printf '%s\n' "$header" | grep -E 'Type: +((EXEC)|(DYN)) '; \
        if [ -n "$machine" ]; then \
            printf '%s\n' "$header" | grep -E "Machine: +$machine$"; \
        fi; \
        description=$(LC_ALL=C file -b "$binary"); \
        printf '%s\n' "$description" | grep -E '^ELF .* (statically linked|static-pie linked)(,|$)'; \
        segments=$(LC_ALL=C readelf -l "$binary"); \
        dynamic=$(LC_ALL=C readelf -d "$binary"); \
        if printf '%s\n' "$segments" | grep -w INTERP; then \
            echo "Unexpected ELF interpreter: $binary" >&2; exit 1; \
        fi; \
        if printf '%s\n' "$dynamic" | grep -F '(NEEDED)'; then \
            echo "Unexpected shared library dependency: $binary" >&2; exit 1; \
        fi; \
    done
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/sbin/nginx" -V

FROM scratch AS runtime

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} /
COPY nginx-upstream-healthcheck/passwd /etc/passwd
COPY nginx-upstream-healthcheck/group /etc/group
EXPOSE 80 443
ENTRYPOINT ["/sbin/nginx", "-p", "/", "-g", "daemon off;"]

FROM verify AS verify-runtime
COPY --from=runtime / /runtime/
COPY nginx-upstream-healthcheck/examples/config/nginx.conf /verification/nginx.conf
RUN set -e; \
    trap 'rm -f /tmp/healthcheck-build-ca.key /runtime/config/backend-ca.pem /runtime/config/healthcheck-verify.conf /runtime/etc/hosts' EXIT; \
    cp /verification/nginx.conf /runtime/config/healthcheck-verify.conf; \
    openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=healthcheck-build-ca \
        -keyout /tmp/healthcheck-build-ca.key -out /runtime/config/backend-ca.pem; \
    echo "127.0.0.1 maintenance.internal" >/runtime/etc/hosts; \
    chroot /runtime /sbin/nginx -p / -t -c config/healthcheck-verify.conf
RUN set -eu; \
    test_dir=$(mktemp -d /runtime/config/healthcheck-verify.XXXXXX); \
    pid=; \
    trap 'status=$?; \
        trap - EXIT; \
        if [ -n "$pid" ]; then \
            if kill "$pid"; then \
                wait "$pid" || status=1; \
            else \
                status=1; \
            fi; \
        fi; \
        rm -rf "$test_dir" || status=1; \
        exit "$status"' EXIT; \
    chmod 755 "$test_dir"; \
    printf '%s\n' \
        'worker_processes 1;' \
        "pid /config/${test_dir##*/}/nginx.pid;" \
        'error_log stderr notice;' \
        'events { worker_connections 64; }' \
        'http {' \
        '    access_log off;' \
        '    upstream backend {' \
        '        zone backend 64k;' \
        '        server 127.0.0.1:18091;' \
        '        healthcheck type=http interval=200ms timeout=500ms fall=2 rise=2 uri=/ready;' \
        '    }' \
        '    server {' \
        '        listen 127.0.0.1:18091;' \
        "        location = /ready { root /config/${test_dir##*/}; }" \
        "        location / { return 200 'backend'; }" \
        '    }' \
        '    server {' \
        '        listen 127.0.0.1:18092;' \
        '        location / { proxy_pass http://backend; }' \
        '    }' \
        '}' >"$test_dir/nginx.conf"; \
    touch "$test_dir/ready"; \
    chroot /runtime /sbin/nginx -p / -c "config/${test_dir##*/}/nginx.conf" -g 'daemon off;' & \
    pid=$!; \
    wait_for_status() { \
        for attempt in $(seq 1 15); do \
            if [ "$(curl --connect-timeout 1 --max-time 2 -o /dev/null -w '%{http_code}' http://127.0.0.1:18092/)" = "$1" ]; then \
                return 0; \
            fi; \
            sleep 1; \
        done; \
        echo "Timed out waiting for proxy status $1" >&2; \
        return 1; \
    }; \
    wait_for_status 200; \
    rm "$test_dir/ready"; \
    wait_for_status 502; \
    touch "$test_dir/ready"; \
    wait_for_status 200

FROM runtime
COPY --from=verify-runtime /runtime/ /
