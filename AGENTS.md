# PROJECT KNOWLEDGE BASE

## OVERVIEW

Docker multi-stage build system for statically-linked binaries.
Builds nginx, haproxy, apache-httpd, coredns, dnsmasq, vector,
monit using musl libc.

## STRUCTURE

```text
static-builds/
├── build.sh              # Main build entry
├── download.sh           # Source download dispatcher
├── Makefile              # Build orchestration (7 targets)
├── .github/
│   ├── scripts/common.sh    # Shared functions
│   └── workflows/           # Unified tag-triggered release
│       ├── release-from-tag.yaml
│       └── template-release.yaml
├── out/                   # Local build outputs (gitignored)
├── nginx/                # Target dirs: Dockerfile + .env
├── haproxy/
├── apache-httpd/
├── coredns/
├── dnsmasq/
├── monit/
└── vector/
```

## WHERE TO LOOK

| Task | Location | Notes |
| --- | --- | --- |
| Add new target | Root + create dir | Follow nginx/ pattern |
| CI release config | .github/workflows/ | release-from-tag + template-release |
| Build definition | */Dockerfile | Multi-stage Alpine |
| Version config | */.env | ALPINE_VERSION, *_VERSION |

## CONVENTIONS

- New top-level directories outside STRUCTURE MUST NOT be added
  (for example, `docs/`, `tests/`).
- Documentation files MUST be limited to `README.md` and `AGENTS.md`
  at any directory level; other documentation filenames and directories
  (for example, `ARTIFACTS.md`, `docs/`) MUST NOT be added.
  Nested directories (for example, `apache-httpd/AGENTS.md`) MAY
  contain README.md or AGENTS.md for target-specific documentation.

- EditorConfig: 4-space indent (2 for .sh/.yaml)
- Makefile targets: nginx, haproxy, apache-httpd, coredns,
  dnsmasq, vector, monit
- Target dir: Must have Dockerfile + .env; download.sh optional
- Upstream source downloads MUST NOT enforce checksum verification/pinning because some upstreams do not publish checksum files. Consumers SHOULD validate sources independently when possible.
- Release workflow MUST use `.github/workflows/release-from-tag.yaml`
  as the only tag-triggered entrypoint and MUST delegate build/release
  logic to `.github/workflows/template-release.yaml`.
- Release workflow MUST run Trivy filesystem scanning and MUST upload
  SARIF results to GitHub Security.
- Release jobs MUST request the minimum required GitHub permission.
  `contents: write` MAY be used only for jobs that publish releases
  or upload release assets.
- `third-party/` is reference-only material for research and
  exploration. It MUST NOT be referenced by this repository's
  implementation and MUST NOT be modified from this repository.
- Shell scripts (`*.sh` and files with a `sh`/`bash` shebang) MUST NOT contain comments except:
  - The shebang line (the first line starting with `#!`).
  - Function documentation comment blocks that are placed immediately above a function definition.
- Function documentation comment blocks MUST:
  - Be contiguous lines starting with `#` (or `##`).
  - Have no blank line between the comment block and the function definition.
  - Describe purpose and expected inputs/outputs/return codes.
  - Use reST Docstring style with field lists, including `:param`, `:return:` (or `:returns:`) and `:rtype:` fields where applicable.
- Inline comments, section header comments, and file header comment blocks MUST NOT be used.
- If explanation is needed, refactor code into a function and document that function instead of adding inline comments.

## Checksum Policy

The repository MUST NOT enforce checksum verification or pinning for
upstream source downloads.

### Rationale

- Many upstream projects do not publish checksum files
- Enforcing checksum verification would prevent building targets with
  legitimate but unsigned releases
- Consumers are responsible for validating sources independently when
  checksum files are available

### Implications

- Build scripts MUST NOT fail if checksum files are not provided by
  upstream
- Release artifacts MUST NOT require checksum verification as part of
  the build process
- This policy ensures consistency across all targets regardless of
  upstream practices

## Third-party Policy

The `third-party/` directory is for research and exploration only.

- The repository MUST NOT reference `third-party/` from build
  scripts, Dockerfiles, workflows, or runtime artifacts.
- The repository MUST NOT introduce changes under `third-party/`
  (including submodule pointer updates).
- The `third-party/` directory MAY be used only for collecting
  information, investigation, and exploration.

## ANTI-PATTERNS (THIS PROJECT)

- No push/PR validation CI (tag-triggered release only)
- No test framework - build infrastructure only

## UNIQUE STYLES

- Tag-triggered release: `nginx-1.28.2.18` → builds + uploads artifact
- Release tags MUST follow `<target>-<official_version>.<x>`.
  - `official_version`: version from target `.env`
    (for example `NGINX_VERSION`, `HTTPD_VERSION`,
    `HAPROXY_VERSION`)
  - `x`: release revision suffix starting at `0` and
    incrementing (`.0`, `.1`, `.2`, ...)
  - examples: `nginx-1.28.2.18`, `httpd-2.4.66.5`, `haproxy-3.2.13.0`
- Unified caller workflow + reusable template pattern
  - caller: `.github/workflows/release-from-tag.yaml`
  - template: `.github/workflows/template-release.yaml`
- Artifacts: local builds output under `out/<target>/...`, while
  CI/release builds output under `<target>/...` for packaging
  compatibility.

## COMMANDS

```bash
make help
make list-targets
make build nginx
make download nginx
make validate-tag nginx nginx-1.28.2.0
```

## NOTES

- Build caching via per-target `<target>/.cache/` directory
- Uses Docker Buildx with BuildKit
