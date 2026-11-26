# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG TARGETARCH
ARG BUILD_TARGET=apache-httpd
ARG HTTPD_VERSION
ARG APR_VERSION
ARG APR_UTIL_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,id=apache-httpd-apk-${TARGETARCH},sharing=locked,target=/var/cache/apk \
    apk add \
        build-base \
        linux-headers \
        perl \
        openssl-dev \
        openssl-libs-static \
        nghttp2-dev \
        nghttp2-static \
        pcre2-dev \
        pcre2-static \
        zlib-dev \
        zlib-static \
        expat-dev \
        expat-static \
        brotli-dev \
        brotli-static
ADD ".cache/downloads/apache-httpd/httpd-${HTTPD_VERSION}.tar.gz" /build/
ADD ".cache/downloads/apache-httpd/apr-${APR_VERSION}.tar.gz" /build/
ADD ".cache/downloads/apache-httpd/apr-util-${APR_UTIL_VERSION}.tar.gz" /build/
RUN mv "/build/apr-${APR_VERSION}" "/build/httpd-${HTTPD_VERSION}/srclib/apr"
RUN mv "/build/apr-util-${APR_UTIL_VERSION}" "/build/httpd-${HTTPD_VERSION}/srclib/apr-util"
WORKDIR "/build/httpd-${HTTPD_VERSION}"
RUN ./configure \
    --prefix=${TARGET_PREFIX} \
    --enable-pie \
    --enable-mods-static=few \
    --enable-socache-shmcb \
    --enable-so=no \
    --enable-watchdog \
    --enable-deflate \
    --enable-brotli \
    --enable-http \
    --enable-remoteip \
    --enable-proxy \
    --enable-proxy-http \
    --enable-proxy-balancer \
    --enable-proxy-hcheck \
    --enable-slotmem-shm \
    --enable-ssl \
    --enable-ssl-staticlib-deps \
    --enable-static-rotatelogs \
    --enable-http2 \
    --enable-nghttp2-staticlib-deps \
    --enable-proxy-http2 \
    --enable-lbmethod-byrequests \
    --enable-lbmethod-bytraffic \
    --enable-lbmethod-bybusyness \
    --disable-autoindex \
    --enable-asis \
    --enable-rewrite \
    --disable-proxy-ajp \
    --disable-proxy-connect \
    --disable-proxy-express \
    --disable-proxy-fcgi \
    --disable-proxy-fdpass \
    --disable-proxy-ftp \
    --disable-proxy-scgi \
    --disable-proxy-uwsgi \
    --disable-lbmethod-heartbeat \
    --with-included-apr \
    --with-mpm=event \
    CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    LDFLAGS="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    LIBS="-lbrotlienc -lbrotlidec -lbrotlicommon"
RUN make "-j$(nproc)"
RUN make install

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=apache-httpd
ARG TARGET_PREFIX
RUN --mount=type=cache,id=apache-httpd-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY <<'VERIFY' /tmp/verify-artifact
set -eo pipefail
# Use readelf for linkage because older file versions mislabel static PIE.
test -x "${TARGET_PREFIX}/bin/httpd"
test -x "${TARGET_PREFIX}/bin/rotatelogs"
file -b "${TARGET_PREFIX}/bin/httpd" | grep -q '^ELF '
readelf -l "${TARGET_PREFIX}/bin/httpd" | ( ! grep -q INTERP )
readelf -d "${TARGET_PREFIX}/bin/httpd" | ( ! grep -q '(NEEDED)' )
file -b "${TARGET_PREFIX}/bin/rotatelogs" | grep -q '^ELF '
readelf -l "${TARGET_PREFIX}/bin/rotatelogs" | ( ! grep -q INTERP )
readelf -d "${TARGET_PREFIX}/bin/rotatelogs" | ( ! grep -q '(NEEDED)' )
timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/httpd" -V
"${TARGET_PREFIX}/bin/httpd" -l
"${TARGET_PREFIX}/bin/httpd" -M
VERIFY
RUN bash /tmp/verify-artifact
RUN echo "daemon:x:2:" > /etc/group
RUN echo "daemon:x:2:2:daemon:/home/nobody:/sbin/nologin" > /etc/passwd

FROM centos:7.9.2009 AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG TARGET_PREFIX
COPY templates/centos7-vault.repo /etc/yum.repos.d/static-builds-vault.repo
RUN --mount=type=cache,id=apache-httpd-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    yum -y --disablerepo='*' --enablerepo="static-builds-vault-$(uname -m)" install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=verify /tmp/verify-artifact /tmp/verify-artifact
RUN bash /tmp/verify-artifact
COPY --from=verify /etc/passwd /etc/group /etc/

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
COPY --from=verify-centos /etc/passwd /etc/group /etc/
EXPOSE 80 443
ENTRYPOINT ["./bin/httpd"]
CMD ["-D", "FOREGROUND"]
