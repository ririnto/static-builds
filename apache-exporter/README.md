# Apache Exporter Static Build

This target builds the upstream Apache Exporter source into a statically linked Go binary.

## Build Options (Explicit)

- Build command: `TARGET=apache-exporter TAG=apache-exporter-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate`.
- The Dockerfile uses a Go builder image based on Alpine and compiles with `CGO_ENABLED=0`, `GOOS=linux`, and the BuildKit `TARGETARCH`.
- The build writes `${TARGET_PREFIX}/bin/apache-exporter`, with `TARGET_PREFIX` defaulting to `/out`.
- The build injects `APACHE_EXPORTER_VERSION` into the upstream version variable with Go linker flags.

## Allowed Target-Specific Variations

- This target uses a Go/Alpine builder and a pure-Go build path instead of the Alpine C toolchain pattern used by other targets.
- The packaged artifact is emitted as `bin/apache-exporter`.
- The container exposes port `9117` and starts `./bin/apache-exporter`.
- The `--version` output can be inspected manually after a build.

## How to Verify

> [!NOTE]
>
> Outputs are under `dist/apache-exporter/`.
> Override with `BUILD_OUTPUT_DEST`.

```bash
TARGET=apache-exporter TAG=apache-exporter-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate
./dist/apache-exporter/bin/apache-exporter --version 2>&1 | grep -F '1.1.3'
```
