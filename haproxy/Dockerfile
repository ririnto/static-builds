ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody
ARG HARDENING_CFLAGS="-O2 -pipe -fPIC -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -ftrivial-auto-var-init=zero"


FROM alpine:${ALPINE_VERSION} AS otel-base
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        ca-certificates \
        cmake \
        git \
        linux-headers \
        openssl-dev \
        openssl-libs-static \
        patch \
        perl \
        pkgconf \
        python3 \
        zlib-dev \
        zlib-static


FROM otel-base AS otel-abseil
ARG ABSEIL_CPP_VERSION
ARG HARDENING_CFLAGS
ADD ".tmp/abseil-cpp-${ABSEIL_CPP_VERSION}.tar.gz" /tmp/
RUN cmake -S "/tmp/abseil-cpp-${ABSEIL_CPP_VERSION}" -B /tmp/abseil-build \
        -DCMAKE_INSTALL_PREFIX=/opt \
        -DCMAKE_CXX_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -DABSL_ENABLE_INSTALL=ON \
        -DABSL_PROPAGATE_CXX_STD=ON \
    && cmake --build /tmp/abseil-build --parallel "$(nproc)" \
    && cmake --install /tmp/abseil-build


FROM otel-base AS otel-protobuf
ARG PROTOBUF_VERSION
ARG HARDENING_CFLAGS
ADD ".tmp/protobuf-${PROTOBUF_VERSION}.tar.gz" /tmp/
COPY --from=otel-abseil /opt /opt
RUN cmake -S "/tmp/protobuf-${PROTOBUF_VERSION}" -B /tmp/protobuf-build \
        -DCMAKE_INSTALL_PREFIX=/opt \
        -DCMAKE_CXX_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -Dprotobuf_BUILD_TESTS=OFF \
        -Dprotobuf_ABSL_PROVIDER=package \
        -DZLIB_LIBRARY=/usr/lib/libz.a \
        -DZLIB_INCLUDE_DIR=/usr/include \
    && cmake --build /tmp/protobuf-build --parallel "$(nproc)" \
    && cmake --install /tmp/protobuf-build


FROM otel-base AS otel-rapidyaml
ARG RAPIDYAML_VERSION
ARG HARDENING_CFLAGS
ADD ".tmp/rapidyaml-${RAPIDYAML_VERSION}-src.tgz" /tmp/
RUN cmake -S "/tmp/rapidyaml-${RAPIDYAML_VERSION}-src" -B /tmp/rapidyaml-build \
        -DCMAKE_INSTALL_PREFIX=/opt \
        -DCMAKE_C_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_CXX_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -DRYML_BUILD_TOOLS=OFF \
        -DRYML_BUILD_TESTS=OFF \
        -DRYML_BUILD_BENCHMARKS=OFF \
        -DRYML_BUILD_EXAMPLES=OFF \
        -DRYML_BUILD_C4C_TESTING=OFF \
        -DGIT=/bin/true \
    && cmake --build /tmp/rapidyaml-build --parallel "$(nproc)" \
    && cmake --install /tmp/rapidyaml-build


FROM otel-base AS otel-re2
ARG RE2_VERSION
ARG HARDENING_CFLAGS
ADD ".tmp/re2-${RE2_VERSION}.tar.gz" /tmp/
COPY --from=otel-abseil /opt /opt
RUN cmake -S "/tmp/re2-${RE2_VERSION}" -B /tmp/re2-build \
        -DCMAKE_INSTALL_PREFIX=/opt \
        -DCMAKE_C_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_CXX_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -DRE2_BUILD_TESTING=OFF \
    && cmake --build /tmp/re2-build --parallel "$(nproc)" \
    && cmake --install /tmp/re2-build


