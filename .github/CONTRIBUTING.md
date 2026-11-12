# Contributing

Read the affected target's README before changing the repository.
Document the problem, affected targets, and acceptance criteria in the task context or pull request.
Create an issue when it helps coordination or tracking, using the bug report template for failures or the change request template for improvements.
When related issues exist, link them instead of expanding a completed issue's scope.

## Documentation

Check Markdown rendering, relative links, and instruction loading paths after editing.
Use the repository's `.markdownlint-cli2.jsonc` when running Markdown lint.

### Contributor validation

The [GitHub validation workflow](workflows/markdown-validation.yaml) runs Go-managed tooling checks on every pull request without building targets or publishing releases.
The separate [build validation workflow](workflows/validate-builds.yaml) runs nonpublishing GoReleaser snapshots on build-relevant pull requests, using native amd64 for every metadata target and native arm64 for all except Vector's fixed x86_64 archive.
Each matrix job has an isolated checkout and distribution directory, checks package SHA-256 and metadata-selected files, bytes, and modes, and uploads an architecture-named package without sharing release caches or credentials.
GitLab invokes Go tests, golangci-lint, shell formatting, and Jsonnet formatting through `go tool task` before generating its package pipeline.
Branch child pipelines start automatically and offer an optional manual build for each target; tag releases run automatically.
Local hooks and the GitHub tooling workflow check main, feature, and release-tag Jsonnet rendering.
Its Go jobs use `golang:alpine` with Go's default `GOTOOLCHAIN=auto` behavior honoring the `go.mod` directive.
The GitLab package component invokes GoReleaser, whose before hook calls the shared rootless Task driver without Docker-in-Docker.
GitLab reuses signed APK downloads in its declared `APK_CACHE_DIR`, isolated by runner, while refreshing repository indexes on every job.
Node, pnpm, and Java checks run through local contributor hooks.
It also supports manual runs.
The GitHub job runs Go tests, golangci-lint, shell formatting, and Jsonnet checks.

Install the Node.js version supported by `package.json` and the pnpm version declared there for Node-side tooling.
Install the Go version declared in `go.mod` for repository tasks and Go-managed contributor tools.
Install Java 25 or a compatible Java runtime for `pnpm run check:branch-lock`, which also runs through `pnpm check` and Git hooks.
Use `go tool task test`, `go tool task fmt`, and `go tool task tidy`.
Invoke managed tools through tool-named Task tasks with arguments after `--`.
On macOS, Go-managed tools may compile dependencies that require system SDK headers through CGO.
Export `SDKROOT=$(xcrun --show-sdk-path)` when the compiler or linker cannot locate the system SDK.
The following tools are managed through `go.mod`:

- Task
- shfmt
- Jsonnet
- jsonnetfmt
- golangci-lint
- GoReleaser
- buildctl

