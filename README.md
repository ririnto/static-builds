# static-builds

Build statically-linked binaries using Docker multi-stage builds for portable, minimal container deployments.

## Features

- Static Linking - Produce fully statically-linked binaries using musl libc
- Multi-stage Builds - Leverage Docker BuildKit for efficient, cacheable builds
- Minimal Images - Target Red Hat UBI9 Micro or similar minimal runtime images
- Extensible - Add new build targets by following a simple directory structure
- Reproducible - Version-controlled configurations via `.env` files

## Prerequisites

- Docker

## Usage

```bash
make build <target>
```

### Makefile Commands

```bash
make help
make list-targets
make build nginx
```

Build artifacts are written directly under `<target>/`.

## Project Structure

```text
.
├── build.sh              # Main build entry point
├── download.sh           # Download dispatcher (routes to module download.sh)
├── .github/
│   └── scripts/
│       └── common.sh     # Shared common functions
└── <target>/             # Build target directory
    ├── .env              # Version configuration
    ├── Dockerfile        # Multi-stage build definition
    ├── download.sh       # Source download script (optional)
    └── ...               # Built artifacts (for example `sbin/`, `bin/`)
```

## Adding a New Target

1. Create a new directory with your target name
2. Add a `.env` file with version variables:

   ```text
   ALPINE_VERSION=3.23
   YOUR_SOFTWARE_VERSION=1.0.0
   ```

3. Add a `Dockerfile` with your multi-stage build configuration
4. Optionally add `download.sh` for source downloads

> [!TIP]
> Check existing targets (`nginx/`, `haproxy/`, `apache-httpd/`) for reference implementations.

## How It Works

1. The `Makefile` invokes `build.sh`, which validates the target directory and required files
2. A BuildKit container is launched via `docker run`
3. BuildKit executes the multi-stage Dockerfile
4. Built artifacts are output directly into the target directory

Build caching is automatically handled via the `.cache/` directory.

## CI Behavior

- Archive upload includes selected release files as a workflow artifact.
- Release upload is optional and packages selected release files into one `.tar.gz` per tag.
- To enable release upload in GitHub Actions, set `release=true` and use one of:
  - `release_tag`: explicit full tag (for example, `httpd-2.4.66.1`)
  - `release_suffix`: custom last segment; CI composes `<name>-<version>.<suffix>`
- Tag push release uses per-target caller workflows:
  - `.github/workflows/release-nginx-from-tag.yaml`
  - `.github/workflows/release-haproxy-from-tag.yaml`
  - `.github/workflows/release-httpd-from-tag.yaml`
  - `.github/workflows/release-coredns-from-tag.yaml`
  - `.github/workflows/release-dnsmasq-from-tag.yaml`
  - `.github/workflows/release-vector-from-tag.yaml`
- Each caller is bound to one tag pattern (`nginx-*`, `haproxy-*`, `httpd-*`, `coredns-*`, `dnsmasq-*`, `vector-*`) and calls reusable template `.github/workflows/template-release.yaml`.
- Template builds mapped target, uploads selected files as artifact, then uploads `${tag}.tar.gz` that contains selected release files.

Selected release binaries:

- `nginx`: `<target>/sbin/nginx`
- `haproxy`: `<target>/sbin/haproxy`
- `apache-httpd`: `<target>/bin/httpd`
- `coredns`: `<target>/coredns`
- `dnsmasq`: `<target>/sbin/dnsmasq`
- `vector`: `<target>/bin/vector`

## Logging Strategy

**Note:** `apache-httpd` releases include both `bin/httpd` and `bin/rotatelogs` for piped logging support.

### Piped logging with rotatelogs

The `rotatelogs` utility is included in apache-httpd releases and can be used for log rotation:

1. **External rotatelogs:** Use a system-installed `rotatelogs` or provide it separately
2. **Alternative rotation tools:** Use `logrotate`, `multilog`, or other log rotation solutions
3. **Application-level logging:** Configure applications to write directly to files managed by external rotation

### Container-native logging alternatives

For containerized deployments, leverage Docker's native logging drivers:

```bash
# Example: Use Docker's built-in log rotation
docker run --log-driver json-file --log-opt max-size=10m --log-opt max-file=3 <image>

# Or use external logging drivers
docker run --log-driver fluentd --log-opt fluentd-address=fluentd:24224 <image>
```

Docker automatically handles log rotation and can forward logs to external systems like ELK, Splunk, or cloud logging services.

### For full details

See the ADR document at `.sisyphus/plans/rotatelogs-packaging-decision.md` for the complete decision rationale and implementation guidance.

### Running haproxy binary image

```bash
docker run --rm \
  -v "$(pwd)/haproxy.cfg:/etc/haproxy/haproxy.cfg:ro" \
  <image> -c -f /etc/haproxy/haproxy.cfg
```