FROM otel-base AS otel-grpc
ARG GRPC_VERSION
ARG HARDENING_CFLAGS
ADD ".tmp/grpc-${GRPC_VERSION}.tar.gz" /tmp/
COPY --from=otel-abseil /opt /opt
COPY --from=otel-protobuf /opt /opt
COPY --from=otel-re2 /opt /opt
RUN apk add c-ares-dev curl-dev curl-static brotli-static libidn2-static libpsl-static libunistring-static nghttp2-static zstd-static \
    && cmake -S "/tmp/grpc-${GRPC_VERSION}" -B /tmp/grpc-build \
        -DCMAKE_INSTALL_PREFIX=/opt \
        -DCMAKE_C_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_CXX_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DgRPC_BUILD_TESTS=OFF \
        -DgRPC_INSTALL=ON \
        -DgRPC_ZLIB_PROVIDER=package \
        -DgRPC_CARES_PROVIDER=package \
        -DgRPC_SSL_PROVIDER=package \
        -DgRPC_PROTOBUF_PROVIDER=package \
        -DgRPC_ABSL_PROVIDER=package \
        -DgRPC_RE2_PROVIDER=package \
        -DgRPC_BUILD_GRPC_CPP_PLUGIN=ON \
        -DgRPC_BUILD_GRPC_CSHARP_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_OBJECTIVE_C_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_PHP_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_NODE_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_PYTHON_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_RUBY_PLUGIN=OFF \
        -DgRPC_BUILD_GRPCPP_OTEL_PLUGIN=OFF \
        -DOPENSSL_ROOT_DIR=/usr \
        -DOPENSSL_SSL_LIBRARY=/usr/lib/libssl.a \
        -DOPENSSL_CRYPTO_LIBRARY=/usr/lib/libcrypto.a \
        -DZLIB_LIBRARY=/usr/lib/libz.a \
        -DZLIB_INCLUDE_DIR=/usr/include \
        -DCMAKE_PREFIX_PATH=/opt:/usr \
    && cmake --build /tmp/grpc-build --parallel "$(nproc)" \
    && cmake --install /tmp/grpc-build


FROM otel-base AS otel-cpp
ARG HAPROXY_OTEL_C_WRAPPER_VERSION
ARG OPENTELEMETRY_CPP_VERSION
ARG OPENTELEMETRY_PROTO_VERSION
ARG HARDENING_CFLAGS
ADD ".tmp/opentelemetry-cpp-${OPENTELEMETRY_CPP_VERSION}.tar.gz" /tmp/
ADD ".tmp/opentelemetry-proto-${OPENTELEMETRY_PROTO_VERSION}.tar.gz" /tmp/
ADD ".tmp/opentelemetry-c-wrapper-${HAPROXY_OTEL_C_WRAPPER_VERSION}.tar.gz" /build/
COPY --from=otel-abseil /opt /opt
COPY --from=otel-protobuf /opt /opt
COPY --from=otel-rapidyaml /opt /opt
COPY --from=otel-grpc /opt /opt
RUN apk add c-ares-dev curl-dev curl-static brotli-static libidn2-static libpsl-static libunistring-static nghttp2-static zstd-static \
    && mv "/build/opentelemetry-c-wrapper-${HAPROXY_OTEL_C_WRAPPER_VERSION}" /build/opentelemetry-c-wrapper \
    && cd "/tmp/opentelemetry-cpp-${OPENTELEMETRY_CPP_VERSION}" \
    && for patch_file in /build/opentelemetry-c-wrapper/scripts/build/000*-opentelemetry-cpp-*.patch; do patch -p1 < "${patch_file}"; done \
    && PKG_CONFIG_PATH=/opt/lib/pkgconfig cmake -S "/tmp/opentelemetry-cpp-${OPENTELEMETRY_CPP_VERSION}" -B build \
        -DCMAKE_INSTALL_PREFIX=/opt \
        -DCMAKE_C_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_CXX_FLAGS="${HARDENING_CFLAGS}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -DBUILD_PACKAGE=OFF \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DCMAKE_CXX_STANDARD=14 \
        -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
        -DOTELCPP_PROTO_PATH="/tmp/opentelemetry-proto-${OPENTELEMETRY_PROTO_VERSION}" \
        -DOPENSSL_USE_STATIC_LIBS=TRUE \
        -DOPENSSL_ROOT_DIR=/usr \
        -DOPENSSL_SSL_LIBRARY=/usr/lib/libssl.a \
        -DOPENSSL_CRYPTO_LIBRARY=/usr/lib/libcrypto.a \
        -DCMAKE_PREFIX_PATH=/opt \
        -DCURL_LIBRARY=/usr/lib/libcurl.a \
        -DCURL_INCLUDE_DIR=/usr/include \
        -DZLIB_LIBRARY=/usr/lib/libz.a \
        -DZLIB_INCLUDE_DIR=/usr/include \
        -DWITH_ABI_VERSION_1=OFF \
        -DWITH_ABI_VERSION_2=ON \
        -DWITH_CONFIGURATION=ON \
        -DOPENTELEMETRY_INSTALL=ON \
        -DWITH_NO_DEPRECATED_CODE=ON \
        -DWITH_OTLP_GRPC=ON \
        -DWITH_OTLP_HTTP=ON \
        -DWITH_OTLP_FILE=ON \
        -DWITH_OTLP_HTTP_COMPRESSION=ON \
        -DWITH_ZIPKIN=ON \
        -DWITH_ELASTICSEARCH=ON \
        -DOTELCPP_VERSIONED_LIBS=OFF \
        -DWITH_ASYNC_EXPORT_PREVIEW=OFF \
        -DWITH_THREAD_INSTRUMENTATION_PREVIEW=OFF \
        -DWITH_METRICS_EXEMPLAR_PREVIEW=OFF \
        -DWITH_OPENTRACING=OFF \
        -DWITH_BENCHMARK=OFF \
        -DWITH_EXAMPLES=OFF \
        -DWITH_FUNC_TESTS=OFF \
        -DBUILD_TESTING=OFF \
    && cmake --build build --parallel "$(nproc)" \
    && cmake --install build \
    && cd /build/opentelemetry-c-wrapper \
    && PKG_CONFIG_PATH=/opt/lib/pkgconfig:/usr/lib/pkgconfig cmake -S . -B build \
        -DCMAKE_INSTALL_PREFIX=/opt \
        -DOPENTELEMETRY_DIR=/opt \
        -DBUILD_STATIC=ON \
        -DENABLE_THREADS=OFF \
        -DWITH_RAPIDYAML=ON \
        -DWITH_LIBFYAML=OFF \
    && cmake --build build --parallel "$(nproc)" \
    && cmake --install build


