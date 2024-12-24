# syntax=docker/dockerfile:1.4
ARG GOLANG_IMAGE_TAG
ARG COREDNS_VERSION
ARG UBI9_MICRO_VERSION


FROM golang:${GOLANG_IMAGE_TAG} AS build
ARG COREDNS_VERSION

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        git \
        make
ADD "src/coredns-${COREDNS_VERSION}.tar.gz" /build/
WORKDIR "/build/coredns-${COREDNS_VERSION}"
RUN make coredns \
        CGO_ENABLED=0 \
        BUILDOPTS="-v -trimpath"


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify
ARG COREDNS_VERSION

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /build/coredns-${COREDNS_VERSION}/coredns /usr/local/coredns/coredns
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h /usr/local/coredns/coredns | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l /usr/local/coredns/coredns | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d /usr/local/coredns/coredns || true; \
    file_output="$(file /usr/local/coredns/coredns 2>&1 || true)"; \
    ldd_output="$(ldd /usr/local/coredns/coredns 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace /usr/local/coredns/coredns -version && \
    echo "=== Installed Plugins ===" && /usr/local/coredns/coredns -plugins


FROM scratch

COPY --from=verify /usr/local/coredns .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
