ARG ALPINE_VERSION
ARG UBI9_MICRO_VERSION
ARG TARGET_PREFIX=/home/nobody
ARG OTEL_PREFIX=/opt/otel


FROM alpine:${ALPINE_VERSION} AS otel-deps-build
ARG ABSEIL_CPP_VERSION
ARG PROTOBUF_VERSION
ARG RE2_VERSION
ARG GRPC_VERSION
ARG OTEL_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        c-ares-dev \
        cmake \
        linux-headers \
        nlohmann-json \
        openssl-dev \
        openssl-libs-static \
        pkgconfig \
        zlib-dev \
        zlib-static
ADD ".tmp/abseil-cpp-${ABSEIL_CPP_VERSION}.tar.gz" /build/
RUN cd "/build/abseil-cpp-${ABSEIL_CPP_VERSION}" && \
    mkdir build && cd build && \
    cmake \
        -DCMAKE_INSTALL_PREFIX="${OTEL_PREFIX}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_CXX_STANDARD=17 \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DBUILD_SHARED_LIBS=OFF \
        -DABSL_PROPAGATE_CXX_STD=ON \
        -DABSL_USE_EXTERNAL_GOOGLETEST=OFF \
        -DABSL_BUILD_TESTING=OFF \
        -DABSL_ENABLE_INSTALL=ON \
        .. && \
    cmake --build . --parallel "$(nproc)" --target install
ADD ".tmp/protobuf-${PROTOBUF_VERSION}.tar.gz" /build/
RUN cd "/build/protobuf-${PROTOBUF_VERSION}" && \
    mkdir build && cd build && \
    cmake \
        -DCMAKE_INSTALL_PREFIX="${OTEL_PREFIX}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_CXX_STANDARD=17 \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DCMAKE_PREFIX_PATH="${OTEL_PREFIX}" \
        -DBUILD_SHARED_LIBS=OFF \
        -Dprotobuf_BUILD_TESTS=OFF \
        -Dprotobuf_BUILD_LIBPROTOC=ON \
        -Dprotobuf_BUILD_PROTOC_BINARIES=ON \
        -Dprotobuf_ABSL_PROVIDER=package \
        -Dprotobuf_INSTALL=ON \
        .. && \
    cmake --build . --parallel "$(nproc)" --target install
ADD ".tmp/re2-${RE2_VERSION}.tar.gz" /build/
RUN cd "/build/re2-${RE2_VERSION}" && \
    mkdir build && cd build && \
    cmake \
        -DCMAKE_INSTALL_PREFIX="${OTEL_PREFIX}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_CXX_STANDARD=17 \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DCMAKE_PREFIX_PATH="${OTEL_PREFIX}" \
        -DBUILD_SHARED_LIBS=OFF \
        -DRE2_BUILD_TESTING=OFF \
        .. && \
    cmake --build . --parallel "$(nproc)" --target install
ADD ".tmp/grpc-${GRPC_VERSION}.tar.gz" /build/
RUN cd "/build/grpc-${GRPC_VERSION}" && \
    mkdir build && cd build && \
    cmake \
        -DCMAKE_INSTALL_PREFIX="${OTEL_PREFIX}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_CXX_STANDARD=17 \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DCMAKE_PREFIX_PATH="${OTEL_PREFIX}" \
        -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
        -DFETCHCONTENT_FULLY_DISCONNECTED=ON \
        -DBUILD_SHARED_LIBS=OFF \
        -DgRPC_INSTALL=ON \
        -DgRPC_BUILD_TESTS=OFF \
        -DgRPC_BUILD_GRPC_CPP_PLUGIN=ON \
        -DgRPC_BUILD_GRPC_CSHARP_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_NODE_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_OBJECTIVE_C_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_PHP_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_PYTHON_PLUGIN=OFF \
        -DgRPC_BUILD_GRPC_RUBY_PLUGIN=OFF \
        -DgRPC_BUILD_GRPCPP_OTEL_PLUGIN=OFF \
        -DgRPC_ABSL_PROVIDER=package \
        -DgRPC_CARES_PROVIDER=package \
        -DgRPC_PROTOBUF_PROVIDER=package \
        -DgRPC_PROTOBUF_PACKAGE_TYPE=CONFIG \
        -DgRPC_RE2_PROVIDER=package \
        -DgRPC_SSL_PROVIDER=package \
        -DgRPC_ZLIB_PROVIDER=package \
        .. && \
    cmake --build . --parallel "$(nproc)" --target install


FROM otel-deps-build AS otel-cpp-build
ARG OTEL_C_WRAPPER_VERSION
ARG OTEL_CPP_VERSION
ARG OTEL_PROTO_VERSION
ARG MS_GSL_VERSION
ARG RAPIDYAML_VERSION
ARG OTEL_PREFIX

RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        curl-dev \
        curl-static \
        nghttp2-static \
        patch
ADD ".tmp/opentelemetry-c-wrapper-${OTEL_C_WRAPPER_VERSION}.tar.gz" /build/
ADD ".tmp/opentelemetry-cpp-${OTEL_CPP_VERSION}.tar.gz" /build/
ADD ".tmp/opentelemetry-proto-${OTEL_PROTO_VERSION}.tar.gz" /build/
ADD ".tmp/ms-gsl-${MS_GSL_VERSION}.tar.gz" /build/
ADD ".tmp/rapidyaml-${RAPIDYAML_VERSION}-src.tgz" /build/
WORKDIR "/build/opentelemetry-cpp-${OTEL_CPP_VERSION}"
RUN set -eu; \
    wrapper_dir="/build/opentelemetry-c-wrapper-${OTEL_C_WRAPPER_VERSION}"; \
    for patch_file in "${wrapper_dir}/scripts/build/"*-opentelemetry-cpp-${OTEL_CPP_VERSION}.patch; do \
        echo "Applying ${patch_file}"; \
        patch -p1 < "${patch_file}"; \
    done
RUN mkdir build && cd build && \
    cmake \
        -DCMAKE_INSTALL_PREFIX="${OTEL_PREFIX}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DCMAKE_CXX_STANDARD=17 \
        -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
        -DCMAKE_PREFIX_PATH="${OTEL_PREFIX}" \
        -DFETCHCONTENT_FULLY_DISCONNECTED=ON \
        -DBUILD_SHARED_LIBS=OFF \
        -DOPENTELEMETRY_INSTALL=ON \
        -DOTELCPP_VERSIONED_LIBS=OFF \
        -DWITH_ABI_VERSION_1=OFF \
        -DWITH_ABI_VERSION_2=ON \
        -DWITH_NO_DEPRECATED_CODE=ON \
        -DWITH_CONFIGURATION=ON \
        -DWITH_OTLP_GRPC=ON \
        -DWITH_OTLP_HTTP=ON \
        -DWITH_OTLP_HTTP_COMPRESSION=ON \
        -DWITH_OTLP_FILE=ON \
        -DWITH_ZIPKIN=ON \
        -DWITH_ELASTICSEARCH=OFF \
        -DWITH_PROMETHEUS=OFF \
        -DWITH_OPENTRACING=OFF \
        -DWITH_BENCHMARK=OFF \
        -DWITH_EXAMPLES=OFF \
        -DWITH_FUNC_TESTS=OFF \
        -DBUILD_TESTING=OFF \
        -DWITH_ASYNC_EXPORT_PREVIEW=ON \
        -DWITH_THREAD_INSTRUMENTATION_PREVIEW=ON \
        -DWITH_METRICS_EXEMPLAR_PREVIEW=ON \
        -DOPENSSL_USE_STATIC_LIBS=TRUE \
        -DCURL_USE_STATIC_LIBS=ON \
        "-DOTELCPP_PROTO_PATH=/build/opentelemetry-proto-${OTEL_PROTO_VERSION}" \
        "-DFETCHCONTENT_SOURCE_DIR_GSL=/build/GSL-${MS_GSL_VERSION}" \
        "-DFETCHCONTENT_SOURCE_DIR_RYML=/build/rapidyaml-${RAPIDYAML_VERSION}-src" \
        .. && \
    cmake --build . --parallel "$(nproc)" --target install


FROM otel-cpp-build AS otel-wrapper-build
ARG OTEL_C_WRAPPER_VERSION
ARG OTEL_PREFIX

WORKDIR "/build/opentelemetry-c-wrapper-${OTEL_C_WRAPPER_VERSION}"
RUN sed -i 's/SHARED ${_lib_sources}/STATIC ${_lib_sources}/' CMakeLists.txt
RUN mkdir build && cd build && \
    PKG_CONFIG_PATH="${OTEL_PREFIX}/lib/pkgconfig" \
    cmake \
        -DCMAKE_INSTALL_PREFIX="${OTEL_PREFIX}" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DCMAKE_CXX_STANDARD=17 \
        -DCMAKE_CXX_EXTENSIONS=ON \
        -DCMAKE_PREFIX_PATH="${OTEL_PREFIX}" \
        -DBUILD_SHARED_LIBS=OFF \
        -DOPENTELEMETRY_DIR="${OTEL_PREFIX}" \
        -DWITH_RAPIDYAML=ON \
        -DWITH_LIBFYAML=OFF \
        .. && \
    cmake --build . --parallel "$(nproc)" --target install
