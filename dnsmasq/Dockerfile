ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS build
ARG DNSMASQ_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        coreutils \
        linux-headers
ADD ".tmp/dnsmasq-${DNSMASQ_VERSION}.tar.gz" /build/
WORKDIR "/build/dnsmasq-${DNSMASQ_VERSION}"
RUN make "-j$(nproc)" \
    CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    LDFLAGS="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    PREFIX=${TARGET_PREFIX} \
    install


FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX
ARG DNSMASQ_VERSION

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY dnsmasq/dnsmasq-verify.sh /usr/local/bin/
RUN /usr/local/bin/dnsmasq-verify.sh


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp 67/udp
ENTRYPOINT ["./sbin/dnsmasq"]
CMD ["-k"]
