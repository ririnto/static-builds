# nginx

This target builds static nginx with njs and VTS compiled into one musl PIE binary.
Both nginx targets independently build the same njs scripting profile.
The selected nginx 1.30.6 release belongs to the Open Source Stable series.
NGINX Open Source does not publish a separate LTS line.

## Modules and Features

- Protocol/security modules: `--with-http_ssl_module`, `--with-http_v2_module`, `--with-http_v3_module`, and `--with-stream_ssl_module`.
- Utility modules: `--with-http_stub_status_module`, `--with-http_gzip_static_module`, `--with-stream_realip_module`, and `--with-stream_ssl_preread_module`.
- Third-party modules include VTS and the njs HTTP and stream modules.
- Explicit removals: selected HTTP modules are disabled with `--without-*` flags, including `fastcgi`, `uwsgi`, `scgi`, and `memcached`.

## Allowed Target-Specific Variations

- This target packages only `sbin/nginx`.
- It includes the QuickJS-NG engine with `js_engine qjs` and excludes optional libxslt support.
- It excludes active upstream healthchecks.

## Runtime configuration

Both nginx images install their operational base configuration at `/config/nginx.conf` and include HTTP snippets from `/config/conf.d/*.conf`.
The compiled path is `config/nginx.conf` beneath the selected prefix, and the images use `-p /`.
The base configuration retains the upstream default HTTP server and supports additional HTTP configuration through `config/conf.d`.
Mount your own configuration or snippets at those paths.
No host configuration is changed by the build.
The images include minimal `nobody` account records for worker processes and run nginx in the foreground.
Release archives contain only the binary, so supply your own `config/nginx.conf`, writable logs, and runtime directories when deploying an archive.

## Scripting Installation

The njs HTTP and stream modules use nginx's standard static `--add-module` installation.
[scripting.conf](examples/njs/scripting.conf) and [scripting.js](examples/njs/scripting.js) provide optional loopback HTTP and TCP examples.
They are repository examples, not runtime dependencies, and are excluded from images, build outputs, and release archives.
Set `js_engine qjs` to select the statically linked QuickJS-NG engine.
The [official engine guidance](https://nginx.org/en/docs/njs/engine.html) recommends QuickJS for new configurations.
The optional libxslt dependency is excluded from this target.

The njs 1.0.2 release fixes XML security issues CVE-2026-101025 and CVE-2026-101019 and hardens Fetch response handling.
It also changes XML node-mutation compatibility and Fetch response validation.
Review the [official njs 1.0.2 release notes](https://nginx.org/en/docs/njs/changes.html#njs1.0.2) before upgrading existing scripts that use XML or Fetch.

On a compatible Linux host, run the repository example explicitly with the built binary from the repository root:

```sh
mkdir -p dist/nginx/logs
./dist/nginx/sbin/nginx -p "$PWD/dist/nginx/" -c "$PWD/nginx/examples/njs/scripting.conf"
curl http://127.0.0.1:18081/njs
./dist/nginx/sbin/nginx -p "$PWD/dist/nginx/" -c "$PWD/nginx/examples/njs/scripting.conf" -s quit
```

The Dockerfile performs these checks before exporting the built installation:

- Static linking assertions
- Executable permission checks

Upstream njs runtime behavior is not re-tested by this target.
The custom [healthcheck target](../nginx-upstream-healthcheck/README.md) validates its healthcheck configuration.
Run `go tool task shfmt -- -d -i 2 -ci` from the repository root to check maintained shell formatting.

## How to Verify

> [!NOTE]
>
> Outputs are under `dist/nginx/`.
> The build caller selects this directory with an explicit `--output` argument.

```bash
./dist/nginx/sbin/nginx -V
```

## Runtime Introspection Output

### Historical capture provenance

The output below is a historical snapshot, not verification of the version currently selected in [metadata.json](../metadata.json).
The recorded output identifies nginx 1.30.0 with nginx-module-vts 0.2.5 and the displayed static PIE configure flags.
The [Dockerfile](Dockerfile) builds from source and checks the built artifact in its Dockerfile verify stage before export.

The historical capture date, repository revision, release tag, exact invocation, and host or container environment are unknown.
The source-backed commands below capture the same introspection surfaces, but do not establish how the historical output was collected.

### Regenerate the snapshot

From the repository root, use the [GoReleaser configuration](../.goreleaser.yaml) to build and package the target selected by the current metadata, then run the capture commands on a compatible Linux host.
The default build platform is `linux/amd64`.
The build caller invokes Task to download the inputs and run rootless BuildKit, including the Dockerfile verify stage.
These commands assume the GoReleaser artifact directory.
Adjust the paths when invoking the low-level Task build with a different `--output` destination.

```bash
TARGET=nginx TAG=nginx-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate
./dist/nginx/sbin/nginx -V 2>&1
```

Record the capture date, repository revision, selected metadata versions, build platform, build environment, exact commands, and results with any replacement output.
Preserve errors and distinguish build results from the host captures above.
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
