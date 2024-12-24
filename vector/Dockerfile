# syntax=docker/dockerfile:1.4
ARG ALPINE_VERSION
ARG VECTOR_VERSION
ARG UBI9_MICRO_VERSION


FROM alpine:${ALPINE_VERSION} AS download
ARG VECTOR_VERSION

WORKDIR /download
ADD "src/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" /download/
RUN mkdir -p /usr/local/vector/bin && \
    mv vector-x86_64-unknown-linux-musl/bin/vector /usr/local/vector/bin/


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=download /usr/local/vector /usr/local/vector
RUN set -eu; \
    readelf -h /usr/local/vector/bin/vector | egrep 'Type:|Machine:'; \
    readelf -l /usr/local/vector/bin/vector | egrep 'INTERP|DYNAMIC' || true; \
    readelf -d /usr/local/vector/bin/vector || true; \
    file_output="$(file /usr/local/vector/bin/vector 2>&1 || true)"; \
    ldd_output="$(ldd /usr/local/vector/bin/vector 2>&1 || true)"; \
    echo "file output: $file_output"; \
    echo "ldd output: $ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary (allowing musl static PIE)." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace /usr/local/vector/bin/vector --version && \
    echo "=== Available Components ===" && /usr/local/vector/bin/vector list


FROM scratch

COPY --from=verify /usr/local/vector .
EXPOSE 8686
ENTRYPOINT ["./bin/vector"]
CMD ["--config", "/etc/vector/vector.yaml"]
