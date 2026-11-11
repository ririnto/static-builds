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

WORKDIR /tmp
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
ADD ".cache/downloads/apache-httpd/httpd-${HTTPD_VERSION}.tar.gz" /tmp/
ADD ".cache/downloads/apache-httpd/apr-${APR_VERSION}.tar.gz" /tmp/
ADD ".cache/downloads/apache-httpd/apr-util-${APR_UTIL_VERSION}.tar.gz" /tmp/
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
ARG TARGETARCH
ARG BUILD_TARGET=apache-httpd
ARG TARGET_PREFIX
RUN --mount=type=cache,id=apache-httpd-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
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
    for binary in "${TARGET_PREFIX}/bin/httpd" "${TARGET_PREFIX}/bin/rotatelogs"; do \
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
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/httpd" -V
RUN set -eu; \
    test_dir=$(mktemp -d); \
    trap 'rm -rf "$test_dir"' EXIT; \
    printf '%s\n' 'static-builds rotatelogs check' >"$test_dir/input"; \
    timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/rotatelogs" "$test_dir/output" 60 <"$test_dir/input"; \
    test "$(cat "$test_dir"/output.*)" = 'static-builds rotatelogs check'
RUN "${TARGET_PREFIX}/bin/httpd" -l
RUN "${TARGET_PREFIX}/bin/httpd" -M

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./bin/httpd"]
CMD ["-D", "FOREGROUND"]
