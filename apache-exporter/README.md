# Apache Exporter Static Build

This target builds the upstream Apache Exporter source into a statically linked Go binary.

## Build Options (Explicit)

- Build command: `make build apache-exporter`.
- The Dockerfile uses a Go builder image based on Alpine and compiles with `CGO_ENABLED=0`, `GOOS=linux`, and the BuildKit `TARGETARCH`.
- The build writes `${TARGET_PREFIX}/bin/apache-exporter`, with `TARGET_PREFIX` defaulting to `/out`.
- The build injects `APACHE_EXPORTER_VERSION` into the upstream version variable with Go linker flags.

## Allowed Target-Specific Variations

- This target uses a Go/Alpine builder and a pure-Go build path instead of the Alpine C toolchain pattern used by other targets.
- The packaged artifact is emitted as `bin/apache-exporter`.
- The container exposes port `9117` and starts `./bin/apache-exporter`.
- The `--version` output is the approved verification surface for this target.

## How to Verify

> [!NOTE]
>
> Outputs are under `.out/apache-exporter/`. Override with `BUILD_OUTPUT_DEST`.

```bash
make build apache-exporter
./.out/apache-exporter/bin/apache-exporter --version 2>&1 | grep -F '1.1.1'
```
