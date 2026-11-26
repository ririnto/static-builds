# syntax=docker/dockerfile:1
ARG GOLANG_IMAGE_TAG
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM golang:${GOLANG_IMAGE_TAG} AS build
ARG TARGETARCH
ARG BUILD_TARGET=coredns
ARG COREDNS_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,id=coredns-apk-${TARGETARCH},sharing=locked,target=/var/cache/apk \
    apk add \
        git \
        make
ADD ".cache/downloads/coredns/coredns-${COREDNS_VERSION}.tar.gz" /build/
WORKDIR "/build/coredns-${COREDNS_VERSION}"
RUN --mount=type=cache,id=coredns-go-build-${TARGETARCH},sharing=locked,target=/root/.cache/go-build \
    --mount=type=cache,id=coredns-go-modules-${TARGETARCH},sharing=locked,target=/go/pkg/mod \
    make coredns \
        GOTOOLCHAIN=local \
        CGO_ENABLED=0 \
        BUILDOPTS="-v -trimpath"

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=coredns
ARG COREDNS_VERSION
ARG TARGET_PREFIX
RUN --mount=type=cache,id=coredns-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /build/coredns-${COREDNS_VERSION}/coredns ${TARGET_PREFIX}/coredns
COPY <<'VERIFY' /tmp/verify-artifact
set -eo pipefail
# Use readelf for linkage because older file versions mislabel static PIE.
test -x "${TARGET_PREFIX}/coredns"
file -b "${TARGET_PREFIX}/coredns" | grep -q '^ELF '
readelf -l "${TARGET_PREFIX}/coredns" | ( ! grep -q INTERP )
readelf -d "${TARGET_PREFIX}/coredns" | ( ! grep -q '(NEEDED)' )
timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/coredns" -version
"${TARGET_PREFIX}/coredns" -plugins
VERIFY
RUN bash /tmp/verify-artifact

FROM centos:7.9.2009 AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG TARGET_PREFIX
COPY templates/centos7-vault.repo /etc/yum.repos.d/static-builds-vault.repo
RUN --mount=type=cache,id=coredns-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    yum -y --disablerepo='*' --enablerepo="static-builds-vault-$(uname -m)" install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=verify /tmp/verify-artifact /tmp/verify-artifact
RUN bash /tmp/verify-artifact

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
