# syntax=docker/dockerfile:1
ARG GO_VERSION
ARG UBI10_MINIMAL_VERSION
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
COPY <<'VERIFY' /tmp/verify-artifact
set -eo pipefail
# Use readelf for linkage because older file versions mislabel static PIE.
test -x "${TARGET_PREFIX}/bin/apache-exporter"
file -b "${TARGET_PREFIX}/bin/apache-exporter" | grep -q '^ELF '
readelf -l "${TARGET_PREFIX}/bin/apache-exporter" | ( ! grep -q INTERP )
readelf -d "${TARGET_PREFIX}/bin/apache-exporter" | ( ! grep -q '(NEEDED)' )
timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/apache-exporter" --version
VERIFY
RUN bash /tmp/verify-artifact

FROM centos:7.9.2009 AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG TARGET_PREFIX
COPY templates/centos7-vault.repo /etc/yum.repos.d/static-builds-vault.repo
RUN --mount=type=cache,id=apache-exporter-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    yum -y --disablerepo='*' --enablerepo="static-builds-vault-$(uname -m)" install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=verify /tmp/verify-artifact /tmp/verify-artifact
RUN bash /tmp/verify-artifact

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
EXPOSE 9117
ENTRYPOINT ["./bin/apache-exporter"]
