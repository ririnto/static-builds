# static-builds

Build statically-linked binaries using Docker multi-stage builds for portable, minimal container deployments.

## Features

- Static Linking - Produce fully statically-linked binaries using musl libc.
- Multi-stage Builds - Use Docker BuildKit for efficient, cacheable builds.
- Minimal Outputs - Verify artifacts with Docker Hub `redhat/ubi10-minimal` stages and export from a `scratch` stage.
- Extensible - Add new build targets by following a simple directory structure.
- Reproducible - Version-controlled configurations via `metadata.json`.

## Prerequisites

Local builds require the Go version declared in `go.mod` and Docker CLI access to a running Linux daemon and the Docker Buildx plugin.
Release packaging uses the host `tar` to preserve archive ordering and metadata.
See the [Taskfile](Taskfile.yaml), [download command](cmd/download/main.go), and [metadata command](cmd/metadata/main.go).

Install `curl` or `wget` to download target sources, and allow network access to their upstream hosts.
The `wget` fallback also requires GNU Coreutils or BusyBox `timeout`, or GNU Coreutils `gtimeout` on macOS, to enforce the per-attempt wall-clock deadline.
Both downloaders honor `DOWNLOAD_MAX_TIME`, which defaults to 300 seconds per attempt.
Target download URLs and build arguments come from [metadata.json](metadata.json).
A clean build also requires builder network access to image registries and each target's package or language dependency repositories.
For example, the [nginx Dockerfile](nginx/Dockerfile) installs Alpine build packages.
The [CoreDNS Dockerfile](coredns/Dockerfile) also compiles Go code, which can require access to Go module sources.

Confirm that plugin with `docker buildx version` and inspect the active builder with `docker buildx inspect --bootstrap`.
The selected worker must support local filesystem export, cache import/export, and execution of `BUILDKIT_PLATFORM`, which defaults to `linux/amd64`.
Use a native worker or separately configured emulation when the worker architecture differs from the selected platform.
The Taskfile writes artifacts to `.out/<target>/` and imports and exports local cache in `.cache/<target>/` by default.

Python 3 is needed for `go tool task list-targets`, not for `go tool task build TARGET=<target>`.
Task, shfmt, Jsonnet, jsonnetfmt, and golangci-lint are managed through `go.mod`.
The pnpm version declared in `package.json` provides equivalent package-script entrypoints.
Target compilers are installed in Dockerfile stages, so they are not host prerequisites.

## Usage

```sh
go tool task
go tool task list-targets
go tool task download TARGET=nginx
go tool task build TARGET=nginx
```

Task uses named variables rather than Make's additional-goal convention.
Use `go tool task build TARGET=<target>`, not `go tool task build <target>`.
Run these commands from the repository root.
The equivalent pnpm commands are:

```sh
pnpm run tasks
pnpm run list-targets
pnpm run download TARGET=nginx
pnpm run build TARGET=nginx
```

Build artifacts are written to `.out/<target>/` by default for both local and CI builds.
You can override this with `BUILD_OUTPUT_DEST`.
Build configuration accepts inherited environment values or Task CLI assignments.
Task CLI assignments take precedence over inherited values.
`BUILDKIT_PLATFORM`, `BUILDKIT_NETWORK`, and `BUILDKIT_CACHE_BACKEND` default to `linux/amd64`, `default`, and `local`.
`BUILD_OUTPUT_DEST` and `CACHE_DIR` default to `.out/<target>/` and `.cache/<target>/` under the repository root.

```sh
BUILDKIT_PLATFORM=linux/arm64 go tool task build TARGET=nginx
go tool task build TARGET=nginx BUILD_OUTPUT_DEST=.out/custom/nginx CACHE_DIR=.cache/custom/nginx
```

### Network and platform behavior

