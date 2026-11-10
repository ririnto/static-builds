# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG HTTPD_VERSION
ARG APR_VERSION
ARG APR_UTIL_VERSION
ARG TARGET_PREFIX

WORKDIR /tmp
RUN --mount=type=cache,target=/var/cache/apk \
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
ADD ".tmp/httpd-${HTTPD_VERSION}.tar.gz" /tmp/
ADD ".tmp/apr-${APR_VERSION}.tar.gz" /tmp/
ADD ".tmp/apr-util-${APR_UTIL_VERSION}.tar.gz" /tmp/
RUN mv "/tmp/apr-${APR_VERSION}" "/tmp/httpd-${HTTPD_VERSION}/srclib/apr"
RUN mv "/tmp/apr-util-${APR_UTIL_VERSION}" "/tmp/httpd-${HTTPD_VERSION}/srclib/apr-util"
WORKDIR "/tmp/httpd-${HTTPD_VERSION}"
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
ARG TARGET_PREFIX
RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN readelf -h "${TARGET_PREFIX}/bin/httpd" | grep -E 'Type:|Machine:'
RUN file "${TARGET_PREFIX}/bin/httpd" | grep -E 'statically linked|static-pie linked'
RUN readelf -h "${TARGET_PREFIX}/bin/rotatelogs" | grep -E 'Type:|Machine:'
RUN file "${TARGET_PREFIX}/bin/rotatelogs" | grep -E 'statically linked|static-pie linked'
RUN test -x "${TARGET_PREFIX}/bin/httpd"
RUN test -x "${TARGET_PREFIX}/bin/rotatelogs"
RUN strace "${TARGET_PREFIX}/bin/httpd" -V
RUN "${TARGET_PREFIX}/bin/httpd" -l
RUN "${TARGET_PREFIX}/bin/httpd" -M

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./bin/httpd"]
CMD ["-D", "FOREGROUND"]
