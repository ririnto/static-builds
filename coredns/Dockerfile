ARG GOLANG_IMAGE_TAG
ARG ALMALINUX10_MINIMAL_TAG
ARG TARGET_PREFIX=/home/nobody


FROM golang:${GOLANG_IMAGE_TAG} AS build
ARG COREDNS_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        git \
        make
ADD ".tmp/coredns-${COREDNS_VERSION}.tar.gz" /build/
WORKDIR "/build/coredns-${COREDNS_VERSION}"
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    make coredns \
        GOTOOLCHAIN=local \
        CGO_ENABLED=0 \
        BUILDOPTS="-v -trimpath"


FROM almalinux:${ALMALINUX10_MINIMAL_TAG} AS verify
ARG COREDNS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /build/coredns-${COREDNS_VERSION}/coredns ${TARGET_PREFIX}/coredns
COPY coredns/coredns-verify.sh /usr/local/bin/
RUN /usr/local/bin/coredns-verify.sh


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