All hosts use the configured Docker Buildx builder directly.
`BUILDKIT_NETWORK=none` disables RUN-step networking.
`BUILDKIT_NETWORK=host` requests host networking from the selected builder.
The build does not automatically configure a Linux VM or cross-platform emulation.
Pass additional Buildx arguments with `go tool task build TARGET=<target> -- <buildx arguments>`.
A build always requires an explicit `TARGET` and fails without one.
The nginx commands in this document are examples, not defaults.

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
+-- Taskfile.yaml           # Repository task entrypoints
+-- go.mod                  # Go commands and toolchain tools
+-- go.sum                  # Go module checksums
+-- .gitlab-ci.yaml         # GitLab CI configuration
+-- cmd/                    # Source download and manifest query commands
|   +-- download/           # Source download command
|   +-- metadata/           # Manifest/version metadata query command
|   +-- package-release/    # Metadata-selected release packaging
|   +-- release-guard/      # Release tag validation
|   +-- release-resolve/    # Tag-to-target resolution
|   +-- release-publish/    # go-github release publication
+-- internal/               # Shared manifest parser
+-- .github/workflows/      # Release and contributor workflows
+-- .gitlab/ci/             # GitLab child pipeline generator
+-- templates/              # GitLab CI components
|   +-- static-release.yml
+-- .tmp/                   # Downloaded source cache (gitignored)
+-- .cache/                 # Build cache (gitignored)
+-- .out/                   # Build outputs (gitignored)
|   +-- <target>/           # Local artifacts (for example `sbin/`, `bin/`)
+-- nginx/                  # nginx Dockerfile and scripting examples
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
Run `go tool task build TARGET=<target>` for each affected target before creating release tags.

Upstream projects use different support policies.
The following policies guide version selection.

