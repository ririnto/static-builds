ARG ALPINE_VERSION
ARG UBI9_MICRO_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS build
ARG NGINX_VERSION
ARG NGINX_VTS_MODULE_VERSION
ARG NGINX_LUA_MODULE_VERSION
JP|ARG NGINX_LUA_UPSTREAM_MODULE_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        libatomic_ops-dev \
        libatomic_ops-static \
        linux-headers \
        luajit \
        luajit-dev \
        openssl-dev \
        openssl-libs-static \
        pcre2-dev \
        pcre2-static \
        zlib-dev \
        zlib-static
ADD "src/nginx-${NGINX_VERSION}.tar.gz" /build/
ADD "src/nginx-module-vts-${NGINX_VTS_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD "src/lua-nginx-module-${NGINX_LUA_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
JP|ADD "src/lua-upstream-nginx-module-${NGINX_LUA_UPSTREAM_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD "src/lua-resty-upstream-healthcheck-${NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION}.tar.gz" "/build/"
RUN --mount=type=cache,target=/var/cache/apk \
    cp "/build/lua-resty-upstream-healthcheck-${NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION}/lib/resty/upstream/healthcheck.lua" \
       /build/resty_upstream_healthcheck.lua && \
    luajit -bg /build/resty_upstream_healthcheck.lua /build/resty_upstream_healthcheck.o && \
    test -s /build/resty_upstream_healthcheck.o
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN LUAJIT_LIB="/usr/lib" LUAJIT_INC="/usr/include/luajit-2.1" ./configure \
    # --help \
    --prefix=${TARGET_PREFIX} \
    # --sbin-path=PATH \
    # --modules-path=PATH \
    # --conf-path=PATH \
    # --error-log-path=PATH \
    # --pid-path=PATH \
    # --lock-path=PATH \
    # --user=USER \
    # --group=GROUP \
    # --build=NAME \
    # --builddir=DIR \
    # --with-select_module \
    # --without-select_module \
    # --with-poll_module \
    # --without-poll_module \
    --with-threads \
    --with-file-aio \
    # --without-quic_bpf_module \
    --with-http_ssl_module \
    --with-http_v2_module \
    --with-http_v3_module \
    --with-http_realip_module \
    # --with-http_addition_module \
    # --with-http_xslt_module \
    # --with-http_xslt_module=dynamic \
    # --with-http_image_filter_module \
    # --with-http_image_filter_module=dynamic \
    # --with-http_geoip_module \
    # --with-http_geoip_module=dynamic \
    # --with-http_sub_module \
    # --with-http_dav_module \
    # --with-http_flv_module \
    # --with-http_mp4_module \
    # --with-http_gunzip_module \
    --with-http_gzip_static_module \
    # --with-http_auth_request_module \
    # --with-http_random_index_module \
    # --with-http_secure_link_module \
    # --with-http_degradation_module \
    # --with-http_slice_module \
    --with-http_stub_status_module \
    # --without-http_charset_module \
    # --without-http_gzip_module \
    --without-http_ssi_module \
    --without-http_userid_module \
    # --without-http_access_module \
    # --without-http_auth_basic_module \
    # --without-http_mirror_module \
    --without-http_autoindex_module \
    # --without-http_geo_module \
    # --without-http_map_module \
    --without-http_split_clients_module \
    # --without-http_referer_module \
    # --without-http_rewrite_module \
    # --without-http_proxy_module \
    --without-http_fastcgi_module \
    --without-http_uwsgi_module \
    --without-http_scgi_module \
    # --without-http_grpc_module \
    --without-http_memcached_module \
    # --without-http_limit_conn_module \
    # --without-http_limit_req_module \
    --without-http_empty_gif_module \
    --without-http_browser_module \
    # --without-http_upstream_hash_module \
    # --without-http_upstream_ip_hash_module \
    # --without-http_upstream_least_conn_module \
    # --without-http_upstream_random_module \
    # --without-http_upstream_keepalive_module \
    # --without-http_upstream_zone_module \
    # --with-http_perl_module \
    # --with-http_perl_module=dynamic \
    # --with-perl_modules_path=PATH \
    # --with-perl=PATH \
    # --http-log-path=PATH \
    # --http-client-body-temp-path=PATH \
    # --http-proxy-temp-path=PATH \
    # --http-fastcgi-temp-path=PATH \
    # --http-uwsgi-temp-path=PATH \
    # --http-scgi-temp-path=PATH \
    # --without-http \
    # --without-http-cache \
    # --with-mail \
    # --with-mail=dynamic \
    # --with-mail_ssl_module \
    # --without-mail_pop3_module \
    # --without-mail_imap_module \
    # --without-mail_smtp_module \
    --with-stream \
    # --with-stream=dynamic \
    --with-stream_ssl_module \
    --with-stream_realip_module \
    # --with-stream_geoip_module \
    # --with-stream_geoip_module=dynamic \
    --with-stream_ssl_preread_module \
    # --without-stream_limit_conn_module \
    # --without-stream_access_module \
    # --without-stream_geo_module \
    # --without-stream_map_module \
    # --without-stream_split_clients_module \
    # --without-stream_return_module \
    # --without-stream_pass_module \
    # --without-stream_set_module \
    # --without-stream_upstream_hash_module \
    # --without-stream_upstream_least_conn_module \
    # --without-stream_upstream_random_module \
    # --without-stream_upstream_zone_module \
    # --with-google_perftools_module \
    # --with-cpp_test_module \
    # --add-module=PATH \
    --add-module="nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
    --add-module="lua-nginx-module-${NGINX_LUA_MODULE_VERSION}" \
VP|    --add-module="lua-upstream-nginx-module-${NGINX_LUA_UPSTREAM_MODULE_VERSION}"
    # --add-dynamic-module=PATH \
    # --with-compat \
    # --with-cc=PATH \
    # --with-cpp=PATH \
    --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    --with-ld-opt="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    # --with-cpu-opt=CPU \
    # --without-pcre \
    # --with-pcre \
    # --with-pcre=DIR \
    # --with-pcre-opt=OPTIONS \
    # --with-pcre-jit \
    # --without-pcre2 \
    # --with-zlib=DIR \
    # --with-zlib-opt=OPTIONS \
    # --with-zlib-asm=CPU \
    --with-libatomic \
    # --with-libatomic=DIR \
    # --with-openssl=DIR \
    # --with-openssl-opt=OPTIONS \
    # --with-debug \
    ;
RUN make "-j$(nproc)"
RUN make install
RUN install -D \
    "/build/lua-resty-upstream-healthcheck-${NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION}/lib/resty/upstream/healthcheck.lua" \
    "${TARGET_PREFIX}/lib/lua/resty/upstream/healthcheck.lua"


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
    strace ${TARGET_PREFIX}/sbin/nginx -V


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]
