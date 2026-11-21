# syntax=docker/dockerfile:1
ARG GO_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/out

FROM golang:${GO_VERSION}-alpine AS build
ARG TARGETARCH
ARG NODE_EXPORTER_VERSION
ARG TARGET_PREFIX

WORKDIR /build
ADD ".cache/downloads/node-exporter/node-exporter-${NODE_EXPORTER_VERSION}.tar.gz" /build/
WORKDIR "/build/node_exporter-${NODE_EXPORTER_VERSION}"
RUN --mount=type=cache,id=node-exporter-go-build-${TARGETARCH},sharing=locked,target=/root/.cache/go-build \
    --mount=type=cache,id=node-exporter-go-modules-${TARGETARCH},sharing=locked,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH="${TARGETARCH}" GOTOOLCHAIN=local \
    go build -buildvcs=false -mod=readonly -trimpath \
        -ldflags "-X github.com/prometheus/common/version.Version=${NODE_EXPORTER_VERSION}" \
        -o "${TARGET_PREFIX}/bin/node_exporter" .
RUN install -m 0644 LICENSE NOTICE "${TARGET_PREFIX}"

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG NODE_EXPORTER_VERSION
ARG TARGET_PREFIX
RUN --mount=type=cache,id=node-exporter-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN test -x "${TARGET_PREFIX}/bin/node_exporter"
RUN file -b "${TARGET_PREFIX}/bin/node_exporter" | grep -Eq '^ELF .* (statically linked|static-pie linked)(,|$)'
RUN readelf -l "${TARGET_PREFIX}/bin/node_exporter" | ( ! grep -q INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/node_exporter" | ( ! grep -q '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace -qq -e trace=execve "${TARGET_PREFIX}/bin/node_exporter" --version 2>&1 \
    | grep -F "node_exporter, version ${NODE_EXPORTER_VERSION} "

FROM scratch
ARG TARGET_PREFIX
COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 9100
ENTRYPOINT ["./bin/node_exporter"]
