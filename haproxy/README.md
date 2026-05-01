# HAProxy Static Build

This target builds a static-PIE `haproxy` binary for musl-based deployments,
tracking the `3.4-dev` development branch and enabling the new native
OpenTelemetry filter (`USE_OTEL=1`) introduced in 3.4-dev9.

## Modules and Features

### Build Options (Explicit)

- TLS/QUIC stack: `USE_OPENSSL=1`, `USE_QUIC=1`, and
  `USE_QUIC_OPENSSL_COMPAT=1`.
- Regex stack: `USE_PCRE2=1` and `USE_PCRE2_JIT=1`.
- Embedded subsystems: `USE_LUA=1`, `USE_PROMEX=1`, and `USE_OTEL=1`.
- Linux runtime features: `USE_EPOLL=1`, `USE_TPROXY=1`,
  `USE_LINUX_TPROXY=1`, `USE_LINUX_SPLICE=1`, `USE_TFO=1`, and
  `USE_CPU_AFFINITY=1`.
- Explicit threading policy: `USE_THREAD=0` (single-thread mode).
- Static link policy: `LDFLAGS` enforces `-static -static-pie
  -static-libgcc -static-libstdc++` so the resulting binary embeds
  the C and C++ runtimes and links all OpenTelemetry/abseil/protobuf
  archives statically.

### OpenTelemetry Profile

