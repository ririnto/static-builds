ARG ALPINE_VERSION
ARG UBI9_MICRO_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS build
ARG HAPROXY_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        linux-headers \
        lua5.4-dev \
        openssl-dev \
        openssl-libs-static \
        pcre2-dev \
        pcre2-static
ADD "src/haproxy-${HAPROXY_VERSION}.tar.gz" /build/
WORKDIR "/build/haproxy-${HAPROXY_VERSION}"
RUN make "-j$(nproc)" \
    TARGET=linux-musl \
    USE_EPOLL=1 \
    # USE_KQUEUE \
    # USE_EVPORTS \
    USE_NETFILTER=1 \
    # USE_PCRE \
    # USE_PCRE_JIT \
    USE_PCRE2=1 \
    USE_PCRE2_JIT=1 \
    USE_POLL=1 \
    USE_THREAD=0 \
    # USE_STATIC_PCRE \
    # USE_STATIC_PCRE2 \
    USE_TPROXY=1 \
    USE_LINUX_TPROXY=1 \
    USE_LINUX_SPLICE=1 \
    USE_LINUX_CAP=1 \
    USE_LIBCRYPT=1 \
    USE_CRYPT_H=1 \
    USE_GETADDRINFO=1 \
    USE_OPENSSL=1 \
    # USE_OPENSSL_AWSLC \
    # USE_OPENSSL_WOLFSSL \
    USE_QUIC=1 \
    USE_QUIC_OPENSSL_COMPAT=1 \
    USE_ENGINE=1 \
    USE_LUA=1 \
    USE_ACCEPT4=1 \
    # USE_CLOSEFROM \
    USE_PRCTL=1 \
    # USE_PROCCTL \
    # USE_ZLIB \
    # USE_SLZ \
    USE_CPU_AFFINITY=1 \
    USE_TFO=1 \
    USE_NS=1 \
    USE_DL=1 \
    USE_MATH=1 \
    USE_RT=1 \
    USE_BACKTRACE=1 \
    USE_PROMEX=1 \
    # USE_DEVICEATLAS \
    # USE_51DEGREES \
    # USE_WURFL \
    # USE_OBSOLETE_LINKER \
    # USE_THREAD_DUMP \
    # USE_OT \
    # USE_MEMORY_PROFILING \
    USE_LIBATOMIC=1 \
    USE_PTHREAD_EMULATION=1 \
    # CC \
    # LD \
    OPT_CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    # CFLAGS \
    LDFLAGS="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    # ARCH_FLAGS \
    # DEP \
    # DEBUG \
    # ERR \
    # FAILFAST \
    # WARN_CFLAGS \
    # NOWARN_CFLAGS \
    # ADDINC \
    # ADDLIB \
    # DEFINE \
    # SILENT_DEFINE \
    # EXTRA \
    # INSTALL \
    # HLUA_PREPEND_PATH \
    # HLUA_PREPEND_CPATH \
    # PCRE_CONFIG \
    # PCREDIR \
    # PCRE_LIB \
    # PCRE_INC \
    # PCRE2_CONFIG \
    SSL_LIB=/usr/lib \
    SSL_INC=/usr/include/openssl \
    LUA_LIB=/usr/lib/lua5.4 \
    LUA_INC=/usr/include/lua5.4 \
    # LUA_LIB_NAME \
    # OT_DEBUG \
    # OT_INC \
    # OT_LIB \
    # OT_RUNPATH \
    # OT_USE_VARS \
    # IGNOREGIT \
    # VERSION \
    # SUBVERS \
    # EXTRAVERSION \
    # VERDATE \
    # VTEST_PROGRAM \
    # DEBUG_USE_ABORT \
    # PCRE2DIR=/usr \
    # ZLIB_LIB=/lib \
    # ZLIB_INC=/usr/include \
    ;
RUN make install \
    # DESTDIR \
    PREFIX=${TARGET_PREFIX} \
    # SBINDIR \
    # MANDIR \
    # DOCDIR \
    ;


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h ${TARGET_PREFIX}/sbin/haproxy | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l ${TARGET_PREFIX}/sbin/haproxy | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d ${TARGET_PREFIX}/sbin/haproxy || true; \
    file_output="$(file ${TARGET_PREFIX}/sbin/haproxy 2>&1 || true)"; \
    ldd_output="$(ldd ${TARGET_PREFIX}/sbin/haproxy 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace ${TARGET_PREFIX}/sbin/haproxy -vv


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/haproxy"]
