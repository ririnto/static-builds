# syntax=docker/dockerfile:1
ARG GOLANG_IMAGE_TAG
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM golang:${GOLANG_IMAGE_TAG} AS build
ARG TARGETARCH
ARG BUILD_TARGET=coredns
ARG COREDNS_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,id=coredns-apk-${TARGETARCH},sharing=locked,target=/var/cache/apk \
    apk add \
        git \
        make
ADD ".cache/downloads/coredns/coredns-${COREDNS_VERSION}.tar.gz" /build/
WORKDIR "/build/coredns-${COREDNS_VERSION}"
RUN --mount=type=cache,id=coredns-go-build-${TARGETARCH},sharing=locked,target=/root/.cache/go-build \
    --mount=type=cache,id=coredns-go-modules-${TARGETARCH},sharing=locked,target=/go/pkg/mod \
    make coredns \
        GOTOOLCHAIN=local \
        CGO_ENABLED=0 \
        BUILDOPTS="-v -trimpath"

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGETARCH
ARG BUILD_TARGET=coredns
ARG COREDNS_VERSION
ARG TARGET_PREFIX
RUN --mount=type=cache,id=coredns-dnf-${TARGETARCH},sharing=locked,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /build/coredns-${COREDNS_VERSION}/coredns ${TARGET_PREFIX}/coredns
RUN set -eu; \
    case "$TARGETARCH" in \
        amd64) machine='Advanced Micro Devices X86-64' ;; \
        arm64) machine='AArch64' ;; \
        arm) machine='ARM' ;; \
        386) machine='Intel 80386' ;; \
        ppc64le) machine='PowerPC64' ;; \
        s390x) machine='IBM S/390' ;; \
        riscv64) machine='RISC-V' ;; \
        *) machine= ;; \
    esac; \
    for binary in "${TARGET_PREFIX}/coredns"; do \
        test -f "$binary"; \
        test -x "$binary"; \
        header=$(LC_ALL=C readelf -h "$binary"); \
        printf '%s\n' "$header" | grep -E 'Type: +((EXEC)|(DYN)) '; \
        if [ -n "$machine" ]; then \
            printf '%s\n' "$header" | grep -E "Machine: +$machine$"; \
        fi; \
        description=$(LC_ALL=C file -b "$binary"); \
        printf '%s\n' "$description" | grep -E '^ELF .* (statically linked|static-pie linked)(,|$)'; \
        segments=$(LC_ALL=C readelf -l "$binary"); \
        dynamic=$(LC_ALL=C readelf -d "$binary"); \
        if printf '%s\n' "$segments" | grep -w INTERP; then \
            echo "Unexpected ELF interpreter: $binary" >&2; exit 1; \
        fi; \
        if printf '%s\n' "$dynamic" | grep -F '(NEEDED)'; then \
            echo "Unexpected shared library dependency: $binary" >&2; exit 1; \
        fi; \
    done
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/coredns" -version
RUN "${TARGET_PREFIX}/coredns" -plugins

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp
ENTRYPOINT ["./coredns"]
