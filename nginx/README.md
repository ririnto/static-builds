# nginx

This target builds static nginx with njs and VTS compiled into one musl PIE binary.
Both nginx targets use the same njs scripting profile.
The selected nginx 1.30.5 release belongs to the Open Source Stable series.
NGINX Open Source does not publish a separate LTS line.

## Modules and Features

- Protocol/security modules: `--with-http_ssl_module`, `--with-http_v2_module`, `--with-http_v3_module`, and `--with-stream_ssl_module`.
- Utility modules: `--with-http_stub_status_module`, `--with-http_gzip_static_module`, `--with-stream_realip_module`, and `--with-stream_ssl_preread_module`.
- Third-party modules include VTS and the njs HTTP and stream modules.
- Explicit removals: selected HTTP modules are disabled with `--without-*` flags, including `fastcgi`, `uwsgi`, `scgi`, and `memcached`.

## Allowed Target-Specific Variations

- This target packages `sbin/nginx` and the njs scripting example under `conf/`.
- It includes the QuickJS-NG engine with `js_engine qjs` and excludes optional libxslt support.
- It excludes active upstream healthchecks.

## Scripting Installation

The njs HTTP and stream modules use nginx's standard static `--add-module` installation.
[scripting.conf](scripting.conf) and [scripting.js](scripting.js) provide loopback HTTP and TCP examples.
Set `js_engine qjs` to select the statically linked QuickJS-NG engine.
The optional libxslt dependency is excluded from this target.
Lua, LuaJIT, OpenResty runtime libraries, and a static FFI compatibility provider are excluded.

On a compatible Linux host, run the packaged example from its deployment prefix:

```sh
mkdir -p .out/nginx/logs
./.out/nginx/sbin/nginx -p "$PWD/.out/nginx/" -c conf/scripting.conf
curl http://127.0.0.1:18081/njs
./.out/nginx/sbin/nginx -p "$PWD/.out/nginx/" -c conf/scripting.conf -s quit
```

The Docker verification stage checks static linking, configuration parsing, and real HTTP/stream njs execution.
It runs nginx with standard CLI clients and requires graceful shutdown.
It does not introduce a separate verification framework.

## How to Verify

> [!NOTE]
>
> Outputs are under `.out/nginx/`.
> Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/nginx/sbin/nginx -V
```

## Runtime Introspection Output

### Historical capture provenance

The output below is a historical snapshot, not verification of the version currently selected in [metadata.json](../metadata.json).
The recorded output identifies nginx 1.30.0 with nginx-module-vts 0.2.5 and the displayed static PIE configure flags.
The [Dockerfile](Dockerfile) builds from source and invokes `nginx -V` in its verification stage.

The historical capture date, repository revision, release tag, exact invocation, and host or container environment are unknown.
The source-backed commands below capture the same introspection surfaces, but do not establish how the historical output was collected.

### Regenerate the snapshot

From the repository root, use the [Makefile](../Makefile) to build the target selected by the current metadata, then run the capture commands on a compatible Linux host.
The default build platform is `linux/amd64`.
The build downloads its inputs and runs Docker Buildx, including the Dockerfile's verification stage.
These commands assume the default output directory and must be adjusted if `BUILD_OUTPUT_DEST` is overridden.

```bash
make build nginx
./.out/nginx/sbin/nginx -V 2>&1
```

Record the capture date, repository revision, selected metadata versions, build platform, build environment, exact commands, and results with any replacement output.
Preserve errors and distinguish build-stage verification from the host captures above.
A current-metadata build produces a new snapshot and does not reconstruct the unknown historical environment.
The historical output below has been preserved without running these commands for this documentation change.

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
