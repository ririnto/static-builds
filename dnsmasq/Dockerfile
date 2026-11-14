# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG TARGETARCH
ARG BUILD_TARGET=dnsmasq
ARG DNSMASQ_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,id=dnsmasq-apk-${TARGETARCH},sharing=locked,target=/var/cache/apk \
    apk add \
        build-base \
        coreutils \
        linux-headers
ADD ".cache/downloads/dnsmasq/dnsmasq-${DNSMASQ_VERSION}.tar.gz" /build/
WORKDIR "/build/dnsmasq-${DNSMASQ_VERSION}"
RUN make "-j$(nproc)" \
    CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    LDFLAGS="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    PREFIX=${TARGET_PREFIX} \
    install

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-e", "-o", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=dnsmasq
ARG TARGET_PREFIX
RUN --mount=type=cache,id=dnsmasq-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN test -x "${TARGET_PREFIX}/sbin/dnsmasq"
RUN file -b "${TARGET_PREFIX}/sbin/dnsmasq" | grep -Eq '^ELF .* (statically linked|static-pie linked)(,|$)'
RUN readelf -l "${TARGET_PREFIX}/sbin/dnsmasq" | ( ! grep -q INTERP )
RUN readelf -d "${TARGET_PREFIX}/sbin/dnsmasq" | ( ! grep -q '(NEEDED)' )

RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/sbin/dnsmasq" -v

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp 67/udp
ENTRYPOINT ["./sbin/dnsmasq"]
CMD ["-k"]
