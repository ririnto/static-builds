# PROJECT KNOWLEDGE BASE

## OVERVIEW

Docker multi-stage build system for statically-linked binaries.
Builds nginx, nginx-upstream-healthcheck, haproxy, apache-exporter, apache-httpd, coredns, dnsmasq, vector, monit using musl libc.

## STRUCTURE

```text
static-builds/
+-- metadata.json           # Canonical build/release metadata
+-- build-env.sh            # Shared build argument normalization
+-- Makefile                # Local development build orchestration
+-- .gitlab-ci.yml          # GitLab CI configuration
+-- download.sh             # Source download dispatcher
+-- metadata.sh             # Metadata query helper
+-- metadata.awk            # Metadata JSON parser
+-- package-release.sh      # Release package helper
+-- package.json            # Contributor check and fix scripts
+-- pnpm-lock.yaml          # Locked Node-side tooling
+-- pnpm-workspace.yaml     # Targeted tooling dependency overrides
+-- .husky/                 # Contributor Git hooks
+-- .github/
|   +-- build.sh            # Docker Buildx build entry
|   +-- release-guard.sh    # Release tag validator
|   +-- workflows/
|       +-- release-from-tag.yaml
|       +-- template-release.yaml
+-- .gitlab/
|   +-- ci/
|   |   +-- package-pipeline.jsonnet # GitLab child pipeline generator
|   +-- build-rootless.sh   # Rootless BuildKit build entry
+-- templates/              # GitLab CI components
|   +-- static-release.yml
+-- proxy/
|   +-- tests/              # Standalone healthcheck functional verification
+-- .cache/                 # Build cache (gitignored)
+-- .tmp/                   # Downloaded source cache (gitignored)
+-- .out/                   # Build outputs (gitignored)
+-- nginx/                  # Plain nginx target and shared nginx helpers
+-- nginx-upstream-healthcheck/ # Upstream healthcheck nginx target
+-- haproxy/
+-- apache-exporter/
+-- apache-httpd/
+-- coredns/
+-- dnsmasq/
+-- monit/
+-- vector/
```

## WHERE TO LOOK

| Task | Location | Notes |
| --- | --- | --- |
| Add new target | Root + create dir | Follow nginx/ pattern |
| CI release config | .github/workflows/, .gitlab-ci.yml, .gitlab/ci/, templates/ | GitHub workflows + GitLab component pipeline |
| Build definition | */Dockerfile | Multi-stage target build |
| Shared nginx helpers | nginx/*.sh | Both nginx targets reuse these build and verification files. |
| Shared Make helpers | build-env.sh, download.sh, metadata.sh, metadata.awk, package-release.sh | Root-level source and package helpers. |
| Build metadata | metadata.json | tag_prefix, release_files, env, downloads |
| Functional healthcheck verification | proxy/tests/ | Separate from target Docker verification. |

## CONVENTIONS

- New top-level directories outside STRUCTURE MUST NOT be added (for example, `docs/`, `tests/`).
- EditorConfig: 4-space indent (2 for .sh/.yaml)
- Prefer explicit loop conditions and positive guarded blocks over mid-function returns, `continue`, and loop `break` statements.
  Use plain `if` / `else if` / `else` chains for repeated guards when evaluation order and cleanup remain identical.
  Preserve natural final returns, required failure propagation, and switch fall-through prevention.
  Preserve cancellation, resource ownership, concurrency admission, and short-circuit evaluation without artificial flags or deeper nesting.
  Existing lint rules and formatter output take precedence over these control-flow preferences.
- Keep scripts and executable source in separate files and reference them from Dockerfiles, configurations, and documentation.
  Do not embed script bodies in those files.
- `metadata.json` MUST be the canonical source of build and release metadata for all targets.
- Versions MUST use the latest supported upstream LTS series when one exists.
- Projects without an upstream LTS series MUST use supported stable releases when available.
- Dependency references MUST use release tags or maintained upstream branches, never commit hashes or container image digests.
- Branch source archives MUST be refreshed on each download call.
- Target directories MUST have a Dockerfile.
  Target-specific download metadata MUST live under that target's `metadata.json` `downloads` entries, and downloader output MUST land directly under root `.tmp/`.
- Upstream source downloads MUST NOT enforce checksum verification/pinning because some upstreams do not publish checksum files.
  Consumers SHOULD validate sources independently when possible.
- Release workflow MUST use `.github/workflows/release-from-tag.yaml` as the only tag-triggered entrypoint and MUST delegate build/release logic to `.github/workflows/template-release.yaml`.
- Release target mapping, release-file selection, and tag-prefix exceptions MUST be maintained in `metadata.json` as the single source of truth.
- Release workflow MUST run Trivy filesystem scanning and MUST upload SARIF results to GitHub Security.
- Release jobs MUST request the minimum required GitHub permission.
  `contents: write` MAY be used only for jobs that publish releases or upload release assets.
- Common static verification logic duplicated across target verify stages MUST be updated in every affected `*/Dockerfile` verify block in the same change.
  Any repository documentation that describes that shared verification contract SHOULD be updated at the same time.
- Allowed target-specific variations MUST be documented in each target's `README.md`.
  Root-level documentation and policy files MUST treat those differences as approved target profiles, not as undocumented exceptions.
- Shell scripts (`*.sh` and files with a `sh`/`bash` shebang) MUST NOT contain comments except:

  - The shebang line (the first line starting with `#!`).
  - Function documentation comment blocks that are placed immediately above a function definition.