RUN set -eu; \
    libs="$(cd ${OTEL_PREFIX}/lib && ls *.a | sed 's/^lib\(.*\)\.a$/-l\1/' | tr '\n' ' ')"; \
    cares_libs="$(pkg-config --static --libs libcares)"; \
    curl_libs="$(pkg-config --static --libs libcurl)"; \
    ssl_libs="$(pkg-config --static --libs libssl libcrypto)"; \
    z_libs="$(pkg-config --static --libs zlib)"; \
    { \
        echo "prefix=${OTEL_PREFIX}"; \
        echo 'exec_prefix=${prefix}'; \
        echo 'libdir=${exec_prefix}/lib'; \
        echo 'includedir=${prefix}/include'; \
        echo ''; \
        echo 'Name: opentelemetry-c-wrapper'; \
        echo 'Description: HAProxy OpenTelemetry C wrapper (static)'; \
        echo "Version: ${OTEL_C_WRAPPER_VERSION}"; \
        echo ''; \
        echo 'Cflags: -I${includedir}'; \
        echo "Libs: -L\${libdir} -Wl,--start-group ${libs} -Wl,--end-group ${cares_libs} ${curl_libs} ${ssl_libs} ${z_libs} -lstdc++ -lpthread -lm"; \
    } > "${OTEL_PREFIX}/lib/pkgconfig/opentelemetry-c-wrapper.pc"


FROM alpine:${ALPINE_VERSION} AS build
ARG HAPROXY_VERSION
ARG TARGET_PREFIX
ARG OTEL_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        brotli-static \
        build-base \
        c-ares-dev \
        curl-dev \
        curl-static \
        libidn2-static \
        libpsl-static \
        libunistring-static \
        linux-headers \
        lua5.4-dev \
        nghttp2-static \
        nghttp3-static \
        openssl-dev \
        openssl-libs-static \
        pcre2-dev \
        pcre2-static \
        pkgconfig \
        zlib-dev \
        zlib-static \
        zstd-static
COPY --from=otel-wrapper-build ${OTEL_PREFIX} ${OTEL_PREFIX}
ADD ".tmp/haproxy-${HAPROXY_VERSION}.tar.gz" /build/
WORKDIR "/build/haproxy-${HAPROXY_VERSION}"
RUN PKG_CONFIG_PATH="${OTEL_PREFIX}/lib/pkgconfig" \
    make "-j$(nproc)" \
        TARGET=linux-musl \
        USE_EPOLL=1 \
        USE_NETFILTER=1 \
        USE_PCRE2=1 \
        USE_PCRE2_JIT=1 \
        USE_POLL=1 \
        USE_THREAD=0 \
        USE_TPROXY=1 \
        USE_LINUX_TPROXY=1 \
        USE_LINUX_SPLICE=1 \
        USE_LINUX_CAP=1 \
        USE_LIBCRYPT=1 \
        USE_CRYPT_H=1 \
        USE_GETADDRINFO=1 \
        USE_OPENSSL=1 \
        USE_QUIC=1 \
        USE_QUIC_OPENSSL_COMPAT=1 \
        USE_ENGINE=1 \
        USE_LUA=1 \
        USE_ACCEPT4=1 \
        USE_PRCTL=1 \
        USE_CPU_AFFINITY=1 \
        USE_TFO=1 \
        USE_NS=1 \
        USE_DL=1 \
        USE_MATH=1 \
        USE_RT=1 \
        USE_BACKTRACE=1 \
        USE_PROMEX=1 \
        USE_OTEL=1 \
        USE_LIBATOMIC=1 \
        USE_PTHREAD_EMULATION=1 \
        OPT_CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
        LDFLAGS="-static -static-pie -static-libgcc -static-libstdc++ -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--gc-sections" \
        SSL_LIB=/usr/lib \
        SSL_INC=/usr/include/openssl \
        LUA_LIB=/usr/lib/lua5.4 \
        LUA_INC=/usr/include/lua5.4
RUN make install \
        PREFIX=${TARGET_PREFIX}


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
    echo "=== haproxy -vv ==="; \
    "${TARGET_PREFIX}/sbin/haproxy" -vv | tee /tmp/haproxy-vv.txt; \
    grep -Eqi 'opentelemetry' /tmp/haproxy-vv.txt || (echo "Error: haproxy was not built with OpenTelemetry support." && exit 1); \
    strings "${TARGET_PREFIX}/sbin/haproxy" | grep -Eq 'OtlpGrpc(Exporter|MetricExporter|LogRecordExporter|Client)' || (echo "Error: OTLP/gRPC exporter symbols were not linked into haproxy." && exit 1); \
    strace -e trace=execve "${TARGET_PREFIX}/sbin/haproxy" -vv >/dev/null


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/haproxy"]
