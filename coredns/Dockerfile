ARG GOLANG_IMAGE_TAG
ARG UBI9_MICRO_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM golang:${GOLANG_IMAGE_TAG} AS build
ARG COREDNS_VERSION
ARG TARGET_PREFIX

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
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /build/coredns-${COREDNS_VERSION}/coredns ${TARGET_PREFIX}/coredns
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h ${TARGET_PREFIX}/coredns | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l ${TARGET_PREFIX}/coredns | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d ${TARGET_PREFIX}/coredns || true; \
    file_output="$(file ${TARGET_PREFIX}/coredns 2>&1 || true)"; \
    ldd_output="$(ldd ${TARGET_PREFIX}/coredns 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace ${TARGET_PREFIX}/coredns -version && \
    echo "=== Installed Plugins ===" && ${TARGET_PREFIX}/coredns -plugins


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
