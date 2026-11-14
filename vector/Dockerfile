# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS download
ARG TARGETARCH
ARG BUILD_TARGET=vector
ARG VECTOR_VERSION
ARG TARGET_PREFIX

WORKDIR /download
ADD ".cache/downloads/vector/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" /download/
RUN mkdir -p ${TARGET_PREFIX}/bin
RUN mv vector-x86_64-unknown-linux-musl/bin/vector ${TARGET_PREFIX}/bin/

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-e", "-o", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=vector
ARG TARGET_PREFIX
RUN --mount=type=cache,id=vector-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=download ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN test "$TARGETARCH" = amd64
RUN test -x "${TARGET_PREFIX}/bin/vector"
RUN file -b "${TARGET_PREFIX}/bin/vector" | grep -Eq '^ELF .* (statically linked|static-pie linked)(,|$)'
RUN readelf -l "${TARGET_PREFIX}/bin/vector" | ( ! grep -q INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/vector" | ( ! grep -q '(NEEDED)' )

RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/vector" --version

RUN "${TARGET_PREFIX}/bin/vector" list

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 8686
ENTRYPOINT ["./bin/vector"]
CMD ["--config", "/etc/vector/vector.yaml"]
