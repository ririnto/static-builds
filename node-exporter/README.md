# Node Exporter static build

This target builds Prometheus Node Exporter for Linux amd64 and arm64.
The upstream [Makefile](https://github.com/prometheus/node_exporter/blob/master/Makefile) uses Promu for Linux builds.
The Dockerfile uses the same Go package, disables CGO, and sets the upstream version field from `metadata.json`.
See the upstream [Promu configuration](https://github.com/prometheus/node_exporter/blob/master/.promu.yml) for the binary name, linker flags, and platform scope.

The release archive contains `bin/node_exporter`, `LICENSE`, and `NOTICE`.
Release tags use `node-exporter-<version>-<suffix>`.
The binary serves Prometheus metrics at `/metrics` on HTTP port `9100` by default.
See the upstream [collector documentation](https://github.com/prometheus/node_exporter#collectors) for platform-specific metric availability.

Build a nonpublishing snapshot and inspect the executable version.

```sh
TAG=node-exporter-local go tool task snapshot -- node-exporter
./dist/node-exporter/bin/node_exporter --version
```
