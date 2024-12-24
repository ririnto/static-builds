ARG ALPINE_VERSION=3.18.9
ARG HAPROXY_VERSION=3.0.7
ARG UBI9_MICRO_VERSION=9.5-1734513256

FROM alpine:${ALPINE_VERSION} AS haproxy-builder
ARG HAPROXY_VERSION
RUN apk del --purge openssl
RUN apk add --no-cache \
        build-base \
        linux-headers \
        zlib-dev \
        zlib-static \
        openssl1.1-compat-dev \
        openssl1.1-compat-libs-static \
        lua5.4-dev \
        pcre2-dev
RUN --mount=type=bind,target=/tmp/mount \
    if [[ -f /tmp/mount/haproxy-${HAPROXY_VERSION}.tar.gz ]]; then cp /tmp/mount/haproxy-${HAPROXY_VERSION}.tar.gz /tmp; \
    else wget https://www.haproxy.org/download/3.0/src/haproxy-${HAPROXY_VERSION}.tar.gz; \
    fi && \
    tar -xzf haproxy-${HAPROXY_VERSION}.tar.gz && \
    cd haproxy-${HAPROXY_VERSION} && \
    make -j$(nproc) \
        TARGET=linux-musl \
        LDFLAGS="-static -static-libgcc -static-libstdc++" \
        # enable epoll() on Linux 2.6. Automatic.
        USE_EPOLL=1 \
        # enable kqueue() on BSD. Automatic.
        USE_KQUEUE=0 \
        # enable event ports on SunOS systems. Automatic.
        USE_EVPORTS=0 \
        # enable netfilter on Linux. Automatic.
        USE_NETFILTER=1 \
        # enable use of libpcre for regex.
        USE_PCRE=0 \
        # enable JIT for faster regex on libpcre >= 8.32
        USE_PCRE_JIT=0 \
        # enable use of libpcre2 for regex. Recommended.
        USE_PCRE2=1 \
        # enable JIT for faster regex on libpcre2
        USE_PCRE2_JIT=1 \
        # enable poll(). Automatic.
        USE_POLL=1 \
        # enable threads support.
        USE_THREAD=1 \
        # enable static libpcre.
        USE_STATIC_PCRE=0 \
        # enable static libpcre2. Recommended.
        USE_STATIC_PCRE2=0 \
        # enable transparent proxy. Automatic.
        USE_TPROXY=1 \
        # enable full transparent proxy. Automatic.
        USE_LINUX_TPROXY=1 \
        # enable kernel 2.6 splicing. Automatic.
        USE_LINUX_SPLICE=1 \
        # enable Linux capabilities.
        USE_LINUX_CAP=1 \
        # enable encrypted passwords using -lcrypt
        USE_LIBCRYPT=1 \
        # set it if your system requires including crypt.h
        USE_CRYPT_H=1 \
        # use getaddrinfo() to resolve IPv6 host names.
        USE_GETADDRINFO=1 \
        # enable use of OpenSSL. Recommended, but see below.
        USE_OPENSSL=1 \
        # enable use of AWS-LC
        USE_OPENSSL_AWSLC=0 \
        # enable use of wolfSSL with the OpenSSL API
        USE_OPENSSL_WOLFSSL=0 \
        # enable use of QUIC with the quictls API (quictls, libressl, boringssl)
        USE_QUIC=1 \
        # enable use of QUIC with the standard openssl API (limited features)
        USE_QUIC_OPENSSL_COMPAT=1 \
        # enable use of OpenSSL Engine.
        USE_ENGINE=1 \
        # enable Lua support.
        USE_LUA=1 \
        # enable use of accept4() on linux. Automatic.
        USE_ACCEPT4=1 \
        # enable use of closefrom() on *bsd, solaris. Automatic.
        USE_CLOSEFROM=0 \
        # enable use of prctl(). Automatic.
        USE_PRCTL=1 \
        # enable use of procctl(). Automatic.
        USE_PROCCTL=0 \
        # enable zlib library support and disable SLZ
        USE_ZLIB=1 \
        # enable slz library instead of zlib (default=enabled)
        USE_SLZ=0 \
        # enable pinning processes to CPU on Linux. Automatic.
        USE_CPU_AFFINITY=1 \
        # enable TCP fast open. Supported on Linux >= 3.7.
        USE_TFO=1 \
        # enable network namespace support. Supported on Linux >= 2.6.24.
        USE_NS=1 \
        # enable it if your system requires -ldl. Automatic on Linux.
        USE_DL=1 \
        # enable use of -lm. Automatic.
        USE_MATH=1 \
        # enable it if your system requires -lrt. Automatic on Linux.
        USE_RT=1 \
        # enable backtrace(). Automatic on Linux.
        USE_BACKTRACE=1 \
        # enable the Prometheus exporter
        USE_PROMEX=1 \
        # enable DeviceAtlas api.
        USE_DEVICEATLAS=0 \
        # enable third party device detection library from 51Degrees
        USE_51DEGREES=0 \
        # enable WURFL detection library from Scientiamobile
        USE_WURFL=0 \
        # enable sd_notify() support.
        USE_SYSTEMD=1 \
        # use when the linker fails to emit __start_init/__stop_init
        USE_OBSOLETE_LINKER=1 \
        # use the more advanced thread state dump system. Automatic.
        USE_THREAD_DUMP=1 \
        # enable the OpenTracing filter
        USE_OT=0 \
        # enable the memory profiler. Linux-glibc only.
        USE_MEMORY_PROFILING=0 \
        # force to link with/without libatomic. Automatic.
        USE_LIBATOMIC=1 \
        # replace pthread's rwlocks with ours
        USE_PTHREAD_EMULATION=1 \
        # use shm_open() for the startup-logs
        USE_SHM_OPEN=1 \
        # force the path to libpcre.
        PCRE2DIR=/usr \
        # force the lib path to libssl/libcrypto
        SSL_LIB=/usr/lib \
        # force the include path to libssl/libcrypto
        SSL_INC=/usr/include/openssl \
        # force the lib path to lua
        LUA_LIB=/usr/lib/lua5.4 \
        # force the include path to lua
        LUA_INC=/usr/include/lua5.4 \
        # Use ZLIB_INC and ZLIB_LIB to force path to zlib.h and libz.{a,so} if needed.
        ZLIB_LIB=/lib \
        ZLIB_INC=/usr/include \
    && \
    make install PREFIX=/usr/local && \
    cd /tmp && \
    rm -rf haproxy-${HAPROXY_VERSION}.tar.gz haproxy-${HAPROXY_VERSION}
RUN ldd_output=$(ldd /usr/local/sbin/haproxy 2>&1 || true) && echo "ldd output: $ldd_output" && \
    ( \
        echo "$ldd_output" | grep -q "Not a valid dynamic program" && \
        echo "Static build verified: No shared libraries found" || \
        (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1) \
    ) && \
    file_output=$(file /usr/local/sbin/haproxy 2>&1 || true) && echo "file output: $file_output" && \
    ( \
        echo "$file_output" | grep -q "statically linked" && \
        echo "Static build verified" || \
        (echo "Error: Dynamic link found. This should be a static build!" && exit 1) \
    )

FROM redhat/ubi9-micro:${UBI9_MICRO_VERSION} AS haproxy
COPY --from=haproxy-builder /usr/local/sbin/haproxy /usr/local/sbin/haproxy
RUN chmod +x /usr/local/sbin/haproxy
ENTRYPOINT ["/usr/local/sbin/haproxy"]