| Dependency | Upstream Policy |
| --- | --- |
| HAProxy | Select the latest supported [LTS branch](https://www.haproxy.org/). |
| Abseil | Select the latest supported [LTS release](https://abseil.io/about/releases). |
| nginx | Select the [stable release series](https://nginx.org/en/download.html). |
| Go | Select a stable release covered by the [Go support policy](https://go.dev/doc/devel/release#policy). |
| Alpine | Select a stable branch covered by the [repository support policy](https://alpinelinux.org/releases/). |
| Red Hat UBI | Select a supported version of the UBI minimal image. |
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
         "ALPINE_VERSION": "3",
         "YOUR_TARGET_VERSION": "1.0.0"
       }
     }
   }
   ```

3. Create Dockerfile: Implement a multi-stage build:

   ```dockerfile
   ARG ALPINE_VERSION

   # Build stage
   FROM alpine:${ALPINE_VERSION} AS build
   ARG YOUR_TARGET_VERSION
   ADD ".tmp/your-target-${YOUR_TARGET_VERSION}.tar.gz" /build/
   # ... build steps ...

   # Verify built artifacts in a Dockerfile stage before export.
   FROM build AS verify
   RUN /your-target --version

   # Final stage
   FROM scratch
   COPY --from=verify /your-target /target/your-target
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

6. Extend the tag trigger filter in [.github/workflows/release-from-tag.yaml](.github/workflows/release-from-tag.yaml) with the target's `tag_prefix` followed by `-*`.
   The existing `resolve-target` job derives target selection from `metadata.json` through `go tool task manifest QUERY=get-target-from-tag VALUE=<tag>` and validates the tag with `go tool task release-guard TARGET=<target> TAG=<tag>`.
   Keep that resolution job and its reusable release call unchanged.
   Release file selection and official versions also come from `metadata.json`.

   ```yaml
   on:
     push:
       tags:
         - 'your-target-*'
   ```

   Add this filter to the existing `on.push.tags` list rather than replacing the caller workflow.

7. Validate: Run `go tool task build TARGET=your-target` to verify the download and build flow works.

### Allowed Target-Specific Variations

Targets share the same root contract, but some targets vary in builder image, release contents, or runtime packaging.

- Document approved target-specific variations in that target's `README.md`.
- Keep the root `README.md` focused on shared repository behavior.
- Treat `nginx`, `nginx-upstream-healthcheck`, `apache-exporter`, `apache-httpd`, `coredns`, `vector`, `haproxy`, `dnsmasq`, and `monit` differences as documented target profiles, not as undocumented exceptions.

### Best Practices

- Security hardening: Use static PIE builds (`-fPIE -pie`).
- Outputs: Verify built artifacts in Dockerfile stages before exporting them without separate verification scripts.
- Caching: Use `--mount=type=cache` for Alpine/DNF caches.
- Documentation: Document approved target-specific variations in each target `README.md`.
- Version variables: Follow naming convention `{TARGET}_VERSION` for consistency inside `metadata.json`.

### Contributor checks

Run `go tool task test-go` and `pnpm check` for the surviving Go commands and contributor tools.
Go lint uses standard `go tool task lint-go` checks, and shell formatting uses `go tool task shfmt-check` through Task.
Builds verify artifacts in Dockerfile stages without per-target verify scripts or separate runtime harnesses.
Contributor checks do not establish runtime acceptance of the custom healthcheck module.

## Release Process

### Creating a Release

Releases are triggered by Git tags following the pattern `<target>-<version>[-<prerelease>].<revision>`:

- Format: `{target}-{official_version}[-{prerelease}].{revision}`
- Example: `nginx-1.28.2.18` (target: nginx, version: 1.28.2, revision: 18)
- Validation: Release tags are validated against `metadata.json`

Release tag event filters remain static in `.github/workflows/release-from-tag.yaml` as required by GitHub.
Target selection, tag-prefix exceptions, release-file selection, and tag-version validation use `metadata.json`.

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
   go tool task build TARGET=your-target
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

1. Task runs `go tool task download TARGET=<target>` to resolve the target's download resources from `metadata.json` before starting the build.
2. The shell build entrypoint invokes Docker Buildx directly to execute the multi-stage Dockerfile.
3. Built artifacts go to `.out/<target>/` for both local builds and CI.
4. Each target verifies built artifacts in its Dockerfile before exporting its `scratch` stage.

Local build caching uses the root `.cache/<target>/` directories by default.
GitHub release builds use the GitHub Actions cache backend with a separate scope for each target.
GitLab package builds use the same direct Buildx command and retain local-cache behavior.

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
- `haproxy`: `sbin/haproxy`
- `apache-exporter`: `bin/apache-exporter`
- `apache-httpd`: `bin/httpd`, `bin/rotatelogs`
- `coredns`: `coredns`
- `dnsmasq`: `sbin/dnsmasq`
- `vector`: `bin/vector`
- `monit`: `bin/monit`

## Logging Strategy

> [!NOTE]
>
> `apache-httpd` releases include both `bin/httpd` and `bin/rotatelogs` for piped logging support.

## GitLab CI

- Configure the GitLab project to use `.gitlab-ci.yaml` as its custom CI/CD configuration path after merge.
- The GitLab check job runs the complete contributor check entrypoint before package generation and retains the manual parent trigger for publication.
- GitLab pipelines on `main` and `feature/*` generate a child configuration with `.gitlab/ci/package-pipeline.jsonnet`.
- One manual parent trigger starts the child pipeline, which builds and publishes its selected targets automatically.
- The package component uses `golang:alpine` and invokes the Buildx shell entrypoint through Task with a TLS-enabled Docker service.
- When `curl` is absent, `go tool task download TARGET=<target>` uses the image's BusyBox `wget` and `timeout` fallback.
- Contributor validation and pipeline generation use `golang:alpine` with Go's default `GOTOOLCHAIN=auto` behavior honoring the `go.mod` directive.
- The prepare job exports the rendered child pipeline, and package jobs invoke named Taskfile tasks without a separate toolchain download.
- GitLab package builds retain artifacts without custom publisher jobs.
- GitLab contributor validation runs Go tests, golangci-lint, shell formatting, Jsonnet, code-fence, and relative-link checks directly without installing Node or pnpm.
- The Node and pnpm contributor suite remains in the existing GitHub workflow and local hooks.
- The package component installs Docker CLI and Buildx from Alpine packages.
- The runner must provide the Docker service and its shared TLS certificate directory without disabling TLS or build sandbox protections.
- `feature/*` branches append `-beta` to the package version.
- Release publication remains in the existing Go release tool on GitHub.
- The generated child pipeline includes the local component at `templates/static-release.yml`.
- GitLab component discovery requires the conventional `templates/*.yml` filename, so `templates/static-release.yml` remains the supported exception to the `.yaml` filename rule.
- The parent configuration uses the supported custom configuration path documented by [GitLab](https://docs.gitlab.com/ci/pipelines/settings/#specify-a-custom-cicd-configuration-file), while the local component follows [GitLab's component discovery convention](https://docs.gitlab.com/ci/components/).
- GitHub release packaging and GitLab package generation both reuse `go tool task package-release TARGET=<target> PACKAGE=<name>`.
GitHub publishing uses go-github with `GH_TOKEN` and `GH_REPO`, falling back to `GITHUB_REPOSITORY` for the repository binding.
A missing release is created only after HTTP 404, and an existing archive asset is replaced before upload.
