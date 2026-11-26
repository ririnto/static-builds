# syntax=docker/dockerfile:1
ARG GOLANG_IMAGE_TAG
ARG UBI10_MINIMAL_VERSION
ARG CENTOS_VERSION
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
# Use readelf for linkage because older file versions mislabel static PIE.
RUN test -x "${TARGET_PREFIX}/coredns"
RUN file -b "${TARGET_PREFIX}/coredns" | grep -q '^ELF '
RUN readelf -l "${TARGET_PREFIX}/coredns" | ( ! grep -q INTERP )
RUN readelf -d "${TARGET_PREFIX}/coredns" | ( ! grep -q '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/coredns" -version
RUN "${TARGET_PREFIX}/coredns" -plugins

FROM centos:${CENTOS_VERSION} AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG CENTOS_VERSION
ARG TARGET_PREFIX
RUN --mount=type=cache,id=coredns-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    if [ "$TARGETARCH" = arm64 ]; then \
        vault="https://vault.centos.org/altarch/${CENTOS_VERSION}/os/aarch64/"; \
    else \
        vault="https://vault.centos.org/${CENTOS_VERSION}/os/x86_64/"; \
    fi; \
    sed -i '/^\[base\]/,/^\[/s/^mirrorlist=/#mirrorlist=/' /etc/yum.repos.d/CentOS-Base.repo; \
    sed -i "/^\\[base\\]/,/^\\[/s|^#\\?baseurl=.*|baseurl=$vault|" /etc/yum.repos.d/CentOS-Base.repo; \
    yum -y --disablerepo='*' --enablerepo=base --setopt=base.gpgcheck=1 install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
# Use readelf for linkage because older file versions mislabel static PIE.
RUN test -x "${TARGET_PREFIX}/coredns"
RUN file -b "${TARGET_PREFIX}/coredns" | grep -q '^ELF '
RUN readelf -l "${TARGET_PREFIX}/coredns" | ( ! grep -q INTERP )
RUN readelf -d "${TARGET_PREFIX}/coredns" | ( ! grep -q '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/coredns" -version
RUN "${TARGET_PREFIX}/coredns" -plugins

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
