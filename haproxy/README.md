# HAProxy Static Build

This target builds a static PIE `haproxy` binary for musl-based
deployments.

## Modules and Features

### Build Options (Explicit)

- TLS/QUIC stack: `USE_OPENSSL=1`, `USE_QUIC=1`, and
  `USE_QUIC_OPENSSL_COMPAT=1`.
- Regex stack: `USE_PCRE2=1` and `USE_PCRE2_JIT=1`.
- Embedded subsystems: `USE_LUA=1` and `USE_PROMEX=1`.
- Linux runtime features: `USE_EPOLL=1`, `USE_TPROXY=1`,
  `USE_LINUX_TPROXY=1`, `USE_LINUX_SPLICE=1`, `USE_TFO=1`, and
  `USE_CPU_AFFINITY=1`.
- Explicit threading policy: `USE_THREAD=0` (single-thread mode).

### Runtime/Packaging Snapshot

- Default runtime tunables and enabled features are captured in
  [Runtime Introspection Output](#runtime-introspection-output) from
  `haproxy -vv`.

## Allowed Target-Specific Variations

- This target intentionally enables `USE_THREAD=0` and ships a
  single-thread HAProxy profile.
- Lua support and the Prometheus exporter are approved parts of this
  target's runtime profile.
- The release artifact for this target is intentionally limited to the
  static `sbin/haproxy` binary.

## How to Verify

> [!NOTE]
> Outputs are under `.out/haproxy/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/haproxy/sbin/haproxy -vv
```

## Runtime Introspection Output

```text
HAProxy version 3.2.13-8bd310ca4 2026/02/19 - https://haproxy.org/
Status: long-term supported branch - will stop receiving fixes around Q2 2030.
Known bugs: http://www.haproxy.org/bugs/bugs-3.2.13.html
Running on: Linux 6.6.87.2-microsoft-standard-WSL2 #1 SMP PREEMPT_DYNAMIC Thu
  Jun  5 18:30:46 UTC 2025 x86_64
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
    USE_LIBATOMIC=1 USE_MATH=1 USE_PRCTL=1 USE_QUIC=1 USE_PROMEX=1
    USE_PCRE2=1 USE_PCRE2_JIT=1 USE_QUIC_OPENSSL_COMPAT=1
  DEBUG   =

Feature list : -51DEGREES +ACCEPT4 +BACKTRACE -CLOSEFROM +CPU_AFFINITY
  +CRYPT_H -DEVICEATLAS +DL +ENGINE +EPOLL -EVPORTS +GETADDRINFO -KQUEUE
  +LIBATOMIC +LIBCRYPT +LINUX_CAP +LINUX_SPLICE +LINUX_TPROXY +LUA +MATH
  -MEMORY_PROFILING +NETFILTER +NS -OBSOLETE_LINKER +OPENSSL
  -OPENSSL_AWSLC -OPENSSL_WOLFSSL -OT -PCRE +PCRE2 +PCRE2_JIT -PCRE_JIT
  +POLL +PRCTL -PROCCTL +PROMEX +PTHREAD_EMULATION +QUIC
  +QUIC_OPENSSL_COMPAT +RT +SLZ +SSL -STATIC_PCRE -STATIC_PCRE2 +TFO
  -THREAD +THREAD_DUMP +TPROXY -WURFL -ZLIB +ACME

Default settings :
  bufsize = 16384, maxrewrite = 1024, maxpollevents = 200

Built with SSL library version : OpenSSL 3.5.5 27 Jan 2026
Running on SSL library version : OpenSSL 3.5.5 27 Jan 2026
SSL library supports TLS extensions : yes
SSL library supports SNI : yes
SSL library supports : TLSv1.0 TLSv1.1 TLSv1.2 TLSv1.3
OpenSSL providers loaded : default
QUIC: connection socket-owner mode support : yes
QUIC: GSO emission support : yes
Built with Lua version : Lua 5.4.8
Built with the Prometheus exporter as a service
Built with network namespace support.
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
       quic : mode=HTTP  side=FE     mux=QUIC  flags=HTX|NO_UPG|FRAMED
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
    [COMP] compression
    [FCGI] fcgi-app
    [SPOE] spoe
    [TRACE] trace
```
