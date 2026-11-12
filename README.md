# static-builds

Build statically-linked binaries using Docker multi-stage builds for portable, minimal container deployments.

## Features

- Static Linking - Produce fully statically-linked binaries using musl libc.
- Multi-stage Builds - Use Docker BuildKit for efficient, cacheable builds.
- Minimal Outputs - Verify artifacts with Docker Hub `redhat/ubi10-minimal` stages and export from a `scratch` stage.
- Extensible - Add new build targets by following a simple directory structure.
- Reproducible - Version-controlled configurations via `metadata.json`.

## Prerequisites

Local container builds require Docker Compose 2.17 or later and access to a running Linux Docker daemon.
The Compose tooling image supplies Go and rootless BuildKit, so Go does not need to be installed on the host.
Direct rootless builds and contributor checks still require the Go version declared in `go.mod`.
Local builds and both CI hosts use [.goreleaser.yaml](.goreleaser.yaml) to orchestrate building, metadata-selected staging, archiving, SHA-256 checksums, and release publication.
See the [state machine inventory](.github/state-machines.md), [Taskfile](Taskfile.yaml), [download command](cmd/download/main.go), and [metadata command](cmd/metadata/main.go).

Install `curl` or `wget` to download target sources, and allow network access to their upstream hosts.
The `wget` fallback also requires GNU Coreutils or BusyBox `timeout`, or GNU Coreutils `gtimeout` on macOS, to enforce the per-attempt wall-clock deadline.
Both downloaders honor `DOWNLOAD_MAX_TIME`, which defaults to 300 seconds per attempt.
Target download URLs and build arguments come from [metadata.json](metadata.json).
A clean build also requires builder network access to image registries and each target's package or language dependency repositories.
For example, the [nginx Dockerfile](nginx/Dockerfile) installs Alpine build packages.
The [CoreDNS Dockerfile](coredns/Dockerfile) also compiles Go code, which can require access to Go module sources.

Direct builds require a configured native Linux host with `buildkitd`, `rootlesskit`, `runc`, subordinate UID/GID mappings, and permission to create unprivileged user namespaces.
Run them as a non-root user; macOS and Windows hosts use Compose.
GitHub installs the module-selected official BuildKit daemon and distribution-packaged `/usr/bin/rootlesskit`, `uidmap`, and `runc` without changing runner security settings.
The selected worker must support local filesystem export, cache import/export, and execution of `BUILDKIT_PLATFORM`, which defaults to `linux/amd64`.
Use a native worker or separately configured emulation when the worker architecture differs from the selected platform.
The Taskfile writes artifacts to `dist/<target>/` and imports and exports local cache in `.cache/<target>/` by default.

The following tools are managed through `go.mod`:

- Task
- shfmt
- Jsonnet
- jsonnetfmt
- golangci-lint
- GoReleaser
- buildctl

The pnpm version declared in `package.json` provides equivalent package-script entrypoints.
Target compilers are installed in Dockerfile stages, so they are not host prerequisites.

## Usage

```sh
go tool task
go tool task download -- nginx
TARGET=nginx TAG=nginx-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate
```

Set `TARGET` and `TAG` for GoReleaser builds, and pass GoReleaser arguments after Task's `--` separator.
Run these commands from the repository root.
The equivalent pnpm commands are:

```sh
pnpm run tasks
pnpm run download -- nginx
TARGET=nginx TAG=nginx-local pnpm run build
```

Build artifacts are written to `dist/<target>/` by default for both local and CI builds.
You can override this with `BUILD_OUTPUT_DEST`.
Archives and checksum files are written separately under `dist/release/<target>/`.
The pnpm build script runs the same nonpublishing GoReleaser snapshot command.

### Independent targets

Each target owns its Dockerfile inputs, downloaded archives, mutable build caches, exported installation, release staging, and package directory.
The build task creates a private context containing only that target's source directory and nonempty completed archive files from its download directory.
Lock files and temporary downloads are excluded, and copying preserves source modes and symbolic links without following them outside the selected inputs.
The Dockerfile channel is limited to the same staged target directory.
The temporary context is removed after the daemon exits, and the root `.dockerignore` remains for the Compose tooling-image build.
Neither nginx target reads the other's configuration, account files, examples, binaries, or packages.
Both compile their own NGINX, njs, QuickJS-NG, and VTS sources using their own metadata entry.
Common base images and the Go tools remain reusable infrastructure, not artifacts produced by another target.
Explicit output or cache overrides must use different destinations when building different targets.

### Local container builds

Run a one-shot rootless build from the repository root:

