ARG FUNCTIONAL_BASE_IMAGE

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS script-tools
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/healthcheck-integration/ ./cmd/healthcheck-integration/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/healthcheck-integration ./cmd/healthcheck-integration

FROM ${FUNCTIONAL_BASE_IMAGE} AS functional

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install bash coreutils-single curl dnsmasq grep openssl sed socat util-linux
RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf clean all

WORKDIR /suite
COPY --from=script-tools /script-tools/healthcheck-integration /usr/local/bin/
COPY proxy/tests/fixtures/ ./proxy/tests/fixtures/
ENTRYPOINT ["/usr/local/bin/healthcheck-integration"]
