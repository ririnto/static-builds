# CoreDNS Static Build

This target builds a static CoreDNS binary from source using Go.

## Modules and Features

### Build Options (Explicit)

- Build command: `make coredns GOTOOLCHAIN=local CGO_ENABLED=0 BUILDOPTS="-v -trimpath"`.
- `GOTOOLCHAIN=local` uses the compiler installed by the Go builder image selected in `metadata.json`.
  It overrides the upstream Makefile's older compiler selection and prevents an automatic toolchain download.
- `CGO_ENABLED=0` forces a pure-Go build without libc runtime dependencies.
- No repository-specific plugin patching is applied during build.

### Runtime/Packaging Snapshot

- Plugin inventory is captured in [Runtime Introspection Output](#runtime-introspection-output) using `coredns -plugins`.
- Binary version/platform metadata is captured with `coredns -version`.

## Allowed Target-Specific Variations

- This target uses a Go builder image and a pure-Go build path instead of the Alpine C toolchain pattern used by many other targets.
- The packaged artifact is emitted as `coredns` instead of a `bin/` or `sbin/` path.
- The verify stage reports the plugin inventory and binary version.

## How to Verify

> [!NOTE]
>
> Outputs are under `dist/coredns/`.
> Override with `BUILD_OUTPUT_DEST`.

```bash
./dist/coredns/coredns -plugins
```

## Runtime Introspection Output

### Historical capture provenance

The output below is a historical snapshot, not verification of the version currently selected in [metadata.json](../metadata.json).
The recorded version output identifies CoreDNS 1.14.1 on `linux/amd64` with Go 1.26.0.
The [Dockerfile](Dockerfile) builds with `CGO_ENABLED=0` and `BUILDOPTS="-v -trimpath"` and checks the built artifact in its Dockerfile verify stage before export.

The historical capture date, repository revision, release tag, exact invocation, and host or container environment are unknown.
The source-backed commands below capture the same introspection surfaces, but do not establish how the historical output was collected.

### Regenerate the snapshot

From the repository root, use the [GoReleaser configuration](../.goreleaser.yaml) to build and package the target selected by the current metadata, then run the capture commands on a compatible Linux host.
The default build platform is `linux/amd64`.
The GoReleaser before hook invokes Task to download the inputs and run rootless BuildKit, including the Dockerfile verify stage.
These commands assume the default output directory and must be adjusted if `BUILD_OUTPUT_DEST` is overridden.

```bash
TARGET=coredns TAG=coredns-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate
./dist/coredns/coredns -version
./dist/coredns/coredns -plugins
```

Record the capture date, repository revision, selected metadata versions, build platform, build environment, exact commands, and results with any replacement output.
Preserve errors and distinguish build results from the host captures above.
A current-metadata build produces a new snapshot and does not reconstruct the unknown historical environment.
The historical output below has been preserved without running these commands for this documentation change.

### coredns -version

```text
CoreDNS-1.14.1
linux/amd64, go1.26.0, 
```

### coredns -plugins

```text
acl
any
auto
autopath
azure
bind
bufsize
cache
cancel
chaos
clouddns
debug
dns64
dnssec
dnstap
erratic
errors
etcd
file
forward
geoip
grpc
grpc_server
header
health
hosts
https
https3
k8s_external
kubernetes
loadbalance
local
log
loop
metadata
minimal
multisocket
nomad
nsid
pprof
prometheus
quic
ready
reload
rewrite
root
route53
secondary
sign
template
timeouts
tls
trace
transfer
tsig
view
whoami
on
```
