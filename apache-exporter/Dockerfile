ARG GO_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/out


FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS script-tools
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/verify-apache-exporter/ ./cmd/verify-apache-exporter/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/verify-apache-exporter ./cmd/verify-apache-exporter


FROM golang:${GO_VERSION}-alpine AS build
ARG APACHE_EXPORTER_VERSION
ARG TARGET_PREFIX
ARG TARGETARCH

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        git
ADD ".tmp/apache-exporter-${APACHE_EXPORTER_VERSION}.tar.gz" /build/
WORKDIR "/build/apache_exporter-${APACHE_EXPORTER_VERSION}"
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    mkdir -p ${TARGET_PREFIX}/bin && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -a -tags netgo -trimpath \
        -ldflags "-X github.com/prometheus/common/version.Version=${APACHE_EXPORTER_VERSION}" \
        -o ${TARGET_PREFIX}/bin/apache-exporter .


FROM registry.access.redhat.com/ubi10/ubi-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=script-tools /script-tools/verify-apache-exporter /usr/local/bin/
RUN /usr/local/bin/verify-apache-exporter


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 9117
ENTRYPOINT ["./bin/apache-exporter"]
