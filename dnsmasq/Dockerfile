# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody

FROM alpine:${ALPINE_VERSION} AS build
ARG DNSMASQ_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        coreutils \
        linux-headers
ADD ".cache/downloads/dnsmasq-${DNSMASQ_VERSION}.tar.gz" /build/
WORKDIR "/build/dnsmasq-${DNSMASQ_VERSION}"
RUN make "-j$(nproc)" \
    CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    LDFLAGS="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    PREFIX=${TARGET_PREFIX} \
    install

FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG TARGET_PREFIX
RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
ARG TARGETARCH
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
    for binary in "${TARGET_PREFIX}/sbin/dnsmasq"; do \
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
RUN timeout --kill-after=5s 30s strace "${TARGET_PREFIX}/sbin/dnsmasq" -v

FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 53 53/udp 67/udp
ENTRYPOINT ["./sbin/dnsmasq"]
CMD ["-k"]
