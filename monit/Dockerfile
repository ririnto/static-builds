ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG MONIT_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        linux-headers \
        openssl-dev \
        openssl-libs-static \
        zlib-dev \
        zlib-static \
        flex \
        byacc
ADD ".tmp/monit-${MONIT_VERSION}.tar.gz" /build/
WORKDIR "/build/monit-${MONIT_VERSION}"
RUN ./configure \
    --prefix=${TARGET_PREFIX} \
    --without-pam \
    --with-largefiles
RUN make "-j$(nproc)"
RUN mkdir -p ${TARGET_PREFIX}/bin && \
    find src -name '*.o' -not -path '*/.*' | xargs ar rcs libsrc.a && \
    find libmonit/src -name '*.o' -not -path '*/.*' | xargs ar rcs liblibmonit.a && \
    gcc -static -static-pie -o ${TARGET_PREFIX}/bin/monit \
        -Wl,--whole-archive libsrc.a liblibmonit.a -Wl,--no-whole-archive \
        /usr/lib/libssl.a /usr/lib/libcrypto.a /usr/lib/libz.a \
        -lpthread -lresolv -ldl -lm


FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY monit/monit-verify.sh /usr/local/bin/
RUN /usr/local/bin/monit-verify.sh


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 2812
ENTRYPOINT ["./bin/monit"]
CMD ["-I"]
