# Apache HTTPd Static Build

This target builds Apache HTTPd as static PIE with selected modules
compiled in.

## Modules and Features

### Build Options (Explicit)

- Core protocol/security: `--enable-http`, `--enable-ssl`,
  `--enable-http2`, `--enable-proxy-http2`, `--enable-remoteip`, and
  `--enable-rewrite`.
- Compression: `--enable-deflate` and `--enable-brotli`.
- Proxy/load-balancer stack: `--enable-proxy`,
  `--enable-proxy-http`, `--enable-proxy-balancer`,
  `--enable-proxy-hcheck`, `--enable-lbmethod-byrequests`,
  `--enable-lbmethod-bytraffic`, and `--enable-lbmethod-bybusyness`.
- Static packaging policy: `--enable-so=no`,
  `--enable-mods-static=few`, and `--disable-autoindex`.
- Utility inclusion: `rotatelogs` is enabled with
  `--enable-static-rotatelogs`.

### Runtime/Packaging Snapshot

- Compiled settings and module inventory are captured in
  [Runtime Introspection Output](#runtime-introspection-output) using
  `httpd -V`, `httpd -l`, and `httpd -M`.

## Allowed Target-Specific Variations

- This target MAY release both `bin/httpd` and
  `bin/rotatelogs` as one approved packaging profile.
- The target directory name remains `apache-httpd`, but release tags
  intentionally use the `httpd-` prefix.
- Verify output from `httpd -M` MAY contain only the expected FQDN
  warning in some environments; treat `httpd -l` as the authoritative
  compiled-module inventory for this target.

## How to Verify

> [!NOTE]
> Outputs are under `.out/apache-httpd/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/apache-httpd/bin/httpd -V
./.out/apache-httpd/bin/httpd -l
```

## Runtime Introspection Output

### httpd -V

```text
Server version: Apache/2.4.66 (Unix)
Server built:   Mar  4 2026 12:32:40
Server's Module Magic Number: 20120211:141
Server loaded:  APR 1.7.6, APR-UTIL 1.6.3, PCRE 10.47 2025-10-21
Compiled using: APR 1.7.6, APR-UTIL 1.6.3, PCRE 10.47 2025-10-21
Architecture:   64-bit
Server MPM:     event
  threaded:     yes (fixed thread count)
    forked:     yes (variable process count)
Server compiled with....
 -D APR_HAS_SENDFILE
 -D APR_HAS_MMAP
 -D APR_HAVE_IPV6 (IPv4-mapped addresses enabled)
 -D APR_USE_PROC_PTHREAD_SERIALIZE
 -D APR_USE_PTHREAD_SERIALIZE
 -D SINGLE_LISTEN_UNSERIALIZED_ACCEPT
 -D APR_HAS_OTHER_CHILD
 -D AP_HAVE_RELIABLE_PIPED_LOGS
 -D DYNAMIC_MODULE_LIMIT=256
 -D HTTPD_ROOT="/home/nobody"
 -D SUEXEC_BIN="/home/nobody/bin/suexec"
 -D DEFAULT_PIDLOG="logs/httpd.pid"
 -D DEFAULT_SCOREBOARD="logs/apache_runtime_status"
 -D DEFAULT_ERRORLOG="logs/error_log"
 -D AP_TYPES_CONFIG_FILE="conf/mime.types"
 -D SERVER_CONFIG_FILE="conf/httpd.conf"
```

### httpd -l

```text
Compiled in modules:
  core.c
  mod_authn_file.c
  mod_authn_core.c
  mod_authz_host.c
  mod_authz_groupfile.c
  mod_authz_user.c
  mod_authz_core.c
  mod_access_compat.c
  mod_auth_basic.c
  mod_socache_shmcb.c
  mod_watchdog.c
  mod_reqtimeout.c
  mod_filter.c
  mod_deflate.c
  mod_brotli.c
  http_core.c
  mod_mime.c
  mod_log_config.c
  mod_env.c
  mod_headers.c
  mod_setenvif.c
  mod_version.c
  mod_remoteip.c
  mod_proxy.c
  mod_proxy_http.c
  mod_proxy_wstunnel.c
  mod_proxy_balancer.c
  mod_proxy_hcheck.c
  mod_slotmem_shm.c
  mod_ssl.c
  mod_http2.c
  mod_proxy_http2.c
  mod_lbmethod_byrequests.c
  mod_lbmethod_bytraffic.c
  mod_lbmethod_bybusyness.c
  event.c
  mod_unixd.c
  mod_status.c
  mod_asis.c
  mod_dir.c
  mod_alias.c
  mod_rewrite.c
```

### httpd -M

```text
AH00558: httpd: Could not reliably determine the server's fully qualified
  domain name, using 172.18.0.3. Set the 'ServerName' directive globally to
  suppress this message
```

> [!NOTE]
> The `httpd -M` output in this environment is only an FQDN warning and
> does not list modules. Treat `httpd -l` as the authoritative compiled-in
> module list for this static build.
