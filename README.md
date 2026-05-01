# static-builds

Build statically-linked binaries using Docker multi-stage builds for portable, minimal container deployments.

## Features

- Static Linking - Produce fully statically-linked binaries using musl libc
- Multi-stage Builds - Leverage Docker BuildKit for efficient, cacheable builds
- Minimal Outputs - Use a UBI9 Micro verify stage for runtime checks and package final artifacts from a `scratch` stage
- Extensible - Add new build targets by following a simple directory structure
- Reproducible - Version-controlled configurations via `metadata.json`

## Prerequisites

- Docker

## Usage

```bash
make build <target>
```

Build artifacts are written to `.out/<target>/` by default for both local and CI builds. You can override this with `BUILD_OUTPUT_DEST`.

### Running haproxy binary image

```bash
docker run --rm \
  -v "$(pwd)/haproxy.cfg:/etc/haproxy/haproxy.cfg:ro" \
  <image> -c -f /etc/haproxy/haproxy.cfg
```

## Project Structure

```text
.
├── metadata.json           # Canonical build/release metadata
├── Makefile                # Local development build orchestration
├── .gitlab-ci.yml          # GitLab CI configuration
├── scripts/                # Shared build/release scripts
│   ├── download.sh         # Source download dispatcher
│   ├── metadata.sh         # Metadata query helper
│   └── package-release.sh  # Release package helper
├── .github/
│   ├── scripts/
│   │   ├── build.sh        # Docker Buildx build entry
│   │   └── release-guard.sh # Release tag validator
│   └── workflows/
│       ├── release-from-tag.yaml
│       └── template-release.yaml
├── .gitlab/
│   ├── ci/
│   │   └── package-pipeline.jsonnet # GitLab child pipeline generator
│   └── scripts/
│       └── build-rootless.sh        # Rootless BuildKit build entry
├── templates/              # GitLab CI components
│   └── static-release.yml
├── .tmp/                   # Downloaded source cache (gitignored)
├── .cache/                 # Build cache (gitignored)
├── .out/                   # Build outputs (gitignored)
│   └── <target>/           # Local artifacts (for example `sbin/`, `bin/`)
└── <target>/               # Build target directory
    ├── Dockerfile          # Multi-stage build definition
    └── ...
```

## Developer Guide

### Target Structure

Each target must follow this structure:

```text
<target>/
├── Dockerfile        # Multi-stage build definition (required)
├── README.md         # Target-specific documentation (optional)
└── AGENTS.md         # Target-specific conventions (optional)
```

### Adding a New Target

1. Create target directory: Create a new directory named after your target (e.g., `your-target/`)

2. Add centralized metadata: Register the target in `metadata.json`:

   ```json
   {
     "your-target": {
       "tag_prefix": "your-target",
       "version_env_var": "YOUR_TARGET_VERSION",
       "release_files": [
         "bin/your-target"
       ],
       "env": {
         "ALPINE_VERSION": "3.23",
         "YOUR_TARGET_VERSION": "1.0.0",
         "UBI9_MICRO_VERSION": "9.5"
       }
     }
   }
   ```

