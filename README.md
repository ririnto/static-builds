# static-builds

Build statically-linked binaries using Docker multi-stage builds for portable, minimal container deployments.

## Features

- Static Linking - Produce fully statically-linked binaries using musl libc
- Multi-stage Builds - Leverage Docker BuildKit for efficient, cacheable builds
- Minimal Images - Target Red Hat UBI9 Micro or similar minimal runtime images
- Extensible - Add new build targets by following a simple directory structure
- Reproducible - Version-controlled configurations via `.env` files

## Prerequisites

- Docker with Docker Compose
- BuildKit support (automatically uses `moby/buildkit:rootless`)

## Usage

```bash
make build TARGET=<target>
```

### Makefile Commands

```bash
make help
make list-targets
make build TARGET=nginx
make build-plain TARGET=haproxy
make build-no-cache TARGET=apache-httpd
make build-plain-no-cache TARGET=nginx
make build-all
make build-all-no-cache
```

Build artifacts are written to `<target>/output/`.

### Example Targets

```bash
make nginx
make haproxy
make apache-httpd
```

## Project Structure

```text
.
├── build.sh              # Main build entry point
├── .github/
│   ├── docker-compose.yaml
│   └── scripts/
│       ├── build.sh      # BuildKit execution script
│       └── download.sh   # Source download utilities
└── <target>/             # Build target directory
    ├── .env              # Version configuration
    ├── Dockerfile        # Multi-stage build definition
    ├── pre-download.sh   # Source download script (optional)
    └── output/           # Built artifacts output directory
```

## Adding a New Target

1. Create a new directory with your target name
2. Add a `.env` file with version variables:

   ```text
   ALPINE_VERSION=3.23
   YOUR_SOFTWARE_VERSION=1.0.0
   ```

3. Add a `Dockerfile` with your multi-stage build configuration
4. Optionally add `pre-download.sh` for source downloads

> [!TIP]
> Check existing targets (`nginx/`, `haproxy/`, `apache-httpd/`) for reference implementations.

## How It Works

1. The `Makefile` invokes `build.sh`, which validates the target directory and required files
2. Docker Compose launches a BuildKit container
3. BuildKit executes the multi-stage Dockerfile
4. Built artifacts are output to the target `output/` directory

Build caching is automatically handled via the `.cache/` directory.

## CI Behavior

- Archive upload always includes the full `<target>/output/` directory as a workflow artifact.
- Release upload is optional and uploads only one selected binary per target.
- To enable release upload in GitHub Actions, set `release=true` and use one of:
  - `release_tag`: explicit full tag (for example, `httpd-2.4.66.1`)
  - `release_suffix`: custom last segment; CI composes `<name>-<version>.<suffix>`
- Tag push release is supported in `.github/workflows/release-from-tag.yaml`.
  - Push tag patterns: `nginx-*`, `haproxy-*`, `httpd-*`, `coredns-*`, `dnsmasq-*`, `vector-*`
  - CI builds the mapped target, uploads full `output/` as artifact, and uploads one selected binary to the matching release tag.

Selected release binaries:

- `nginx`: `<target>/output/sbin/nginx`
- `haproxy`: `<target>/output/sbin/haproxy`
- `apache-httpd`: `<target>/output/bin/httpd`
- `coredns`: `<target>/output/coredns`
- `dnsmasq`: `<target>/output/sbin/dnsmasq`
- `vector`: `<target>/output/bin/vector`