FROM alpine:${ALPINE_VERSION} AS build
ARG HAPROXY_VERSION
ARG HAPROXY_OTEL_ADDON_REF
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        brotli-static \
        c-ares-dev \
        curl-dev \
        curl-static \
        libidn2-static \
        libpsl-static \
        libunistring-static \
        nghttp2-static \
        zstd-static \
        linux-headers \
        lua5.4-dev \
        openssl-dev \
        openssl-libs-static \
        patch \
        pcre2-dev \
        pcre2-static \
        zlib-static
ADD ".tmp/haproxy-${HAPROXY_VERSION}.tar.gz" /build/
ADD ".tmp/haproxy-opentelemetry-${HAPROXY_OTEL_ADDON_REF}.tar.gz" /build/
COPY --from=otel-cpp /opt /opt
COPY haproxy/thread-use-thread-guard.patch /build/thread-use-thread-guard.patch
WORKDIR "/build/haproxy-${HAPROXY_VERSION}"
RUN patch -p1 < /build/thread-use-thread-guard.patch
RUN PKG_CONFIG_PATH=/opt/lib/pkgconfig make "-j$(nproc)" \
    TARGET=linux-musl \
    EXTRA_MAKE="/build/haproxy-opentelemetry-${HAPROXY_OTEL_ADDON_REF}" \
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
    USE_SHM_OPEN=1 \
    USE_LIBCRYPT=1 \
    USE_CRYPT_H=1 \
    USE_GETADDRINFO=1 \
    USE_OPENSSL=1 \
    # USE_OPENSSL_AWSLC \
    # USE_OPENSSL_WOLFSSL \
    USE_KTLS=1 \
    # USE_ECH \
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
    # USE_TRACE \
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
    ADDLIB="$(pkg-config --libs --static libcurl)" \
    # ADDLIB \
    # DEFINE \
    # SILENT_DEFINE \
    # EXTRA \
    # INSTALL \
    # HLUA_PREPEND_PATH \
    # HLUA_PREPEND_CPATH \
    # OTEL_DEBUG \
    # OTEL_INC \
    # OTEL_LIB \
    # OTEL_RUNPATH \
    OTEL_STATIC=1 \
    OTEL_USE_VARS=1 \
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


FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
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
