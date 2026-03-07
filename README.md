# static-builds

Build statically-linked binaries using Docker multi-stage builds for
portable, minimal container deployments.

## Features

- Static Linking - Produce fully statically-linked binaries using
  musl libc
- Multi-stage Builds - Leverage Docker BuildKit for efficient,
  cacheable builds
- Minimal Images - Target Red Hat UBI9 Micro or similar minimal
  runtime images
- Extensible - Add new build targets by following a simple directory
  structure
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

Build artifacts are written to `out/<target>/` by default for local
builds. In CI (`CI=true` or `GITHUB_ACTIONS=true`), artifacts remain
under `<target>/` for release packaging compatibility. You can
override both behaviors with `BUILD_OUTPUT_DEST`.

## Project Structure

```text
.
├── build.sh              # Main build entry point
├── download.sh           # Download dispatcher (routes to module download.sh)
├── .github/
│   └── scripts/
│       └── common.sh     # Shared common functions
├── out/                  # Local build outputs (gitignored)
│   └── <target>/         # Local artifacts (for example `sbin/`, `bin/`)
└── <target>/             # Build target directory
    ├── .env              # Version configuration
    ├── Dockerfile        # Multi-stage build definition
    ├── download.sh       # Source download script (optional)
    └── ...               # CI/release artifacts under `<target>/`
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
> Check existing targets (`nginx/`, `haproxy/`, `apache-httpd/`)
> for reference implementations.

## Developer Guide

### Target Structure

Each target must follow this structure:

```text
<target>/
├── .env              # Version configuration (required)
├── Dockerfile        # Multi-stage build definition (required)
├── download.sh       # Source download script (optional)
├── README.md        # Target-specific documentation (optional)
└── AGENTS.md        # Target-specific conventions (optional)
```

### Adding a New Target

1. Create target directory: Create a new directory named after your target (e.g., `your-target/`)

2. Add .env file: Create `.env` with version variables:

   ```text
   ALPINE_VERSION=3.23
   YOUR_TARGET_VERSION=1.0.0
   UBI9_MICRO_VERSION=9.5
   ```

3. Create Dockerfile: Implement a multi-stage build:

   ```dockerfile
   # Build stage
   FROM alpine:${ALPINE_VERSION} AS build
   ARG YOUR_TARGET_VERSION
   ADD "src/your-target-${YOUR_TARGET_VERSION}.tar.gz" /build/
   # ... build steps ...

   # Verify stage (optional but recommended)
   FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify
   ARG YOUR_TARGET_VERSION
   COPY --from=build /your-target /target/your-target
   # ... verification steps (ELF check, static linking, strace) ...

   # Final stage
   FROM scratch
   COPY --from=verify /target /target
   ENTRYPOINT ["/target/your-target"]
   ```

4. Optional download.sh: Create `download.sh` to fetch source files:

   ```bash
   #!/usr/bin/env sh
   set -eu
   # Source common functions
   . "$(dirname -- "$0")/../.github/scripts/common.sh"

   download_tarball "https://example.com/your-target-${YOUR_TARGET_VERSION}.tar.gz" \
                 "src/your-target-${YOUR_TARGET_VERSION}.tar.gz"
   ```

5. Update Makefile: Add target to `TARGETS` variable:

   ```makefile
   TARGETS := nginx haproxy apache-httpd coredns dnsmasq vector monit your-target
   ```

6. Update release workflow: Add release configuration in `.github/workflows/release-from-tag.yaml`:

   ```yaml
   - startsWith(github.ref_name, 'your-target-') && 'your-target'
   - startsWith(github.ref_name, 'your-target-') && 'your-target/bin/your-target'
   ```

7. Validate: Run `make build your-target` to verify the build works

### Best Practices

- Security hardening: Use static PIE builds (`-fPIE -pie`)
- Verification: Always include a verify stage with ELF checks, static linking verification, and strace validation
- Caching: Use `--mount=type=cache` for Alpine/DNF caches
- Documentation: Document target-specific decisions in `AGENTS.md` if needed
- Version variables: Follow naming convention `{TARGET}_VERSION` for consistency

## Release Process

### Creating a Release

Releases are triggered by Git tags following the pattern `<target>-<version>.<revision>`:

- Format: `{target}-{official_version}.{revision}`
- Example: `nginx-1.28.2.18` (target: nginx, version: 1.28.2, revision: 18)
- Validation: Release tags are validated against `.env` file versions

### Steps to Release

1. Update versions: Update version variables in target `.env` file:

   ```text
   YOUR_TARGET_VERSION=2.0.0
   ```

2. Test build: Verify that build works locally:

   ```bash
   make build your-target
   ```

3. Commit changes: Commit version updates:

   ```bash
   git add your-target/.env
   git commit -m "Update your-target to 2.0.0"
   ```

4. Create tag: Create and push release tag:

   ```bash
   git tag your-target-2.0.0.0
   git push origin your-target-2.0.0
   ```

5. CI automation: GitHub Actions automatically:
   - Validates tag format and version
   - Builds target
   - Scans for vulnerabilities (Trivy)
   - Uploads artifacts to GitHub Actions
   - Creates GitHub Release with `.tar.gz` package

### Release Tag Format

Tags MUST follow this format:

- `{target}-{version}.{revision}`
- target: Target name (e.g., nginx, haproxy, apache-httpd)
- version: Official version from `.env` file (e.g., 1.28.2)
- revision: Release revision suffix starting at 0, incrementing for rebuilds (e.g., 18)

Valid examples:
- `nginx-1.28.2.18` (nginx version 1.28.2, revision 18)
- `httpd-2.4.66.5` (apache-httpd version 2.4.66, revision 5)
- `haproxy-3.2.13.0` (haproxy version 3.2.13, revision 0)

Invalid examples:
- `nginx-1.28.2` (missing revision suffix)
- `custom-1.0.0.0` (unknown target)
- `nginx-1.28.2.x` (non-numeric revision)

## How It Works

1. The `Makefile` invokes `build.sh`, which validates target
   directory and required files
2. Docker BuildKit executes the multi-stage Dockerfile via
   `docker buildx build`
3. Built artifacts go to `out/<target>/` for local builds, and to
   `<target>/` in CI for release packaging compatibility
Build caching is automatically handled via per-target
`<target>/.cache/` directories.

## CI Behavior

- Archive upload includes selected release files as a workflow
  artifact.
- Release upload is optional and packages selected release files
  into one `.tar.gz` per tag.
- Tag push release uses unified workflow
  `.github/workflows/release-from-tag.yaml`.
- Workflow automatically determines target from tag pattern and calls
  reusable template `.github/workflows/template-release.yaml`.
- Template builds mapped target, scans for vulnerabilities,
  uploads selected files as artifact, then uploads `${tag}.tar.gz`
  that contains selected release files.

Selected release binaries:

- `nginx`: `<target>/sbin/nginx`
- `haproxy`: `<target>/sbin/haproxy`
- `apache-httpd`: `<target>/bin/httpd`
- `coredns`: `<target>/coredns`
- `dnsmasq`: `<target>/sbin/dnsmasq`
- `vector`: `<target>/bin/vector`
- `monit`: `<target>/bin/monit`

## Logging Strategy

> [!NOTE]
> `apache-httpd` releases include both `bin/httpd` and
> `bin/rotatelogs` for piped logging support.

### Piped logging with rotatelogs

The `rotatelogs` utility is included in apache-httpd releases and
can be used for log rotation:

1. External rotatelogs: Use a system-installed `rotatelogs` or
   provide it separately
2. Alternative rotation tools: Use `logrotate`, `multilog`,
   or other log rotation solutions
3. Application-level logging: Configure applications to write
   directly to files managed by external rotation

### Container-native logging alternatives

For containerized deployments, leverage Docker's native logging
drivers:

```bash
# Example: Use Docker's built-in log rotation
docker run --log-driver json-file --log-opt max-size=10m --log-opt
  max-file=3 <image>

# Or use external logging drivers
docker run --log-driver fluentd --log-opt
  fluentd-address=fluentd:24224 <image>
```

Docker automatically handles log rotation and can forward logs to
external systems like ELK, Splunk, or cloud logging services.

### For full details

See [apache-httpd/AGENTS.md](apache-httpd/AGENTS.md) for
complete decision rationale and implementation guidance.

### Running haproxy binary image

```bash
docker run --rm \
  -v "$(pwd)/haproxy.cfg:/etc/haproxy/haproxy.cfg:ro" \
  <image> -c -f /etc/haproxy/haproxy.cfg
```
