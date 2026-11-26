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
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=nginx-upstream-healthcheck
ARG TARGET_PREFIX
RUN --mount=type=cache,id=nginx-upstream-healthcheck-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY <<'VERIFY' /tmp/verify-artifact
set -eo pipefail
# Use readelf for linkage because older file versions mislabel static PIE.
test -x "${TARGET_PREFIX}/sbin/nginx"
file -b "${TARGET_PREFIX}/sbin/nginx" | grep -q '^ELF '
readelf -l "${TARGET_PREFIX}/sbin/nginx" | ( ! grep -q INTERP )
readelf -d "${TARGET_PREFIX}/sbin/nginx" | ( ! grep -q '(NEEDED)' )
timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/sbin/nginx" -V
VERIFY
RUN bash /tmp/verify-artifact
RUN echo "nobody:x:65534:" > /etc/group
RUN echo "nobody:x:65534:65534:nobody:/home/nobody:/sbin/nologin" > /etc/passwd

FROM centos:7.9.2009 AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG TARGET_PREFIX
COPY templates/centos7-vault.repo /etc/yum.repos.d/static-builds-vault.repo
RUN --mount=type=cache,id=nginx-upstream-healthcheck-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    yum -y --disablerepo='*' --enablerepo="static-builds-vault-$(uname -m)" install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=verify /tmp/verify-artifact /tmp/verify-artifact
RUN bash /tmp/verify-artifact
COPY --from=verify /etc/passwd /etc/group /etc/

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} /
COPY --from=verify-centos /etc/passwd /etc/group /etc/
EXPOSE 80 443
ENTRYPOINT ["/sbin/nginx", "-p", "/", "-g", "daemon off;"]