- Function documentation comment blocks MUST:

  - Be contiguous lines starting with a single `#`.
  - Have no blank line between the comment block and the function definition.
  - Describe purpose and expected inputs/outputs/return codes.
  - Use `@param`, `@return`, and `@return-type` tag lines where applicable.
- Inline comments, section header comments, and file header comment blocks MUST NOT be used.
- If explanation is needed, refactor code into a function and document that function instead of adding inline comments.

## Checksum Policy

The repository MUST NOT enforce checksum verification or pinning for upstream source downloads.

### Rationale

- Many upstream projects do not publish checksum files
- Enforcing checksum verification would prevent building targets with legitimate but unsigned releases
- Consumers are responsible for validating sources independently when checksum files are available

### Implications

- Build scripts MUST NOT fail if checksum files are not provided by upstream
- Release artifacts MUST NOT require checksum verification as part of the build process
- This policy ensures consistency across all targets regardless of upstream practices

## ANTI-PATTERNS (THIS PROJECT)

- No target builds or releases in push/PR validation CI (documentation validation only).
- No test framework - build infrastructure only

## UNIQUE STYLES

- Tag-triggered release: `nginx-1.28.2.18` builds and uploads the plain nginx artifact.
  `nginx-upstream-healthcheck-1.28.3.0` builds and uploads the upstream healthcheck nginx artifact.
- Release tags MUST follow `<target>-<official_version>[-<prerelease>].<x>`.

  - `official_version`: version from `metadata.json` (for example `NGINX_VERSION`, `HTTPD_VERSION`, `HAPROXY_VERSION`)
  - `prerelease`: optional pre-release suffix (for example `-beta`, `-rc1`)
  - `x`: release revision suffix starting at `0` and incrementing (`.0`, `.1`, `.2`, ...)
  - examples: `nginx-1.28.2.18`, `nginx-upstream-healthcheck-1.28.3.0`, `httpd-2.4.66.5`, `haproxy-3.2.13-beta.0`

- Unified caller workflow + reusable template pattern

  - caller: `.github/workflows/release-from-tag.yaml`
  - template: `.github/workflows/template-release.yaml`
- Artifacts: local builds and CI/release builds both output under `.out/<target>/...`.
- GitLab package pipelines on `main` and `feature/*` SHOULD use a single manual parent job that generates a child pipeline via `.gitlab/ci/package-pipeline.jsonnet` and reuses `templates/static-release.yml`.
- GitLab pipelines MUST upload to Package Registry only.
  `feature/*` branches MUST append `-beta` to the package version.

## CONTRIBUTION DOCUMENTS

Follow [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md) for prose, YAML filenames, commits, issues, and pull requests.

## DEVELOPMENT LIFECYCLE

- Define the affected targets and observable acceptance evidence before changing source or tooling.
- Complete each bounded change with its applicable checks before starting dependent work.
- Reproduce failures through the real build or verifier entrypoint and fix their root causes.
- Test observable behavior rather than implementation details or repeated configuration values.
- Inspect exported binaries and release package members when changing build or packaging behavior.
  Static checks alone do not establish runtime acceptance.
- Review the complete posted diff and confirm its current head before normal server-side integration.
- Record actual, reused, failed, and unrun evidence separately.
- Confirm named `main` contains the merge and the issue closes before declaring delivery complete.
- Stop task-owned processes and descendants before removing unused temporary resources.
  Preserve shared resources, unique uncommitted work, and artifacts needed for recovery.

These rules adapt the applicable [pstack lifecycle principles](https://github.com/cursor/plugins/tree/main/pstack).
The [Sinon Harness tooling](https://github.com/ririnto/sinon/tree/main/plugins/harness/tooling) supplies the relevant pnpm baseline.
This repository has no browser frontend, React application, or Gradle project.
Apply target build verification rather than frontend or Gradle tooling.

## COMMANDS

```bash
pnpm install --frozen-lockfile
pnpm run check
pnpm run fix
make list-targets
make build nginx
make build nginx-upstream-healthcheck
make download nginx
```

## NOTES

- Build caching via root `.cache/<target>/` directory
- Uses Docker Buildx with BuildKit
