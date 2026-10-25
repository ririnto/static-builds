ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS download
ARG VECTOR_VERSION
ARG TARGET_PREFIX

WORKDIR /download
ADD ".tmp/vector-${VECTOR_VERSION}-x86_64-unknown-linux-musl.tar.gz" /download/
RUN mkdir -p ${TARGET_PREFIX}/bin && \
    mv vector-x86_64-unknown-linux-musl/bin/vector ${TARGET_PREFIX}/bin/


FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=download ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY vector/vector-verify.sh /usr/local/bin/
RUN /usr/local/bin/vector-verify.sh


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 8686
ENTRYPOINT ["./bin/vector"]
CMD ["--config", "/etc/vector/vector.yaml"]
