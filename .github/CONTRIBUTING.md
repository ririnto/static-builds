# Contributing

Read the affected target's README before changing the repository.
Document the problem, affected targets, and acceptance criteria in an issue.
Use the bug report template for failures and the change request template for improvements.
Link related issues instead of expanding a completed issue's scope.

## Documentation

Check Markdown rendering, relative links, and instruction loading paths after editing.
Use the repository's `.markdownlint-cli2.jsonc` when running Markdown lint.

### Markdown validation

The [GitHub validation workflow](workflows/markdown-validation.yaml) reports documentation and tooling checks on every pull request without building targets or publishing releases.
GitLab runs Go tests, golangci-lint, shell formatting, and Jsonnet checks directly before its manual package trigger.
Its Go jobs use `golang:alpine` with Go's default `GOTOOLCHAIN=auto` behavior honoring the `go.mod` directive.
The package component invokes the same direct Docker Buildx Taskfile entrypoint as local builds.
Node and pnpm validation remains in the GitHub workflow and local contributor hooks.
It also supports manual runs.
Every pull request is checked because deleting a linked file can break documentation without changing Markdown.
The Markdown job also runs rule tests, TypeScript validation, Ultracite, Go tests, golangci-lint, shell formatting, and Jsonnet checks.

Install the Node.js version supported by `package.json` and the pnpm version declared there for Node-side tooling.
Install the Go version declared in `go.mod` for repository tasks and Go-managed contributor tools.
Run Go tests, `gofmt -w cmd internal`, and `go mod tidy` directly.
Invoke managed tools through tool-named Task tasks with arguments after `--`.
On macOS, Go-managed tools may compile dependencies that require system SDK headers through CGO.
Export `SDKROOT=$(xcrun --show-sdk-path)` when the compiler or linker cannot locate the system SDK.
The following tools are managed through `go.mod`:

- Task
- shfmt
- Jsonnet
- jsonnetfmt
- golangci-lint
- gorelease
- buildctl

The Taskfile selects every tracked shell script with `git ls-files -z -- '**/*.sh'` for shfmt.
Use `go tool task jsonnetfmt -- -i` to format the package pipeline and `go tool task jsonnetfmt -- --test` to check it.
Use `go tool task release -- <gorelease flags>` for Go module compatibility checks.
`go tool task buildctl -- --help` exposes the BuildKit client without changing the Docker Buildx build driver.
Target builds require Docker Buildx and a running Linux Docker daemon.
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
The existing rule tests exercise the actual configuration through Markdownlint's public API.
The direct `markdown-it` dependency supplies the parser used by the token-based rules.
The CLI configuration respects Git ignore files.
Only the supported `customRules` spelling loads packages through the installed CLI2 loader.
The lowercase `customrules` spelling does not register rules through this entrypoint.
All checks exclude ignored build outputs and source caches.
These checks run without building targets or invoking release jobs.
Node dependencies are development-only and are excluded from the Docker build context.
Dockerfiles copy explicit target inputs and release packaging selects only metadata-listed target outputs.

### Contributor toolchain standards

The JavaScript and TypeScript lint, formatter, and compiler configurations form one contributor-tooling group.
`oxlint.config.ts` extends `ultracite/oxlint/core` and preserves its environment, ignore patterns, rules, and test overrides.
The preset supplies native ESLint, TypeScript, Unicorn, Oxc, import, JSDoc, Node, and Promise plugins.
The same configuration checks function-body spacing and adjacent test registrations without replacing the base rules.
The test spacing rule recognizes registrations imported from Vitest, Jest globals, and Bun, but does not recognize `node:test` registrations.
`check:ultracite` runs Ultracite with type-aware lint across its discovered files.
`check:typescript` checks those TypeScript files through `tsconfig.json` and the Node LTS compiler preset.
Ultracite checks the njs runtime example in `nginx/scripting.js` for lint and formatting.
Both nginx targets validate the scripting configuration, while only the custom healthcheck target runs runtime probes.
The njs example uses upstream `njs-types` declarations through its own TypeScript configuration.

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
Ultracite remains the only direct caller of Oxlint and Oxfmt.

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
Link the issue and report the commands and results that cover the change.
Mark failed and unrun checks as such.
Review the complete posted change and confirm its current head before normal server-side integration.
Publish through the feature branch and pull request rather than pushing changes directly to `main`.

## Verification

Explain in the issue when existing guards remain because inversion requires artificial flags or deeper nesting.

Run `go test ./...` for the retained command tests.
The contributor check entrypoint includes these tests, `go tool task golangci-lint -- run ./...` checks, and `go tool task shfmt -- -d -i 2 -ci` formatting through Task.
Target builds run their Dockerfile verify stages without per-target verify scripts or external runtime harnesses.
The custom healthcheck module remains maintained production source.

For build changes, run `go tool task build -- <target>` for each affected target before creating release tags.
The release workflows run on release tags or reusable workflow calls.
The separate Markdown validation workflow runs on pull requests or manual requests with read-only repository access.
Do not create release tags to validate documentation changes.
