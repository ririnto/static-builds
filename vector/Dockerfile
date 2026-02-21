ARG ALPINE_VERSION
ARG UBI9_MICRO_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS download
ARG VECTOR_VERSION
ARG TARGET_PREFIX

WORKDIR /download
ADD "src/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" /download/
RUN mkdir -p ${TARGET_PREFIX}/bin && \
    mv vector-x86_64-unknown-linux-musl/bin/vector ${TARGET_PREFIX}/bin/


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=download ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN set -eu; \
    readelf -h ${TARGET_PREFIX}/bin/vector | egrep 'Type:|Machine:'; \
    readelf -l ${TARGET_PREFIX}/bin/vector | egrep 'INTERP|DYNAMIC' || true; \
    readelf -d ${TARGET_PREFIX}/bin/vector || true; \
    file_output="$(file ${TARGET_PREFIX}/bin/vector 2>&1 || true)"; \
    ldd_output="$(ldd ${TARGET_PREFIX}/bin/vector 2>&1 || true)"; \
    echo "file output: $file_output"; \
    echo "ldd output: $ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary (allowing musl static PIE)." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace ${TARGET_PREFIX}/bin/vector --version && \
    echo "=== Available Components ===" && ${TARGET_PREFIX}/bin/vector list


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 8686
ENTRYPOINT ["./bin/vector"]
CMD ["--config", "/etc/vector/vector.yaml"]
