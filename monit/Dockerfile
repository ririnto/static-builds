# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG CENTOS_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG TARGETARCH
ARG BUILD_TARGET=monit
ARG MONIT_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,id=monit-apk-${TARGETARCH},sharing=locked,target=/var/cache/apk \
    apk add \
        build-base \
        linux-headers \
        openssl-dev \
        openssl-libs-static \
        zlib-dev \
        zlib-static \
        flex \
        byacc
ADD ".cache/downloads/monit/monit-${MONIT_VERSION}.tar.gz" /build/
WORKDIR "/build/monit-${MONIT_VERSION}"
RUN ./configure \
    --prefix=${TARGET_PREFIX} \
    --without-pam \
    --with-largefiles
RUN make "-j$(nproc)"
RUN mkdir -p ${TARGET_PREFIX}/bin
RUN find src -name '*.o' -not -path '*/.*' -exec ar rcs libsrc.a {} +
RUN find libmonit/src -name '*.o' -not -path '*/.*' -exec ar rcs liblibmonit.a {} +
RUN gcc -static -static-pie -o ${TARGET_PREFIX}/bin/monit \
        -Wl,--whole-archive libsrc.a liblibmonit.a -Wl,--no-whole-archive \
        /usr/lib/libssl.a /usr/lib/libcrypto.a /usr/lib/libz.a \
        -lpthread -lresolv -ldl -lm

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG BUILD_TARGET=monit
ARG TARGET_PREFIX
RUN --mount=type=cache,id=monit-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN test -x "${TARGET_PREFIX}/bin/monit"
RUN file -b "${TARGET_PREFIX}/bin/monit" | grep '^ELF '
RUN readelf -l "${TARGET_PREFIX}/bin/monit" | ( ! grep INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/monit" | ( ! grep '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/monit" -V

FROM centos:${CENTOS_VERSION} AS verify-centos
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
ARG TARGETARCH
ARG CENTOS_VERSION
ARG TARGET_PREFIX
RUN sed -ri '/^\[base\]/,/^\[/s/^mirrorlist=/#mirrorlist=/' /etc/yum.repos.d/CentOS-Base.repo
RUN sed -ri '/^\[base\]/,/^\[/s|^#?baseurl=https?://mirror\.centos\.org/(centos/)?(altarch/)?\$releasever/|baseurl=https://vault.centos.org/\2'"${CENTOS_VERSION}"'/|' /etc/yum.repos.d/CentOS-Base.repo
RUN --mount=type=cache,id=monit-yum-${TARGETARCH},sharing=locked,target=/var/cache/yum \
    yum -y --disablerepo='*' --enablerepo=base --setopt=base.gpgcheck=1 install file binutils strace
COPY --from=verify ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN test -x "${TARGET_PREFIX}/bin/monit"
RUN file -b "${TARGET_PREFIX}/bin/monit" | grep '^ELF '
RUN LC_ALL=C readelf -h "${TARGET_PREFIX}/bin/monit" | grep -E '^[[:space:]]*Type:[[:space:]]+(EXEC|DYN)([[:space:]]|$)'
RUN readelf -l "${TARGET_PREFIX}/bin/monit" | ( ! grep INTERP )
RUN readelf -d "${TARGET_PREFIX}/bin/monit" | ( ! grep '(NEEDED)' )
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/bin/monit" -V

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify-centos ${TARGET_PREFIX} .
EXPOSE 2812
ENTRYPOINT ["./bin/monit"]
CMD ["-I"]
