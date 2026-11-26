# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG CENTOS_VERSION
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
# Use readelf for linkage because older file versions mislabel static PIE.
RUN test "$TARGETARCH" = amd64
RUN test -x "${TARGET_PREFIX}/bin/vector"
RUN file -b "${TARGET_PREFIX}/bin/vector" | grep '^ELF '
RUN readelf -l "${TARGET_PREFIX}/bin/vector" | ( ! grep INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/vector" | ( ! grep '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/vector" --version
RUN "${TARGET_PREFIX}/bin/vector" list

FROM centos:${CENTOS_VERSION} AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG CENTOS_VERSION
ARG TARGET_PREFIX
RUN --mount=type=cache,id=vector-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
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
RUN test "$TARGETARCH" = amd64
RUN test -x "${TARGET_PREFIX}/bin/vector"
RUN file -b "${TARGET_PREFIX}/bin/vector" | grep '^ELF '
RUN readelf -l "${TARGET_PREFIX}/bin/vector" | ( ! grep INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/vector" | ( ! grep '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/vector" --version
RUN "${TARGET_PREFIX}/bin/vector" list

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
EXPOSE 8686
ENTRYPOINT ["./bin/vector"]
CMD ["--config", "/etc/vector/vector.yaml"]
