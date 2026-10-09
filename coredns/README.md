# CoreDNS Static Build

This target builds a static CoreDNS binary from source using Go.

## Modules and Features

### Build Options (Explicit)

- Build command: `make coredns CGO_ENABLED=0 BUILDOPTS="-v -trimpath"`.
- `CGO_ENABLED=0` forces a pure-Go build without libc runtime dependencies.
- No repository-specific plugin patching is applied during build.

### Runtime/Packaging Snapshot

- Plugin inventory is captured in [Runtime Introspection Output](#runtime-introspection-output) using `coredns -plugins`.
- Binary version/platform metadata is captured with `coredns -version`.

## Allowed Target-Specific Variations

- This target uses a Go builder image and a pure-Go build path instead of the Alpine C toolchain pattern used by many other targets.
- The packaged artifact is emitted as `coredns` instead of a `bin/` or `sbin/` path.
- Plugin inventory and version reporting are the approved verification surface for this target.

## How to Verify

> [!NOTE]
> Outputs are under `.out/coredns/`.
> Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/coredns/coredns -plugins
```

## Runtime Introspection Output

### Historical capture provenance

The output below is a historical snapshot, not verification of the version currently selected in [metadata.json](../metadata.json).
The recorded version output identifies CoreDNS 1.14.1 on `linux/amd64` with Go 1.26.0.
The [Dockerfile](Dockerfile) builds with `CGO_ENABLED=0` and `BUILDOPTS="-v -trimpath"`, then invokes `-version` and `-plugins` in its verification stage.

The historical capture date, repository revision, release tag, exact invocation, and host or container environment are unknown.
The source-backed commands below capture the same introspection surfaces, but do not establish how the historical output was collected.

### Regenerate the snapshot

From the repository root, use the [Makefile](../Makefile) to build the target selected by the current metadata, then run the capture commands on a compatible Linux host.
The default build platform is `linux/amd64`.
The build downloads its inputs and runs Docker Buildx, including the Dockerfile's verification stage.
These commands assume the default output directory and must be adjusted if `BUILD_OUTPUT_DEST` is overridden.

```bash
make build coredns
./.out/coredns/coredns -version
./.out/coredns/coredns -plugins
```

Record the capture date, repository revision, selected metadata versions, build platform, build environment, exact commands, and results with any replacement output.
Preserve errors and distinguish build-stage verification from the host captures above.
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
