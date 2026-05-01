ARG ALPINE_VERSION
ARG UBI9_MICRO_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS build
ARG NGINX_VERSION
ARG NGINX_VTS_MODULE_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
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
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN ./configure \
    --prefix=${TARGET_PREFIX} \
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
    --add-module="nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
    --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    --with-ld-opt="-static -static-pie -static-libgcc -Wl,-E -rdynamic -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed" \
    --with-libatomic \
    ;
RUN make "-j$(nproc)"
RUN make install


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h ${TARGET_PREFIX}/sbin/nginx | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l ${TARGET_PREFIX}/sbin/nginx | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d ${TARGET_PREFIX}/sbin/nginx || true; \
    file_output="$(file ${TARGET_PREFIX}/sbin/nginx 2>&1 || true)"; \
    ldd_output="$(ldd ${TARGET_PREFIX}/sbin/nginx 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace ${TARGET_PREFIX}/sbin/nginx -V; \
    test ! -d ${TARGET_PREFIX}/lualib || (echo "Error: lualib directory should not exist" && exit 1); \
    test ! -d ${TARGET_PREFIX}/lib/lua || (echo "Error: lib/lua directory should not exist" && exit 1)


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]
