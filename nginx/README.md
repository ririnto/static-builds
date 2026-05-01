# nginx

This target builds plain static nginx with the repository's standard non-resty nginx modules.

## Modules and Features

- Protocol/security modules: `--with-http_ssl_module`, `--with-http_v2_module`, `--with-http_v3_module`, and `--with-stream_ssl_module`.
- Utility modules: `--with-http_stub_status_module`, `--with-http_gzip_static_module`, `--with-stream_realip_module`, and `--with-stream_ssl_preread_module`.
- Third-party modules: `--add-module=nginx-module-vts-*`.
- Explicit removals: selected HTTP modules are disabled with `--without-*` flags, including `fastcgi`, `uwsgi`, `scgi`, and `memcached`.

## Allowed Target-Specific Variations

- This target ships only `sbin/nginx`.
- This target intentionally does not package Lua, resty, or upstream healthcheck runtime files.

## How to Verify

> [!NOTE]
>
> Outputs are under `.out/nginx/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/nginx/sbin/nginx -V
```

## Runtime Introspection Output

```text
nginx version: nginx/1.30.0
configure arguments:
  --prefix=/home/nobody --with-threads --with-file-aio
  --with-http_ssl_module --with-http_v2_module --with-http_v3_module
  --with-http_realip_module --with-http_gzip_static_module
  --with-http_stub_status_module --without-http_ssi_module
  --without-http_userid_module --without-http_autoindex_module
  --without-http_split_clients_module --without-http_fastcgi_module
  --without-http_uwsgi_module --without-http_scgi_module
  --without-http_memcached_module --without-http_empty_gif_module
  --without-http_browser_module --with-stream --with-stream_ssl_module
  --with-stream_realip_module --with-stream_ssl_preread_module
  --add-module=nginx-module-vts-0.2.5
  --with-cc-opt='-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection
    -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks
    -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero
    -Wformat -Wformat=2 -Werror=format-security'
  --with-ld-opt='-static -static-pie -static-libgcc -Wl,-E -rdynamic
    -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code
    -Wl,--as-needed' --with-libatomic
```
