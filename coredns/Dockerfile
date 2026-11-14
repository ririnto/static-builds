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
ARG TARGETARCH
ARG BUILD_TARGET=coredns
ARG COREDNS_VERSION
ARG TARGET_PREFIX
RUN --mount=type=cache,id=coredns-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /build/coredns-${COREDNS_VERSION}/coredns ${TARGET_PREFIX}/coredns
RUN test -x "${TARGET_PREFIX}/coredns"
RUN file -b "${TARGET_PREFIX}/coredns" | grep -Eq '^ELF .* (statically linked|static-pie linked)(,|$)'
RUN readelf -l "${TARGET_PREFIX}/coredns" > /tmp/program-headers && test "$(grep -c INTERP /tmp/program-headers)" -eq 0
RUN readelf -d "${TARGET_PREFIX}/coredns" > /tmp/dynamic-section && test "$(grep -c '(NEEDED)' /tmp/dynamic-section)" -eq 0

RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/coredns" -version

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
