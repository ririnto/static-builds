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
ADD ".cache/downloads/nginx-${NGINX_VERSION}.tar.gz" /build/
ADD ".cache/downloads/nginx-module-vts-${NGINX_VTS_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD ".cache/downloads/njs-${NJS_VERSION}.tar.gz" /build/
ADD ".cache/downloads/quickjs-${QUICKJS_VERSION}.tar.gz" /build/
RUN cmake -S "/build/quickjs-${QUICKJS_VERSION}" -B /build/quickjs-build \
    -DBUILD_SHARED_LIBS=OFF -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_BUILD_TYPE=Release
RUN cmake --build /build/quickjs-build "-j$(nproc)"
RUN cmake --install /build/quickjs-build
ENV NJS_QUICKJS=YES
ENV NJS_LIBXSLT=NO
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
  "--add-module=/build/njs-${NJS_VERSION}/nginx" \
  --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
  --with-ld-opt="-L/usr/local/lib -static -static-pie -static-libgcc -Wl,-E -rdynamic -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed" \
  --with-libatomic
RUN make "-j$(nproc)"
RUN make install
RUN mkdir -p "${TARGET_PREFIX}/config/conf.d"
COPY nginx/config/nginx.conf ${TARGET_PREFIX}/config/nginx.conf

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX
RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
ARG TARGETARCH
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

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} /
COPY nginx-upstream-healthcheck/passwd /etc/passwd
COPY nginx-upstream-healthcheck/group /etc/group
EXPOSE 80 443
ENTRYPOINT ["/sbin/nginx", "-p", "/", "-g", "daemon off;"]
