# static-builds

Build statically-linked binaries using Docker multi-stage builds for portable, minimal container deployments.

## Features

- Static Linking - Produce fully statically-linked binaries using musl libc.
- Multi-stage Builds - Use Docker BuildKit for efficient, cacheable builds.
- Minimal Outputs - Use a UBI10 Minimal verify stage for runtime checks and package final artifacts from a `scratch` stage.
- Extensible - Add new build targets by following a simple directory structure.
- Reproducible - Version-controlled configurations via `metadata.json`.

## Prerequisites

- Docker

## Usage

```bash
make build <target>
```

Build artifacts are written to `.out/<target>/` by default for both local and CI builds.
You can override this with `BUILD_OUTPUT_DEST`.

### Running haproxy binary image

```bash
docker run --rm \
  -v "$(pwd)/haproxy.cfg:/etc/haproxy/haproxy.cfg:ro" \
  <image> -c -f /etc/haproxy/haproxy.cfg
```

## Project Structure

```text
.
+-- metadata.json           # Canonical build/release metadata
+-- Makefile                # Local development build orchestration
+-- .gitlab-ci.yml          # GitLab CI configuration
+-- scripts/                # Shared build/release scripts
|   +-- download.sh         # Source download dispatcher
|   +-- metadata.sh         # Metadata query helper
|   +-- package-release.sh  # Release package helper
+-- .github/
|   +-- scripts/
|   |   +-- build.sh        # Docker Buildx build entry
|   |   +-- release-guard.sh # Release tag validator
|   +-- workflows/
|       +-- release-from-tag.yaml
|       +-- template-release.yaml
+-- .gitlab/
|   +-- ci/
|   |   +-- package-pipeline.jsonnet # GitLab child pipeline generator
|   +-- scripts/
|       +-- build-rootless.sh        # Rootless BuildKit build entry
+-- templates/              # GitLab CI components
|   +-- static-release.yml
+-- .tmp/                   # Downloaded source cache (gitignored)
+-- .cache/                 # Build cache (gitignored)
+-- .out/                   # Build outputs (gitignored)
|   +-- <target>/           # Local artifacts (for example `sbin/`, `bin/`)
+-- <target>/               # Build target directory
    +-- Dockerfile          # Multi-stage build definition
    +-- ...
```

## Contributing

Follow the [contribution guide](.github/CONTRIBUTING.md) for documentation style, validation, commit bodies, issues, and pull requests.

## Developer Guide

### Dependency Updates

`metadata.json` defines source versions and build image versions for every target.
Use the latest supported LTS release when upstream provides an LTS series.
Use the supported stable release when upstream does not provide an LTS series.
For nginx, follow the stable release series.
Use release tags for source archives and dependency references.
Use a maintained upstream branch when no suitable release tag exists.
Do not pin dependencies with commit hashes or container image digests.
The downloader refreshes branch archives on every call and reuses cached release archives.
Concurrent branch downloads use a lock for each destination.
Lock waits stop after 60 seconds.
Run `make build <target>` for each affected target before creating release tags.

Upstream projects use different support policies.
The following policies guide version selection.

