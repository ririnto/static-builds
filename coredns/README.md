# CoreDNS Static Build

This target builds a static CoreDNS binary from source using Go.

## Modules and Features

### Build Options (Explicit)

- Build command: `make coredns CGO_ENABLED=0 BUILDOPTS="-v -trimpath"`.
- `CGO_ENABLED=0` forces a pure-Go build without libc runtime
  dependencies.
- No repository-specific plugin patching is applied during build.

### Runtime/Packaging Snapshot

- Plugin inventory is captured in
  [Runtime Introspection Output](#runtime-introspection-output) using
  `coredns -plugins`.
- Binary version/platform metadata is captured with
  `coredns -version`.

## Allowed Target-Specific Variations

- This target intentionally uses a Go builder image and a pure-Go build
  path instead of the Alpine C toolchain pattern used by many other
  targets.
- The packaged artifact is intentionally emitted as `coredns`
  instead of a `bin/` or `sbin/` path.
- Plugin inventory and version reporting are the approved verification
  surface for this target.

## How to Verify

> [!NOTE]
> Outputs are under `.out/coredns/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/coredns/coredns -plugins
```

## Runtime Introspection Output

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
