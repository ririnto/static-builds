# syntax=docker/dockerfile:1
ARG GO_VERSION
ARG UBI10_MINIMAL_VERSION
ARG CENTOS_VERSION
ARG TARGET_PREFIX=/out

FROM golang:${GO_VERSION}-alpine AS build
ARG TARGETARCH
ARG BUILD_TARGET=apache-exporter
ARG APACHE_EXPORTER_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,id=apache-exporter-apk-${TARGETARCH},sharing=locked,target=/var/cache/apk \
    apk add \
        git
ADD ".cache/downloads/apache-exporter/apache-exporter-${APACHE_EXPORTER_VERSION}.tar.gz" /build/
WORKDIR "/build/apache_exporter-${APACHE_EXPORTER_VERSION}"
RUN mkdir -p ${TARGET_PREFIX}/bin
RUN --mount=type=cache,id=apache-exporter-go-build-${TARGETARCH},sharing=locked,target=/root/.cache/go-build \
    --mount=type=cache,id=apache-exporter-go-modules-${TARGETARCH},sharing=locked,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -a -tags netgo -trimpath \
        -ldflags "-X github.com/prometheus/common/version.Version=${APACHE_EXPORTER_VERSION}" \
        -o ${TARGET_PREFIX}/bin/apache-exporter .

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=apache-exporter
ARG TARGET_PREFIX
RUN --mount=type=cache,id=apache-exporter-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
# Use readelf for linkage because older file versions mislabel static PIE.
RUN test -x "${TARGET_PREFIX}/bin/apache-exporter"
RUN file -b "${TARGET_PREFIX}/bin/apache-exporter" | grep -q '^ELF '
RUN readelf -l "${TARGET_PREFIX}/bin/apache-exporter" | ( ! grep -q INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/apache-exporter" | ( ! grep -q '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/apache-exporter" --version

FROM centos:${CENTOS_VERSION} AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG CENTOS_VERSION
ARG TARGET_PREFIX
RUN --mount=type=cache,id=apache-exporter-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
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
RUN test -x "${TARGET_PREFIX}/bin/apache-exporter"
RUN file -b "${TARGET_PREFIX}/bin/apache-exporter" | grep -q '^ELF '
RUN readelf -l "${TARGET_PREFIX}/bin/apache-exporter" | ( ! grep -q INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/apache-exporter" | ( ! grep -q '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/apache-exporter" --version

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
EXPOSE 9117
ENTRYPOINT ["./bin/apache-exporter"]
