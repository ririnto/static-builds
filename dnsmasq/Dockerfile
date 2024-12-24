# syntax=docker/dockerfile:1.4
ARG ALPINE_VERSION
ARG DNSMASQ_VERSION
ARG UBI9_MICRO_VERSION


FROM alpine:${ALPINE_VERSION} AS build
ARG DNSMASQ_VERSION

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        linux-headers
ADD "src/dnsmasq-${DNSMASQ_VERSION}.tar.gz" /build/
WORKDIR "/build/dnsmasq-${DNSMASQ_VERSION}"
RUN make "-j$(nproc)" \
    CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    LDFLAGS="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    PREFIX=/usr/local/dnsmasq \
    install


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /usr/local/dnsmasq /usr/local/dnsmasq
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h /usr/local/dnsmasq/sbin/dnsmasq | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l /usr/local/dnsmasq/sbin/dnsmasq | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d /usr/local/dnsmasq/sbin/dnsmasq || true; \
    file_output="$(file /usr/local/dnsmasq/sbin/dnsmasq 2>&1 || true)"; \
    ldd_output="$(ldd /usr/local/dnsmasq/sbin/dnsmasq 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace /usr/local/dnsmasq/sbin/dnsmasq -v


FROM scratch

COPY --from=verify /usr/local/dnsmasq .
EXPOSE 53 53/udp 67/udp
ENTRYPOINT ["./sbin/dnsmasq"]
CMD ["-k"]