| Dependency | Upstream Policy |
| --- | --- |
| HAProxy | Select the latest supported [LTS branch](https://www.haproxy.org/). |
| Abseil | Select the latest supported [LTS release](https://abseil.io/about/releases). |
| nginx | Select the [stable release series](https://nginx.org/en/download.html). |
| Go | Select a stable release covered by the [Go support policy](https://go.dev/doc/devel/release#policy). |
| Alpine | Select a stable branch covered by the [repository support policy](https://alpinelinux.org/releases/). |
| UBI | Follow the [UBI support lifecycle](https://access.redhat.com/support/policy/updates/ubi). |
| Other dependencies | Select stable releases when upstream provides no LTS series. |

GitHub release jobs use the `ubuntu-26.04` LTS runner label.
GitLab child pipelines select the release component through `CI_COMMIT_REF_NAME`, which names the current branch or tag.
The parent job expands the complete component reference before generating the child pipeline.

`.github/dependabot.yaml` checks GitHub Actions weekly.
It groups action updates into one pull request.
Dependabot does not update the custom version fields in `metadata.json`.
Docker image tags come from that metadata through build arguments, so Dependabot cannot resolve them from the Dockerfiles.
Update those values, GitLab CI dependencies, and the Markdown schema reference manually.

### Target Structure

Each target must follow this structure:

```text
<target>/
+-- Dockerfile        # Multi-stage build definition (required)
+-- README.md         # Target-specific documentation (optional)
```

### Adding a New Target

1. Create target directory: Create a new directory named after your target (e.g., `your-target/`).

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
         "ALPINE_VERSION": "3.24.2",
         "YOUR_TARGET_VERSION": "1.0.0",
         "UBI10_MINIMAL_VERSION": "10.2-1790753097"
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
   FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
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

6. Update release trigger mapping: Add the tag trigger and target selection case in `.github/workflows/release-from-tag.yaml`.
   Release file selection and official versions now come from `metadata.json`.

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

7. Validate: Run `make build your-target` to verify the download and build flow works.

### Allowed Target-Specific Variations

Targets share the same root contract, but some targets vary in builder image, release contents, or runtime packaging.

- Document approved target-specific variations in that target's `README.md`.
- Keep the root `README.md` focused on shared repository behavior.
- Treat `nginx`, `nginx-upstream-healthcheck`, `apache-exporter`, `apache-httpd`, `coredns`, `vector`, `haproxy`, `dnsmasq`, and `monit` differences as documented target profiles, not as undocumented exceptions.

### Best Practices

- Security hardening: Use static PIE builds (`-fPIE -pie`).
- Verification: Always include a verify stage with ELF checks, static linking verification, and strace validation.
- Caching: Use `--mount=type=cache` for Alpine/DNF caches.
- Documentation: Document approved target-specific variations in each target `README.md`.
- Version variables: Follow naming convention `{TARGET}_VERSION` for consistency inside `metadata.json`.

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

3. Commit version updates with a title and an explanatory body.
   In the editor, describe the reason, changes, actual verification results, and material limitations.

   ```bash
   git add metadata.json
   git commit
   ```

4. Create tag: Create and push release tag.
   Use the target name as the tag prefix, except `apache-httpd`, which uses `httpd-`:

   ```bash
   git tag your-target-2.0.0.0
   git push origin your-target-2.0.0.0
   ```

5. CI automation: GitHub Actions automatically:
   - Validates tag format and version.
   - Builds target.
   - Scans for vulnerabilities (Trivy).
   - Uploads artifacts to GitHub Actions.
   - Creates GitHub Release with `.tar.gz` package.

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

1. `scripts/download.sh` resolves each target's download resources from `metadata.json`, then the build script runs Docker Buildx.
2. Docker BuildKit executes the multi-stage Dockerfile via `docker buildx build`.
3. Built artifacts go to `.out/<target>/` for both local builds and CI.
4. Verify stages run inside UBI10 Minimal, and the final exported artifact comes from the target's `scratch` stage.

Build caching uses the root `.cache/<target>/` directories.

## CI Behavior

- Archive upload includes selected release files as a workflow artifact.
- Release upload is optional and packages selected release files into one `.tar.gz` per tag.
- Tag push release uses unified workflow `.github/workflows/release-from-tag.yaml`.
- Workflow automatically determines the target from tag pattern and calls reusable template `.github/workflows/template-release.yaml`.
- Template builds mapped target, scans for vulnerabilities, uploads the full `.out/<target>/` tree as artifact, then uploads `${tag}.tar.gz` that contains selected release files.

Selected release contents:

- `nginx`: `sbin/nginx` with njs and VTS modules compiled in.
- `nginx-upstream-healthcheck`: `sbin/nginx` with upstream healthcheck, njs and VTS modules compiled in.
- Both nginx targets include their scripting examples under `conf/`.
- The healthcheck target replaces the former `nginx-resty-upstream-healthcheck` target and release prefix.
- `haproxy`: `sbin/haproxy`
- `apache-exporter`: `bin/apache-exporter`
- `apache-httpd`: `bin/httpd`, `bin/rotatelogs`
- `coredns`: `coredns`
- `dnsmasq`: `sbin/dnsmasq`
- `vector`: `bin/vector`
- `monit`: `bin/monit`

## Logging Strategy

> [!NOTE]
> `apache-httpd` releases include both `bin/httpd` and `bin/rotatelogs` for piped logging support.

## GitLab CI

- GitLab pipelines on `main` and `feature/*` expose manual package jobs that generate a child pipeline with `.gitlab/ci/package-pipeline.jsonnet`.
- `feature/*` branches append `-beta` to the package version.
- GitLab uploads to Package Registry only.
  GitHub uploads to Release only.
- The generated child pipeline includes the local component at `templates/static-release.yml`.
- GitHub release packaging and GitLab package generation both reuse `scripts/package-release.sh`.