```sh
mkdir -p .cache dist
docker compose build build
TARGET=dnsmasq docker compose run --rm build
```

Replace `dnsmasq` with the required target.
The service defaults to a nonpublishing GoReleaser snapshot with `TAG=local`.
Set `TAG=dnsmasq-local` alongside `TARGET` to name the archive `dist/release/dnsmasq/dnsmasq-local.tar.gz` and checksum `dist/release/dnsmasq/dnsmasq-local_checksums.txt`.
No target is selected when it is omitted.
The output remains under `dist/<target>/`, and source downloads and exported build cache remain under `.cache/downloads/<target>/` and `.cache/`.
The checkout is read-only inside the service except for the cache and distribution directories.
Named volumes retain the Go tool module cache, Go tool compilation cache, and target-specific BuildKit state directories.
The service does not mount the Docker socket, publish ports, use host networking, or run in privileged mode.

The image starts from `golang:alpine` and runs the upstream daemonless BuildKit script as a non-root user.
The tooling image reuses architecture-specific APK download caches while refreshing repository indexes whenever its package installation step runs.
Its native snapshotter avoids a FUSE device, and its default process sandbox remains enabled.
Following the [BuildKit rootless Docker configuration](https://github.com/moby/buildkit/blob/v0.34.2/docs/rootless.md#docker), Compose relaxes seccomp and masked system paths inside this dedicated build container.
Only run trusted build inputs in this container because these settings reduce its isolation.
An AppArmor-enabled Docker host additionally needs a compatible rootless AppArmor profile configured by its administrator.
Do not disable host security controls to work around a runner denial.

The builder itself runs on the Docker host's native architecture.
`BUILDKIT_PLATFORM` selects the target artifact architecture, not the architecture of the rootless daemon.
For example, `BUILDKIT_PLATFORM=linux/arm64 TARGET=dnsmasq docker compose run --rm build` builds an arm64 target when its dependencies support that platform.
Building a different architecture requires working emulation in the Docker host.
The `vector` target currently selects a fixed x86_64 upstream archive.

On native Linux, the bind-mounted directories must be writable by the image's user, whose default UID is 1000.
Set `BUILDER_UID=$(id -u)` when building the tooling image if the host uses another non-root UID.
Keep that UID consistent when reusing the named cache volumes.
The container does not recursively change ownership of the checkout.
Go VCS stamping is disabled for container tooling because a linked worktree's Git directory may be outside the mounted source tree.

### Build configuration

GoReleaser and its low-level build drivers accept inherited environment values.

| Variable | Default | Values and purpose |
| --- | --- | --- |
| `TARGET` | None | Require an explicit metadata target. |
| `TAG` | None; `local` in Compose | Name the archive and checksum file. |
| `BUILDKIT_PLATFORM` | `linux/amd64` | Select a target platform supported by the builder, such as `linux/arm64`. |
| `BUILDKIT_NETWORK` | `default` | Select `default`, `host`, or `none` for RUN-step networking. |
| `BUILDKIT_CACHE_BACKEND` | `local` | Select `local` or `gha` for cache import and export. |
| `BUILD_OUTPUT_DEST` | `dist/<target>/` | Select an artifact output directory relative to the repository root or an absolute path. |
| `CACHE_DIR` | `.cache/<target>/` | Select a local cache directory relative to the repository root or an absolute path when using the `local` backend. |

Build invocation examples:

- `TARGET=nginx TAG=nginx-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate`
- `BUILDKIT_PLATFORM=linux/arm64 TARGET=nginx TAG=nginx-local pnpm run build`
- `BUILD_OUTPUT_DEST=dist/custom/nginx CACHE_DIR=.cache/custom/nginx TARGET=nginx TAG=nginx-local pnpm run build`
- `TARGET=dnsmasq TAG=dnsmasq-local pnpm run build` on a configured native Linux host.

### Network and platform behavior

Every build uses the GoReleaser before hook and `go tool task build` to start an ephemeral rootless BuildKit daemon.
Taskfile selects the native snapshotter and a persistent per-target state directory under `${XDG_DATA_HOME:-$HOME/.local/share}/buildkit/`.
Each invocation creates a private temporary runtime directory for the daemon socket and removes it after the daemon exits.
Existing `BUILDKITD_FLAGS` remain available for explicitly configured host requirements; GitLab retains its documented process-sandbox exception.
`BUILDKIT_NETWORK=none` disables RUN-step networking.
`BUILDKIT_NETWORK=host` requests host networking from the selected builder.
The host request requires a daemon explicitly configured to allow `network.host` and the build client flag `--allow network.host`; the task does not grant either entitlement automatically.
The build does not automatically configure a Linux VM or cross-platform emulation.
A build always requires an explicit `TARGET` and fails without one.
The nginx commands in this document are examples, not defaults.
For low-level troubleshooting only, bypass GoReleaser with `go tool task build -- dnsmasq --progress=plain`.
This driver command accepts extra BuildKit client flags but does not stage release files, archive them, generate checksums, or publish releases.
Tool tasks also pass arguments through: `go tool task jsonnetfmt -- -i` formats the built-in pipeline file.
`go tool task buildctl -- --help` exposes the same Go-managed BuildKit client used by the build task.
`go tool task goreleaser -- <arguments>` runs the shared build and release configuration in [.goreleaser.yaml](.goreleaser.yaml).

### Build and package without publishing

Build one target, stage only its metadata-selected release paths, and create its archive and checksum without publishing:

```sh
TARGET=dnsmasq TAG=dnsmasq-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate
```

GoReleaser writes `dist/release/<TARGET>/<TAG>.tar.gz` and `dist/release/<TARGET>/<TAG>_checksums.txt`.
The command cleans only that target's GoReleaser output directory, preserving other targets' packages.
Serialize builds of the same target because its output, staging, and package directories are shared between invocations.
Generated SHA-256 checksums describe the distributed archive and do not enforce checksums on downloaded upstream sources.
Snapshot mode neither creates a tag nor publishes a release.
Build-producing entrypoints enable verbose hook output and use a six-hour GoReleaser timeout.
Use `--timeout` to choose a different limit, and remember that this cannot extend a CI runner or project timeout.
Publication-only jobs remain non-verbose and skip before hooks.
Verbose Git diagnostics can include credentialed remote URLs, so do not save or share unredacted logs from local credentialed checkouts.

GoReleaser OSS supports [build orchestration through before hooks](https://goreleaser.com/customization/general/hooks/), [archive checksums](https://goreleaser.com/customization/package/checksum/), and [GitHub or GitLab releases](https://goreleaser.com/customization/publish/scm/).
The configuration's string before hook validates the target and backend, invokes the selected Task driver, and stages the manifest-selected release paths after a successful build.
Task owns the low-level Docker invocation, while target Dockerfiles remain authoritative for compilation and artifact verification.
Staging uses `dist/release-files/<TARGET>/` outside GoReleaser's distribution directory because before hooks run before distribution initialization.
The hook clears that target's staging directory only after validation and a successful build.
GoReleaser then archives the staged files and generates their archive checksum.
Its mixed-language targets do not use GoReleaser's native Go builder or the Pro-only prebuilt builder.
The pinned GoReleaser OSS version does not template its distribution directory.
The Task entrypoint validates the target, renders only the distribution path into a temporary copy of the shared configuration, and removes that copy on exit.
This also runs before publication with hooks skipped, so missing or unknown targets cannot clean another target's distribution directory.
Use the named Task entrypoint rather than invoking GoReleaser release directly against the shared configuration.

Both hosts invoke GoReleaser once in the build job instead of separate build and package commands.
Their separate publication jobs receive the built archive, set `RELEASE_ARCHIVE` to its preserved copy, and run GoReleaser with `--skip=before,archive,validate` to recompute the checksum and publish.
No build or staging hook runs under publishing credentials.
GitLab branch jobs use the same nonpublishing snapshot flow and retain artifacts without creating GitLab Releases.
GitLab release-tag pipelines build one manifest-selected target, then publish its archive and checksum in a separate GoReleaser job.
The publisher uses the existing `CI_JOB_TOKEN` with [GoReleaser's job-token authentication](https://goreleaser.com/customization/publish/scm/gitlab/) and stores assets in the Generic Package Registry.
It selects GitLab explicitly with `GORELEASER_FORCE_TOKEN=gitlab`, reads the repository from the GitLab checkout's origin, and uses `CI_API_V4_URL` and `CI_SERVER_URL` for the API and download host.
No persistent token is configured and no build is repeated in the publisher.
The triggering user must already have permission to publish the release and access any protected tag, and the project must allow package uploads.
Select only one release provider per invocation, and do not use a GitLab draft release as a nonpublishing test because GitLab has no draft-release mode.

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
+-- compose.yaml            # Local Go and rootless BuildKit environment
+-- go.mod                  # Go commands and toolchain tools
+-- go.sum                  # Go module checksums
+-- .gitlab-ci.yaml         # GitLab CI configuration
+-- cmd/                    # Source download and manifest query commands
|   +-- download/           # Source download command
|   +-- metadata/           # Manifest/version metadata query command
+-- internal/               # Shared manifest parser
+-- .github/workflows/      # Release and contributor workflows
+-- .gitlab/ci/             # GitLab child pipeline generator
+-- templates/              # GitLab CI components
|   +-- static-release.yml
+-- .cache/                 # Reusable build and source caches (gitignored)
|   +-- downloads/          # Source archives and transfer locks
+-- dist/                   # Build outputs (gitignored)
|   +-- <target>/           # Local artifacts (for example `sbin/`, `bin/`)
|   +-- release-files/      # Metadata-selected staging, outside GoReleaser dist
|   +-- release/            # Archives and SHA-256 checksums
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
Each target's `env` declares `"UBI10_MINIMAL_VERSION": "10.2"`.
The Taskfile `build` task reads it with `go tool task manifest -- get-env <target>` and passes it to BuildKit as `--opt build-arg:UBI10_MINIMAL_VERSION=10.2`.
Each Dockerfile declares `ARG UBI10_MINIMAL_VERSION` and consumes the value in `FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify`.
The registry-implicit `redhat/ubi10-minimal` reference resolves through Docker Hub.
Use the latest supported LTS release when upstream provides an LTS series.
Use the supported stable release when upstream does not provide an LTS series.
For nginx, follow the stable release series.
Use release tags for source archives and dependency references.
Use a maintained upstream branch when no suitable release tag exists.
Do not pin dependencies with commit hashes or container image digests.
The downloader refreshes branch archives on every call and reuses cached release archives.
Concurrent branch downloads use a lock for each destination.
Lock waits stop after 60 seconds.
Run `TARGET=<target> TAG=<target>-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate` for each affected target before creating release tags.

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
   ADD ".cache/downloads/your-target/your-target-${YOUR_TARGET_VERSION}.tar.gz" /build/
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
   The `resolve-target` job derives target selection from `metadata.json` through `go tool task manifest -- get-target-from-tag <tag>`.
   The reusable release workflow validates the tag against the manifest-selected prefix and official version.
   Keep that resolution job and its reusable release call unchanged.
   Release file selection and official versions also come from `metadata.json`.

   ```yaml
   on:
     push:
       tags:
         - 'your-target-*'
   ```

   Add this filter to the existing `on.push.tags` list rather than replacing the caller workflow.

7. Validate: Run `TARGET=your-target TAG=your-target-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate` to verify the download and build flow works.

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

Run `go tool task test` and `pnpm check` for the surviving Go commands and contributor tools.
Go lint uses `go tool task golangci-lint -- run ./...` checks, and shell formatting uses `go tool task shfmt -- -d` through Task.
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
   TARGET=your-target TAG=your-target-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate
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
   - Runs GoReleaser to build the target, stage release files, archive them, and generate SHA-256 checksums.
   - Scans for vulnerabilities (Trivy).
   - Uploads artifacts to GitHub Actions.
   - Publishes the existing `.tar.gz` archive and checksum with GoReleaser in a separate job.

### Release Tag Format

Push release tags to each host where a release should be published.
GitHub and GitLab do not replicate tags or Releases automatically.
GitLab release publication depends on available CI capacity and a runner that supports the documented rootless build requirements.

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

1. GoReleaser runs its before hook, which selects the Task build driver and runs `go tool task download -- <target>` to resolve the target's download resources from `metadata.json` before starting the build.
2. The build task invokes the upstream daemonless rootless BuildKit script to execute the multi-stage Dockerfile.
3. Built artifacts go to `dist/<target>/` for both local builds and CI.
4. Each target verifies built artifacts in its Dockerfile before exporting its `scratch` stage.
5. The GoReleaser hook stages only metadata-selected release files, then GoReleaser creates the archive and SHA-256 checksum.
6. Separate CI publication jobs use GoReleaser to publish the existing archive without rebuilding.

Local build caching uses the root `.cache/<target>/` directories by default.
GitHub release builds use the GitHub Actions cache backend with a separate scope for each target.
Compose and GitLab package builds use daemonless rootless BuildKit and retain local-cache behavior.

## CI Behavior

- Build-relevant pull requests run the [nonpublishing build validation workflow](.github/workflows/validate-builds.yaml) for all metadata targets on native amd64 and all except the x86_64-only Vector target on native arm64.
- Each isolated job runs the GoReleaser snapshot flow, including Dockerfile verification, then checks the generated SHA-256 checksum and exact metadata-selected archive files, bytes, and modes before uploading an architecture-named package artifact.
- Build validation uses read-only repository access and ephemeral local caches, without release credentials, shared release caches, tag creation, or publication.
- Archive upload includes selected release files as a workflow artifact.
- GoReleaser creates one `.tar.gz` and SHA-256 checksum per tag during the build job, with publication handled separately.
- Tag push release uses unified workflow `.github/workflows/release-from-tag.yaml`.
- Workflow automatically determines the target from tag pattern and calls reusable template `.github/workflows/template-release.yaml`.
- Template builds the mapped target, scans for vulnerabilities, uploads the full `dist/<target>/` tree as an artifact, then uploads `${tag}.tar.gz` and `${tag}_checksums.txt`.

Selected release contents:

- `nginx`: `sbin/nginx` with njs and VTS modules compiled in.
- `nginx-upstream-healthcheck`: `sbin/nginx` with upstream healthcheck, njs and VTS modules compiled in.
- Both nginx archives contain only the executable; optional njs and healthcheck examples remain in the repository.
- Nginx images use `config/nginx.conf` and `config/conf.d/`, and preserve user-provided configuration mounted at those locations.
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
- The GitLab check job runs Go-managed contributor checks before package generation and retains the manual parent trigger.
- GitLab rejects merge-request pipelines at the workflow boundary and keeps existing branch and tag pipelines.
- A branch push with an open merge request runs only its branch pipeline.
- GitLab pipelines on `main` and `feature/*` generate a child configuration with `.gitlab/ci/package-pipeline.gitlab-ci.jsonnet`.
- One manual parent trigger starts branch child pipelines, which build selected targets and retain their artifacts.
- Tag pipelines resolve one target from metadata and validate the official version and numeric revision before generating the child pipeline.
- The tag child pipeline runs automatically and publishes only after its package job succeeds.
- The parent trigger mirrors child pipeline status so failed builds or publication fail the parent pipeline.
- The package component uses `golang:alpine` and invokes the shared GoReleaser rootless build flow.
- When `curl` is absent, `go tool task download -- <target>` uses the image's BusyBox `wget` and `timeout` fallback.
- Contributor validation and pipeline generation use `golang:alpine` with Go's default `GOTOOLCHAIN=auto` behavior honoring the `go.mod` directive.
- The prepare job exports the rendered child pipeline, and package jobs invoke named Taskfile tasks without a separate toolchain download.
- GitLab branch package builds retain artifacts without publishing Releases.
- Tag publishers copy the archive from the successful package job before GoReleaser cleans its output directory and recomputes the archive checksum.
- GoReleaser uploads the archive and checksum as generic packages and links them from the GitLab Release.
- GitLab contributor validation invokes Go tests, golangci-lint, shell formatting, and Jsonnet checks through `go tool task`.
- The Node, pnpm, and Java contributor checks run in local hooks only.
- The package component installs BuildKit, fuse-overlayfs, runc, RootlessKit, and UID mapping helpers from Alpine packages.
- GitLab caches signed APK files under `.cache/apk/<Alpine-release>/<architecture>/` with a runner-isolated cache key.
- The shared installer refreshes repository indexes before each install, and caches neither indexes nor installed system directories.
- GitHub retains the existing Go module cache and BuildKit GHA layer cache; it does not install APK packages on its Ubuntu host or persist APK cache mounts.
- It creates user 1000 and maps `user:100000:65536` in both `/etc/subuid` and `/etc/subgid`.
- It invokes the upstream daemonless script as that user without Docker services.
- The runner must permit unprivileged user namespaces and rootless mounts.
- `feature/*` branches append `-beta` to the package version.
- Build orchestration, release-file staging, packaging, generated archive checksums, and tag-triggered publication use GoReleaser on both hosts.
- The generated child pipeline includes the local component at `templates/static-release.yml`.
- GitLab component discovery requires the conventional `templates/*.yml` filename, so `templates/static-release.yml` remains the supported exception to the `.yaml` filename rule.
- The parent configuration uses the supported custom configuration path documented by [GitLab](https://docs.gitlab.com/ci/pipelines/settings/#specify-a-custom-cicd-configuration-file), while the local component follows [GitLab's component discovery convention](https://docs.gitlab.com/ci/components/).
- The GoReleaser before hook queries `go tool task manifest -- get-release-files <target>` for selected paths.
- Both hosts export the GoReleaser archive and its SHA-256 checksum file.

GitHub publishing uses GoReleaser with `GITHUB_TOKEN`.
GoReleaser creates the release or replaces its existing archive asset.
The configuration uses the tag as the release title and publishes releases with `draft: false` and `prerelease: "false"`.
Reruns apply this configuration to existing releases.
The workflow validates the repository tag format before using GoReleaser with `--skip=validate` for target-prefixed tags.
