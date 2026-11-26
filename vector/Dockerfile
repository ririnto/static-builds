# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS download
ARG TARGETARCH
ARG BUILD_TARGET=vector
ARG VECTOR_VERSION
ARG TARGET_PREFIX

WORKDIR /build
ADD ".cache/downloads/vector/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" /build/
RUN mkdir -p ${TARGET_PREFIX}/bin
RUN mv vector-x86_64-unknown-linux-musl/bin/vector ${TARGET_PREFIX}/bin/

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=vector
ARG TARGET_PREFIX
RUN --mount=type=cache,id=vector-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=download ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY <<'VERIFY' /tmp/verify-artifact
set -eo pipefail
# Use readelf for linkage because older file versions mislabel static PIE.
test "$TARGETARCH" = amd64
test -x "${TARGET_PREFIX}/bin/vector"
file -b "${TARGET_PREFIX}/bin/vector" | grep -q '^ELF '
readelf -l "${TARGET_PREFIX}/bin/vector" | ( ! grep -q INTERP )
readelf -d "${TARGET_PREFIX}/bin/vector" | ( ! grep -q '(NEEDED)' )
timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/vector" --version
"${TARGET_PREFIX}/bin/vector" list
VERIFY
RUN bash /tmp/verify-artifact

FROM centos:7.9.2009 AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG TARGET_PREFIX
COPY templates/centos7-vault.repo /etc/yum.repos.d/static-builds-vault.repo
RUN --mount=type=cache,id=vector-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    yum -y --disablerepo='*' --enablerepo="static-builds-vault-$(uname -m)" install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=verify /tmp/verify-artifact /tmp/verify-artifact
RUN bash /tmp/verify-artifact

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
EXPOSE 8686
ENTRYPOINT ["./bin/vector"]
CMD ["--config", "/etc/vector/vector.yaml"]
