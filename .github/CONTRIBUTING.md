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
GitLab runs the same complete repository check entrypoint before its manual package trigger.
It also supports manual runs.
Every pull request is checked because deleting a linked file can break documentation without changing Markdown.
The checks use maintained markdownlint-cli2, rumdl, and Lychee releases.
The Markdown job also runs the rule tests, TypeScript validation, and Ultracite checks.

Install the latest Node.js 24 LTS release and the pnpm version declared in `package.json` for Node-side tooling.
Use rumdl 0.3.1 and Lychee 0.24.2, matching the workflow's tool versions.
Install ShellCheck and shfmt for the existing Makefile shell gates.
Run the checks from the repository root:

```sh
pnpm install --frozen-lockfile
pnpm run check
pnpm run fix
```

The root package manifest references Node-side tooling through the pnpm workspace catalog.
`pnpm-workspace.yaml` defines the catalog-backed `katex` development override required by `micromark-extension-math`.
The lockfile records the resolved dependency graph.
`check` runs Markdown lint, rule tests, TypeScript validation, Ultracite, code-fence validation, relative-link validation, ShellCheck, and shell formatting checks.
`fix` applies Markdown lint fixes and the existing Makefile shell formatter.
It also applies Ultracite fixes.
Markdownlint loads the upstream `markdownlint-rule-search-replace` package through `customRules`.
It reports the first Unicode box-drawing character per line, including fenced code, without automatic replacements.
The existing rule tests exercise the actual configuration through Markdownlint's public API.
Their direct `markdown-it` dependency supplies the legacy parser required by the upstream rule.
The local `docs/table-separators` rule requires exactly three hyphens in each table delimiter cell.
It preserves alignment colons and fixes only the delimiter hyphens.
The CLI configuration respects Git ignore files.
Only the supported `customRules` spelling loads packages through the installed CLI2 loader.
The lowercase `customrules` spelling does not register rules through this entrypoint.
The fence check supplements Markdown lint with rumdl's built-in MD046 rule, including missing closing fences.
Lychee checks relative file paths and heading fragments offline, excluding network URLs.
Its `.lychee.toml` configuration explicitly excludes generated paths, including when checking Git worktrees.
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
Ultracite checks the njs runtime example in `nginx/scripting.js` for lint and formatting, while target builds provide its runtime-aware validation.
The njs example uses upstream `njs-types` declarations through its own TypeScript configuration.

The table maps effective enforcement to engineering qualities rather than adding another prose rule set.

| Pillar | Effective enforcement | Applicable scope and rationale |
| --- | --- | --- |
| Performance | Ultracite core `no-await-in-loop`, `unicorn/prefer-set-has`, and `unicorn/prefer-array-some`, plus `oxc/no-accumulating-spread`. | The linted tooling checks sequential work, collection operations, and repeated accumulator copies. |
| Stability | Ultracite core `no-unsafe-finally`, `no-unsafe-optional-chaining`, `no-constant-condition`, and `import/no-cycle`, plus type-aware lint, TypeScript checking, and ShellCheck. | Tooling and maintained shell scripts receive static checks for unsafe control flow, invalid accesses, types, and shell failures. |
| Consistency | Core `prefer-named-capture-group` and `require-unicode-regexp`, Markdownlint `table-column-style`, and configured upstream `search-replace`. | Linted tooling and maintained Markdown retain readable regex captures, compact tables, and portable tree markers. |
| Maintainability | Core `no-unused-vars` and `import/no-duplicates`, spacing rules, rumdl MD046, and offline Lychee checks. | Tooling and documentation avoid unused bindings, duplicate imports, unclosed fences, and broken local references. |

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

Run `python3 proxy/tests/verify-metadata.py` after changing metadata queries or parsing.

- Run `python3 proxy/tests/verify-download.py` for downloader regressions.
- Run `python3 proxy/tests/verify-nginx.py` for nginx verifier regressions.
- Run `python3 proxy/tests/verify-runner.py` for Docker runner regressions.

The downloader and nginx verifier require GNU Coreutils `timeout` or `gtimeout`.
The downloader also requires wget and OpenSSL.
Linux containers running process-lifecycle fixtures need an init process to reap orphaned children.
Use Docker's `--init` option when running these fixtures in a container.
The native healthcheck module is maintained project source.
Changes to its scheduling, probe I/O, or cleanup require a target build and the separate [functional suite](../nginx-upstream-healthcheck/README.md#build-verification).

For build changes, run `make build <target>` for each affected target before creating release tags.
The release workflows run on release tags or reusable workflow calls.
The separate Markdown validation workflow runs on pull requests or manual requests with read-only repository access.
Do not create release tags to validate documentation changes.
