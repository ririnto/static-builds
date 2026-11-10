ARG GOLANG_IMAGE_TAG
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS script-tools
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/verify-coredns/ ./cmd/verify-coredns/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/verify-coredns ./cmd/verify-coredns


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


FROM registry.access.redhat.com/ubi10/ubi-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG COREDNS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /build/coredns-${COREDNS_VERSION}/coredns ${TARGET_PREFIX}/coredns
COPY --from=script-tools /script-tools/verify-coredns /usr/local/bin/
RUN /usr/local/bin/verify-coredns


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