The Taskfile selects tracked shell scripts and contributor hooks with NUL-delimited Git output and invokes shfmt through typed-list Task iteration.
The shfmt task owns the shared two-space indentation and case indentation flags, while callers select check or write mode.
Tool tasks quote each CLI_ARGS_LIST element separately so argument boundaries remain explicit.
Use `go tool task jsonnetfmt -- -i` to format the package pipeline and `go tool task jsonnetfmt -- --test` to check it.
`go tool task buildctl -- --help` exposes the same Go-managed BuildKit client used by the build task.
Local container builds use compose.yaml with a golang-based non-root image and the upstream daemonless BuildKit script.
Run `mkdir -p .cache dist`, `docker compose build build`, and `TARGET=<target> docker compose run --rm build` from the repository root.
The dedicated container relaxes seccomp and masked system paths as described in the [official rootless Docker guidance](https://github.com/moby/buildkit/blob/v0.34.2/docs/rootless.md#docker).
It keeps BuildKit's process sandbox, uses the native snapshotter without a FUSE device, and never mounts the Docker socket.
Direct local builds require a non-root user on a configured Linux host with BuildKit, RootlessKit, runc, subordinate UID/GID mappings, and unprivileged user namespaces.
Go manages the BuildKit client and daemon through `go.mod`; GitHub installs distribution-packaged `rootlesskit`, `uidmap`, and `runc` without changing host security controls.
Use Compose on macOS and Windows.
GitLab builds use a non-root user with subordinate UID and GID ranges of `100000:65536`.
The runner must permit unprivileged user namespaces and rootless mounts.
The job uses `--oci-worker-no-process-sandbox` because the container masks `/proc`, so build processes share the daemon process namespace.
This runner-specific compromise differs from Compose, which keeps the process sandbox enabled.
Self-managed GitLab runners should follow [GitLab's tailored seccomp guidance](https://docs.gitlab.com/ci/docker/using_buildkit/#rootless-build-fails-with-permission-errors) rather than changing security settings from job scripts.
See the root README for [build configuration and invocation examples](../README.md#build-configuration).
Run the checks from the repository root:

```sh
pnpm install --frozen-lockfile
pnpm run check
pnpm run fix
```

The root package manifest references Node-side tooling through the pnpm workspace catalog.
`pnpm-workspace.yaml` defines the catalog-backed `katex` development override required by `micromark-extension-math`.
The lockfile records the resolved dependency graph.
`check` runs markdownlint-cli2, rule tests, TypeScript validation, Ultracite, download and manifest Go tests, golangci-lint, shell formatting, Jsonnet evaluation, and Jsonnet formatting checks.
Markdown validation uses pnpm and markdownlint-cli2, including the `code-block-style` rule for fenced code blocks.
`fix` applies the following changes:

- Markdown lint fixes
- Shell formatting through the Go-managed Task entrypoint
- Jsonnet formatting through the Go-managed Task entrypoint
- Ultracite fixes

Markdownlint loads the local Markdownlint rules through `customRules`, applying the shared Markdownlint baseline with the repository's stricter exceptions.
The `docs/no-box-drawing` rule reports the first Unicode box-drawing character per line, including fenced code, without automatic replacements.
The `docs/no-em-dash` rule rejects em dashes in text tokens while skipping code spans, code blocks, HTML tokens, and image alt text.
Static Builds retains the default `single-title` front matter check, which is stricter than the shared baseline.
Static Builds keeps the stricter table HTML exception by allowing only `br` inside table markup while the shared baseline also allows `code`.
The local `docs/table-separators` rule requires exactly three hyphens in each table delimiter cell.
It preserves alignment colons and fixes only the delimiter hyphens.
The Markdownlint rule tests exercise owned behavior through Markdownlint's public API.
`check:test` runs the Go command tests followed by Node's native `**/*.test.ts` discovery.
Focused Oxlint tests use its official RuleTester API without retesting the upstream preset or formatter.
The direct `markdown-it` dependency supplies the parser used by the token-based rules.
The CLI configuration respects Git ignore files.
Only the supported `customRules` spelling loads packages through the installed CLI2 loader.
The lowercase `customrules` spelling does not register rules through this entrypoint.
All checks exclude ignored build outputs and source caches.
These checks run without building targets or invoking release jobs.
Check target mapping with `go tool task manifest -- get-target-from-tag <tag>` without creating a tag.
Render a tag child pipeline with the Jsonnet `--tla-str release_tag=<tag>` argument and one explicit target.
Use `TARGET=<target> TAG=<target>-local go tool task goreleaser -- release --clean --snapshot --verbose --timeout=6h --skip=publish,validate` for nonpublishing build, archive, and checksum checks.
Run one target at a time because `--clean` removes the previous GoReleaser distribution output.
The Compose service and `TARGET=<target> TAG=<target>-local pnpm run build` use this same snapshot flow.
Compose defaults `TAG` to `local`, but never supplies a default target.
Live release publication is separate from configuration validation.
Node dependencies are development-only and are excluded from the Docker build context.
Dockerfiles copy explicit target inputs and release packaging selects only metadata-listed target outputs.

### Contributor toolchain standards

The JavaScript and TypeScript lint, formatter, and compiler configurations form one contributor-tooling group.
`oxlint.config.ts` extends `ultracite/oxlint/core` and preserves its environment, ignore patterns, rules, and test overrides.
The preset supplies native ESLint, TypeScript, Unicorn, Oxc, import, JSDoc, Node, and Promise plugins.
The same configuration loads repository-owned plugins from `plugins/` to check function-body spacing and adjacent test registrations without replacing the base rules.
The test spacing rule recognizes registrations imported from Vitest, Jest globals, and Bun, but does not recognize `node:test` registrations.
`check:ultracite` runs Ultracite with type-aware lint across its discovered files.
`check:typescript` checks those TypeScript files through `tsconfig.json` and the Node LTS compiler preset.
Ultracite checks the optional njs examples in each nginx target's `examples/njs/scripting.js` for lint and formatting.
The njs example is not copied into runtime images or release archives.
The custom healthcheck target checks its health transitions through a local proxy.
The njs example uses upstream `njs-types` declarations through its own TypeScript configuration.
That separate configuration preserves the njs runtime globals, which differ from Node's Buffer and module types.

The table maps effective enforcement to engineering qualities rather than adding another prose rule set.

| Pillar | Effective enforcement | Applicable scope and rationale |
| --- | --- | --- |
| Performance | Ultracite core `no-await-in-loop`, `unicorn/prefer-set-has`, and `unicorn/prefer-array-some`, plus `oxc/no-accumulating-spread`. | The linted tooling checks sequential work, collection operations, and repeated accumulator copies. |
| Stability | Ultracite core `no-unsafe-finally`, `no-unsafe-optional-chaining`, `no-constant-condition`, and `import/no-cycle`, plus type-aware lint, TypeScript checking. | Tooling receives static checks for unsafe control flow, invalid accesses, and types. |
| Maintainability | Core `no-unused-vars` and `import/no-duplicates`, spacing rules, and Markdownlint MD046. | Tooling avoids unused bindings and duplicate imports while documentation uses consistent code blocks. |
| Consistency | Core `prefer-named-capture-group` and `require-unicode-regexp`, Markdownlint `table-column-style`, and the local `docs/no-box-drawing` rule. | Linted tooling and maintained Markdown retain readable regex captures, compact tables, and portable tree markers. |

These static checks do not establish measured performance improvements or target runtime acceptance.

### Formatter safety

`oxfmt.config.ts` spreads `ultracite/oxfmt`, preserves its required ignores, excludes Markdown, and sets `trailingComma` to `none`.
The inherited formatter can sort bound imports, rewrite recognized embedded or Tailwind literals, and normalize comments or raw literal spellings.
`fix:ultracite` runs Ultracite's automatic fixes.
Automatic JavaScript and TypeScript fixes can remove module side effects and require individual review.

Formatter acceptance requires unchanged parsed values, meaningful member order, literal contents, and comments, followed by an idempotent second pass.
Conditional package exports and YAML command scalars require fresh safety verification when their contents change.
Ultracite remains the only CLI caller of Oxlint and Oxfmt.

Run `pnpm audit --prod --audit-level low` for production dependency scope.
Report `pnpm audit --audit-level low` findings separately, including upstream findings without available patches.
Do not add audit exclusions or change lint rules to hide findings.

### Git hooks

Dependency installation runs Husky's `prepare` script in this checkout.
Husky installs the tracked `.husky/pre-commit` and `.husky/pre-push` hooks through `.husky/_`.
The hooks invoke `pnpm run pre-commit` and `pnpm run pre-push` respectively.
Both scripts run the complete contributor checks and block Git operations when a check fails.
Install the documented native tools before committing or pushing.
Preserve existing hook owners through reviewed composition when adopting these hooks in an existing checkout.

## Commits and pull requests

Create a feature branch from the current `main` branch.
Keep each change focused and reviewable.
Use a commit title followed by a blank line and a body.
Explain the reason, main changes, verification results, and material limitations in the body.
Use one sentence per source line in commit bodies, issues, and pull requests.

Open an ordinary pull request against `main` using the repository's pull request template.
Link a related issue when one exists and report the commands and results that cover the change.
Mark failed and unrun checks as such.
Review the complete posted change and confirm its current head before normal server-side integration.
Publish through the feature branch and pull request rather than pushing changes directly to `main`.

## Verification

Explain in the task context, pull request, or related issue when existing guards remain because inversion requires artificial flags or deeper nesting.

Run `go tool task test` for the retained command tests.
The contributor check entrypoint includes these tests, `go tool task golangci-lint -- run ./...` checks, and `go tool task shfmt -- -d` formatting through Task.
Target builds run their Dockerfile verify stages without per-target verify scripts or external runtime harnesses.
The custom healthcheck module remains maintained production source.

For build changes, run the nonpublishing GoReleaser snapshot command for each affected target and inspect its exported binaries, archive members, and checksum before creating release tags.
The release workflows run on release tags or reusable workflow calls.
The separate Markdown validation workflow runs on pull requests or manual requests with read-only repository access.
Do not create release tags to validate documentation changes.
