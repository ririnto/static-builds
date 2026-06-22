# HAProxy 2.4 Static Build

This target builds a static `haproxy` binary from the 2.4 LTS branch for musl-based deployments.

## Modules and Features

### Build Options (Explicit)

- TLS stack: `USE_OPENSSL=1` with static OpenSSL from Alpine.
- Regex stack: `USE_PCRE2=1` and `USE_PCRE2_JIT=1`.
- Embedded subsystems: `USE_LUA=1` and `USE_PROMEX=1` for Lua scripting and the Prometheus exporter.
- Linux runtime features: `USE_EPOLL=1`, `USE_TPROXY=1`, `USE_LINUX_TPROXY=1`, `USE_LINUX_SPLICE=1`, `USE_TFO=1`, and `USE_CPU_AFFINITY=1`.
- Threading: `THREAD` is inherited from the `linux-musl` target defaults; `USE_THREAD_DUMP=1` keeps the advanced thread dump path explicit.

### Runtime/Packaging Profile

- Default runtime tunables and enabled features are captured in [Runtime Introspection Output](#runtime-introspection-output) from `haproxy -vv`.
- This target links HAProxy as static non-PIE with `-fno-PIE` and `-static -no-pie`; HAProxy 2.4 crashes during Lua initialization under static PIE.
- QUIC is not enabled. HAProxy 2.4 ships an experimental QUIC implementation that requires the quictls OpenSSL fork; this target uses stock OpenSSL for universal compatibility.
- OpenTelemetry is not available on the 2.4 branch. The HAProxy OpenTelemetry addon requires HAProxy 2.6 or later.
- Lua 5.4 links against Alpine's `lua5.4-dev` static archive at `/usr/lib/lua5.4/liblua.a`.
- `LIBATOMIC` is not enabled in this build profile.

## Allowed Target-Specific Variations

- This target builds from the 2.4 LTS branch and intentionally omits features that require HAProxy 2.6+ or non-stock TLS libraries.
- This target uses static non-PIE linkage instead of static PIE to keep Lua initialization stable on HAProxy 2.4. `USE_THREAD=0`, `USE_PTHREAD_EMULATION=1`, and `USE_LIBATOMIC=1` still crash with exit 139 under static PIE on `haproxy -vv`.
- Lua support is part of this target's universal runtime profile.
- The release artifact for this target is limited to the static `sbin/haproxy` binary.

## How to Verify

> [!NOTE]
>
> Outputs are under `.out/haproxy-2.4/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/haproxy-2.4/sbin/haproxy -vv
```

## Runtime Introspection Output

```text
HAProxy version 2.4.35-fe925c077 2026/05/11 - https://haproxy.org/
Status: long-term supported branch - will stop receiving fixes around Q2 2026.
Known bugs: http://www.haproxy.org/bugs/bugs-2.4.35.html
Running on: Linux 6.18.33.1-microsoft-standard-WSL2 #1 SMP PREEMPT_DYNAMIC Fri Jun  5 01:12:21 UTC 2026 x86_64
Build options :
  TARGET  = linux-musl
  CPU     = generic
  CC      = cc
  CFLAGS  = -O2 -O2 -pipe -fno-PIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security -Wall -Wextra -Wdeclaration-after-statement -fwrapv -Wno-address-of-packed-member -Wno-unused-label -Wno-sign-compare -Wno-unused-parameter -Wno-clobbered -Wno-missing-field-initializers -Wno-cast-function-type -Wtype-limits -Wshift-negative-value -Wshift-overflow=2 -Wduplicated-cond -Wnull-dereference -Wno-deprecated-declarations
  OPTIONS = USE_EPOLL=1 USE_NETFILTER=1 USE_PCRE2=1 USE_PCRE2_JIT=1 USE_POLL=1 USE_TPROXY=1 USE_LINUX_TPROXY=1 USE_LINUX_SPLICE=1 USE_LIBCRYPT=1 USE_CRYPT_H=1 USE_GETADDRINFO=1 USE_OPENSSL=1 USE_LUA=1 USE_ACCEPT4=1 USE_CPU_AFFINITY=1 USE_TFO=1 USE_NS=1 USE_DL=1 USE_RT=1 USE_PRCTL=1 USE_THREAD_DUMP=1 USE_PROMEX=1
  DEBUG   = 

Feature list : -51DEGREES +ACCEPT4 -BACKTRACE -CLOSEFROM +CPU_AFFINITY +CRYPT_H -DEVICEATLAS +DL +EPOLL -EVPORTS +FUTEX +GETADDRINFO -KQUEUE +LIBCRYPT +LINUX_SPLICE +LINUX_TPROXY +LUA -MEMORY_PROFILING +NETFILTER +NS -OBSOLETE_LINKER +OPENSSL -OT -PCRE +PCRE2 +PCRE2_JIT -PCRE_JIT +POLL +PRCTL -PRIVATE_CACHE -PROCCTL +PROMEX -PTHREAD_PSHARED -QUIC +RT +SLZ -STATIC_PCRE -STATIC_PCRE2 -SYSTEMD +TFO +THREAD +THREAD_DUMP +TPROXY -WURFL -ZLIB

Default settings :
  bufsize = 16384, maxrewrite = 1024, maxpollevents = 200

Built with multi-threading support (MAX_THREADS=64, default=28).
Built with OpenSSL version : OpenSSL 3.5.7 9 Jun 2026
Running on OpenSSL version : OpenSSL 3.5.7 9 Jun 2026
OpenSSL library supports TLS extensions : yes
OpenSSL library supports SNI : yes
OpenSSL library supports : TLSv1.0 TLSv1.1 TLSv1.2 TLSv1.3
Built with Lua version : Lua 5.4.8
Built with the Prometheus exporter as a service
Built with network namespace support.
Built with libslz for stateless compression.
Compression algorithms supported : identity("identity"), deflate("deflate"), raw-deflate("deflate"), gzip("gzip")
Built with transparent proxy support using: IP_TRANSPARENT IPV6_TRANSPARENT IP_FREEBIND
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
              h2 : mode=HTTP       side=FE|BE     mux=H2       flags=HTX|CLEAN_ABRT|HOL_RISK|NO_UPG
            fcgi : mode=HTTP       side=BE        mux=FCGI     flags=HTX|HOL_RISK|NO_UPG
       <default> : mode=HTTP       side=FE|BE     mux=H1       flags=HTX
              h1 : mode=HTTP       side=FE|BE     mux=H1       flags=HTX|NO_UPG
       <default> : mode=TCP        side=FE|BE     mux=PASS     flags=
            none : mode=TCP        side=FE|BE     mux=PASS     flags=NO_UPG

Available services : prometheus-exporter
Available filters :
    [SPOE] spoe
    [CACHE] cache
    [FCGI] fcgi-app
    [COMP] compression
    [TRACE] trace
```
