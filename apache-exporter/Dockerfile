ARG GO_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/out


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


FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h ${TARGET_PREFIX}/bin/apache-exporter | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l ${TARGET_PREFIX}/bin/apache-exporter | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d ${TARGET_PREFIX}/bin/apache-exporter || true; \
    file_output="$(file ${TARGET_PREFIX}/bin/apache-exporter 2>&1 || true)"; \
    ldd_output="$(ldd ${TARGET_PREFIX}/bin/apache-exporter 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace ${TARGET_PREFIX}/bin/apache-exporter --version


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 9117
ENTRYPOINT ["./bin/apache-exporter"]
