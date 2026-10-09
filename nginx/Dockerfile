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
        -DBUILD_SHARED_LIBS=OFF -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_BUILD_TYPE=Release && \
    cmake --build /build/quickjs-build "-j$(nproc)" && \
    cmake --install /build/quickjs-build
ENV NJS_QUICKJS=YES NJS_LIBXSLT=NO
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
    --add-module="/build/njs-${NJS_VERSION}/nginx" \
    --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    --with-ld-opt="-L/usr/local/lib -static -static-pie -static-libgcc -Wl,-E -rdynamic -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed" \
    --with-libatomic \
    ;
RUN make "-j$(nproc)" && make install
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/



FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG NJS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace openssl
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
    test ! -d ${TARGET_PREFIX}/lualib; \
    test ! -d ${TARGET_PREFIX}/lib/lua; \
    test ! -d ${TARGET_PREFIX}/share/lua; \
    ${TARGET_PREFIX}/sbin/nginx -t -c conf/scripting.conf
RUN <<'EOF'
set -eu
nginx_bin="${TARGET_PREFIX}/sbin/nginx"
"$nginx_bin" -c conf/scripting.conf -g 'daemon off;' &
nginx_pid=$!
trap '"$nginx_bin" -s quit -c conf/scripting.conf; wait "$nginx_pid"' EXIT
curl --retry 10 --retry-connrefused --retry-delay 1 --retry-max-time 10 --max-time 2 -fsS http://127.0.0.1:18081/njs > /tmp/njs-result
grep -Fx "njs ${NJS_VERSION}" /tmp/njs-result
curl -fsS http://127.0.0.1:18081/njs | grep -Fx "njs ${NJS_VERSION}"
timeout 3 bash -c 'cat < /dev/tcp/127.0.0.1/18082' | grep -Fx "njs ${NJS_VERSION}"
EOF


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]