- The OTel filter is wired against
  [haproxytech/opentelemetry-c-wrapper](https://github.com/haproxytech/opentelemetry-c-wrapper)
  v1.0.2, which sits on top of `opentelemetry-cpp` 1.26.0.
- `opentelemetry-cpp` is built with the OTLP/gRPC, OTLP/HTTP,
  OTLP/File, and Zipkin exporters enabled (`WITH_OTLP_GRPC=ON`,
  `WITH_OTLP_HTTP=ON`, `WITH_OTLP_FILE=ON`, `WITH_ZIPKIN=ON`). gRPC
  support pulls in `re2` and `gRPC` with provider flags pointing at
  locally built `abseil-cpp`, `protobuf`, and `re2`, plus Alpine-provided
  `c-ares`, `openssl`, and `zlib`, so the entire stack remains
  statically linked.
- Preview surfaces required by the wrapper are enabled:
  `WITH_ASYNC_EXPORT_PREVIEW=ON`, `WITH_THREAD_INSTRUMENTATION_PREVIEW=ON`,
  `WITH_METRICS_EXEMPLAR_PREVIEW=ON`. The C++ stack is linked with ABI
  v2 (`WITH_ABI_VERSION_2=ON`, `WITH_NO_DEPRECATED_CODE=ON`).
- Transitive C++ dependencies (`abseil-cpp`, `protobuf`, `re2`, `gRPC`,
  `rapidyaml`, `ms-gsl`, `opentelemetry-proto`) are compiled from pinned
  source tarballs with `BUILD_SHARED_LIBS=OFF` and
  `CMAKE_POSITION_INDEPENDENT_CODE=ON`. The `make download` phase
  retrieves them ahead of time so the Docker build itself does not
  reach GitHub.
- The gRPC build does not require `git submodule init`: incomplete
  `third_party` submodule directories in the gRPC tag archive are bypassed
  by `*_PROVIDER=package` flags. CMake `FetchContent` is forced into fully
  disconnected mode so missing source inputs fail the build instead of
  attempting an in-build network fetch.

### Runtime/Packaging Snapshot

- Default runtime tunables and enabled features are captured in
  [Runtime Introspection Output](#runtime-introspection-output) from
  `haproxy -vv`.

## Allowed Target-Specific Variations

- This target intentionally enables `USE_THREAD=0` and ships a
  single-thread HAProxy profile.
- Lua, Prometheus exporter, and OpenTelemetry filter are approved
  parts of this target's runtime profile.
- The release artifact for this target is intentionally limited to the
  static `sbin/haproxy` binary.
- Pinned upstream versions: `OTEL_C_WRAPPER_VERSION=1.0.2`,
  `OTEL_CPP_VERSION=1.26.0`, `OTEL_PROTO_VERSION=1.8.0`,
  `MS_GSL_VERSION=4.2.1`, `RAPIDYAML_VERSION=0.10.0`,
  `ABSEIL_CPP_VERSION=20250127.1`, `PROTOBUF_VERSION=29.5`,
  `RE2_VERSION=2025-08-05`, `GRPC_VERSION=1.70.2`. These are
  enumerated in `metadata.json` as the single source of truth.

## How to Verify

> [!NOTE]
> Outputs are under `.out/haproxy/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/haproxy/sbin/haproxy -vv
./.out/haproxy/sbin/haproxy -vv | grep -i opentelemetry
strings ./.out/haproxy/sbin/haproxy | grep -E 'OtlpGrpc(Exporter|MetricExporter|LogRecordExporter|Client)'
```

The OTel build is confirmed when `haproxy -vv` reports a line such as
`Built with OpenTelemetry support (C++ version 1.26.0, C Wrapper
version 1.0.2-...)`. The Docker verify stage also checks that OTLP/gRPC
exporter symbols are linked into the final static binary.

## Runtime Introspection Output

```text
HAProxy version 3.4-dev10-c6d45fec8 2026/04/29 - https://haproxy.org/
Status: development branch - not safe for use in production.
Known bugs: https://github.com/haproxy/haproxy/issues?q=is:issue+is:open
Running on: Linux 6.12.76-linuxkit #1 SMP Fri Apr 17 14:56:37 UTC 2026 x86_64
Build options :
  TARGET  = linux-musl
  CC      = cc
  CFLAGS  = -O2 -pipe -fPIE -fstack-protector-strong
    -fstack-clash-protection -ffunction-sections -fdata-sections
    -fno-delete-null-pointer-checks -fno-strict-overflow
    -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2
    -Werror=format-security -g -fwrapv -fvect-cost-model=very-cheap
  OPTIONS = USE_EPOLL=1 USE_NETFILTER=1 USE_POLL=1 USE_THREAD=0
    USE_PTHREAD_EMULATION=1 USE_BACKTRACE=1 USE_TPROXY=1
    USE_LINUX_TPROXY=1 USE_LINUX_CAP=1 USE_LINUX_SPLICE=1 USE_LIBCRYPT=1
    USE_CRYPT_H=1 USE_ENGINE=1 USE_GETADDRINFO=1 USE_OPENSSL=1 USE_LUA=1
    USE_ACCEPT4=1 USE_CPU_AFFINITY=1 USE_TFO=1 USE_NS=1 USE_DL=1 USE_RT=1
    USE_LIBATOMIC=1 USE_MATH=1 USE_PRCTL=1 USE_OTEL=1 USE_QUIC=1
    USE_PROMEX=1 USE_PCRE2=1 USE_PCRE2_JIT=1 USE_QUIC_OPENSSL_COMPAT=1
  DEBUG   =

Feature list : -51DEGREES +ACCEPT4 +ACME +BACKTRACE -CLOSEFROM
  +CPU_AFFINITY +CRYPT_H -DEVICEATLAS +DL -ECH +ENGINE +EPOLL -EVPORTS
  +GETADDRINFO +HAVE_TCP_MD5SIG -KQUEUE +KTLS +LIBATOMIC +LIBCRYPT
  +LINUX_CAP +LINUX_SPLICE +LINUX_TPROXY +LUA +MATH -MEMORY_PROFILING
  +NETFILTER +NS -OBSOLETE_LINKER +OPENSSL -OPENSSL_AWSLC -OPENSSL_WOLFSSL
  -OT +OTEL -PCRE +PCRE2 +PCRE2_JIT -PCRE_JIT +POLL +PRCTL -PROCCTL
  +PROMEX +PTHREAD_EMULATION +QUIC +QUIC_OPENSSL_COMPAT +RT +SHM_OPEN
  +SLZ +SSL -STATIC_PCRE -STATIC_PCRE2 +TFO -THREAD +THREAD_DUMP +TPROXY
  -WURFL -ZLIB

Default settings :
  bufsize = 16384, maxrewrite = 1024, maxpollevents = 200

Built with SSL library version : OpenSSL 3.5.6 7 Apr 2026
Running on SSL library version : OpenSSL 3.5.6 7 Apr 2026
SSL library supports TLS extensions : yes
SSL library supports SNI : yes
SSL library default verify directory : /etc/ssl/certs
SSL library supports : TLSv1.0 TLSv1.1 TLSv1.2 TLSv1.3
OpenSSL providers loaded : default
QUIC: connection sock-per-conn mode support : yes
QUIC: GSO emission support : yes
Built with Lua version : Lua 5.4.8
Built with the Prometheus exporter as a service
Built with network namespace support.
Built with OpenTelemetry support (C++ version 1.26.0, C Wrapper version 1.0.2-844).
Built without multi-threading support (USE_THREAD not set).
Built with libslz for stateless compression.
Compression algorithms supported : identity("identity"), deflate("deflate"),
  raw-deflate("deflate"), gzip("gzip")
Built with transparent proxy support using: IP_TRANSPARENT IPV6_TRANSPARENT
  IP_FREEBIND
Built with PCRE2 version : 10.47 2025-10-21
PCRE2 library supports JIT : yes
Encrypted password support via crypt(3): yes
Built with gcc compiler version 15.2.0

Available polling systems :
      epoll : pref=300,  test result OK
       poll : pref=200,  test result OK
     select : pref=150,  test result OK
Total: 3 (3 usable), will use epoll.

Available multiplexer protocols :
(protocols marked as <default> cannot be specified using 'proto' keyword)
       qmux : mode=HTTP  side=FE|BE  mux=QMUX  flags=HTX|NO_UPG
       quic : mode=HTTP  side=FE|BE  mux=QUIC  flags=HTX|NO_UPG|FRAMED
         h2 : mode=HTTP  side=FE|BE  mux=H2    flags=HTX|HOL_RISK|NO_UPG
  <default> : mode=HTTP  side=FE|BE  mux=H1    flags=HTX
         h1 : mode=HTTP  side=FE|BE  mux=H1    flags=HTX|NO_UPG
       fcgi : mode=HTTP  side=BE     mux=FCGI  flags=HTX|HOL_RISK|NO_UPG
  <default> : mode=SPOP  side=BE     mux=SPOP  flags=HOL_RISK|NO_UPG
       spop : mode=SPOP  side=BE     mux=SPOP  flags=HOL_RISK|NO_UPG
  <default> : mode=TCP   side=FE|BE  mux=PASS  flags=
       none : mode=TCP   side=FE|BE  mux=PASS  flags=NO_UPG

Available services : prometheus-exporter
Available filters :
    [BWLIM] bwlim-in
    [BWLIM] bwlim-out
    [CACHE] cache
    [COMP] comp-req
    [COMP] comp-res
    [COMP] compression
    [FCGI] fcgi-app
    [OTEL] opentelemetry
    [SPOE] spoe
    [TRACE] trace
```