3. Create Dockerfile: Implement a multi-stage build:

   ```dockerfile
   # Build stage
   FROM alpine:${ALPINE_VERSION} AS build
   ARG YOUR_TARGET_VERSION
   ADD ".tmp/your-target-${YOUR_TARGET_VERSION}.tar.gz" /build/
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

4. Add download metadata: Define each upstream resource directly in `metadata.json`:

   ```json
   {
     "your-target": {
       "downloads": [
         {
           "url": "https://example.com/your-target-{YOUR_TARGET_VERSION}.tar.gz",
           "name": "your-target-{YOUR_TARGET_VERSION}.tar.gz"
         }
       ]
     }
   }
   ```

5. Verify: Targets are loaded from `metadata.json`.

6. Update release trigger mapping: Add the tag trigger and target selection case in `.github/workflows/release-from-tag.yaml`. Release file selection and official versions now come from `metadata.json`.

   ```yaml
   on:
     push:
       tags:
         - 'your-target-*'

   jobs:
     release:
       with:
         target: >-
           startsWith(github.ref_name, 'your-target-') && 'your-target'
   ```

7. Validate: Run `make build your-target` to verify the download and build flow works

### Allowed Target-Specific Variations

Targets share the same root contract, but some targets intentionally vary in builder image, release contents, or runtime packaging.

- Document approved target-specific variations in that target's `README.md`.
- Keep the root `README.md` focused on shared repository behavior.
- Treat `nginx`, `nginx-resty-upstream-healthcheck`, `apache-httpd`, `coredns`, `vector`, `haproxy`, `dnsmasq`, and `monit` differences as documented target profiles, not as undocumented exceptions.

### Best Practices

- Security hardening: Use static PIE builds (`-fPIE -pie`)
- Verification: Always include a verify stage with ELF checks, static linking verification, and strace validation
- Caching: Use `--mount=type=cache` for Alpine/DNF caches
- Documentation: Document approved target-specific variations in each target `README.md`; use `AGENTS.md` for repository-wide policy
- Version variables: Follow naming convention `{TARGET}_VERSION` for consistency inside `metadata.json`

## Release Process

### Creating a Release

Releases are triggered by Git tags following the pattern `<target>-<version>[-<prerelease>].<revision>`:

- Format: `{target}-{official_version}[-{prerelease}].{revision}`
- Example: `nginx-1.28.2.18` (target: nginx, version: 1.28.2, revision: 18)
- Validation: Release tags are validated against `metadata.json`

Current release tag triggers and target selection still live in `.github/workflows/release-from-tag.yaml` because GitHub event filters must stay static, but release-file selection and tag-version validation now come from `metadata.json`.

### Steps to Release

1. Update versions: Edit the target entry in `metadata.json`:

   ```json
   {
     "your-target": {
       "env": {
         "YOUR_TARGET_VERSION": "2.0.0"
       }
     }
   }
   ```

2. Test build: Verify that build works locally:

   ```bash
   make build your-target
   ```

3. Commit changes: Commit version updates:

   ```bash
   git add metadata.json
   git commit -m "Update your-target to 2.0.0"
   ```

4. Create tag: Create and push release tag. Use the target name as the tag prefix, except `apache-httpd`, which uses `httpd-`:

   ```bash
   git tag your-target-2.0.0.0
   git push origin your-target-2.0.0.0
   ```

5. CI automation: GitHub Actions automatically:
   - Validates tag format and version
   - Builds target
   - Scans for vulnerabilities (Trivy)
   - Uploads artifacts to GitHub Actions
   - Creates GitHub Release with `.tar.gz` package

### Release Tag Format

Tags MUST follow this format:

- `{target}-{version}[-{prerelease}].{revision}`
- target: Target tag prefix (e.g., nginx, haproxy, httpd)
- version: Official version from `metadata.json` (e.g., 1.28.2)
- prerelease: Optional pre-release suffix (e.g., beta, rc1)
- revision: Release revision suffix starting at 0, incrementing for rebuilds (e.g., 18)

Valid examples:

- `nginx-1.28.2.18` (nginx version 1.28.2, revision 18)
- `httpd-2.4.66.5` (apache-httpd version 2.4.66, revision 5)
- `haproxy-3.2.13-beta.0` (haproxy version 3.2.13-beta, revision 0)

Invalid examples:

- `nginx-1.28.2` (missing revision suffix)
- `custom-1.0.0.0` (unknown target)
- `nginx-1.28.2.x` (non-numeric revision)

## How It Works

1. `scripts/download.sh` resolves each target's download resources from `metadata.json`, then the build script runs Docker Buildx
2. Docker BuildKit executes the multi-stage Dockerfile via `docker buildx build`
3. Built artifacts go to `.out/<target>/` for both local builds and CI
4. Verify stages run inside UBI9 Micro, and the final exported artifact comes from the target's `scratch` stage

Build caching is automatically handled via root `.cache/<target>/` directories.

## CI Behavior

- Archive upload includes selected release files as a workflow artifact.
- Release upload is optional and packages selected release files into one `.tar.gz` per tag.
- Tag push release uses unified workflow `.github/workflows/release-from-tag.yaml`.
- Workflow automatically determines the target from tag pattern and calls reusable template `.github/workflows/template-release.yaml`.
- Template builds mapped target, scans for vulnerabilities, uploads the full `.out/<target>/` tree as artifact, then uploads `${tag}.tar.gz` that contains selected release files.

Selected release contents:

- `nginx`: `sbin/nginx`
- `nginx-resty-upstream-healthcheck`: `sbin/nginx`, `lualib/resty/core.lua`, `lualib/resty/core/`, `lualib/resty/upstream/`
- `haproxy`: `sbin/haproxy`
- `apache-httpd`: `bin/httpd`, `bin/rotatelogs`
- `coredns`: `coredns`
- `dnsmasq`: `sbin/dnsmasq`
- `vector`: `bin/vector`
- `monit`: `bin/monit`

## Logging Strategy

> [!NOTE]
> `apache-httpd` releases include both `bin/httpd` and `bin/rotatelogs` for piped logging support.
> For details, see [apache-httpd/AGENTS.md](apache-httpd/AGENTS.md).

## GitLab CI

- GitLab pipelines on `main` and `feature/*` expose manual package jobs that generate a child pipeline with `.gitlab/ci/package-pipeline.jsonnet`.
- `feature/*` branches append `-beta` to the package version.
- GitLab uploads to Package Registry only. GitHub uploads to Release only.
- The generated child pipeline includes the local component at `templates/static-release.yml`.
- GitHub release packaging and GitLab package generation both reuse `scripts/package-release.sh`.
