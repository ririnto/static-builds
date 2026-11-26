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
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=dnsmasq
ARG TARGET_PREFIX
RUN --mount=type=cache,id=dnsmasq-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY <<'VERIFY' /tmp/verify-artifact
set -eo pipefail
# Use readelf for linkage because older file versions mislabel static PIE.
test -x "${TARGET_PREFIX}/sbin/dnsmasq"
file -b "${TARGET_PREFIX}/sbin/dnsmasq" | grep -q '^ELF '
readelf -l "${TARGET_PREFIX}/sbin/dnsmasq" | ( ! grep -q INTERP )
readelf -d "${TARGET_PREFIX}/sbin/dnsmasq" | ( ! grep -q '(NEEDED)' )
timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/sbin/dnsmasq" -v
VERIFY
RUN bash /tmp/verify-artifact
RUN echo "nobody:x:65534:" > /etc/group
RUN echo "nobody:x:65534:65534:nobody:/home/nobody:/sbin/nologin" > /etc/passwd

FROM centos:7.9.2009 AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG TARGET_PREFIX
COPY templates/centos7-vault.repo /etc/yum.repos.d/static-builds-vault.repo
RUN --mount=type=cache,id=dnsmasq-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    yum -y --disablerepo='*' --enablerepo="static-builds-vault-$(uname -m)" install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=verify /tmp/verify-artifact /tmp/verify-artifact
RUN bash /tmp/verify-artifact
COPY --from=verify /etc/passwd /etc/group /etc/

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
COPY --from=verify-centos /etc/passwd /etc/group /etc/
EXPOSE 53 53/udp 67/udp
ENTRYPOINT ["./sbin/dnsmasq"]
CMD ["-k"]
