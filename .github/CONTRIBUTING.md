# Contributing

Read the affected target's README before changing the repository.
Document the problem, affected targets, and acceptance criteria in an issue.
Use the bug report template for failures and the change request template for improvements.
Link related issues instead of expanding a completed issue's scope.

## Documentation

Check Markdown rendering, relative links, and instruction loading paths after editing.
Use the repository's `.markdownlint-cli2.jsonc` when running Markdown lint.

### Markdown validation

The [Markdown validation workflow](workflows/markdown-validation.yaml) reports Markdown, code-fence, and relative-link checks on every pull request.
It also supports manual runs.
Every pull request is checked because deleting a linked file can break documentation without changing Markdown.
The checks use maintained markdownlint-cli2, rumdl, and Lychee releases.
The Markdown job also runs the custom rule tests, TypeScript validation, and Ultracite checks.

Install Node.js 24.22.0 and pnpm 12.14.0 for Node-side tooling.
Use rumdl 0.3.0 and Lychee 0.24.2, matching the workflow's tool versions.
Install ShellCheck and shfmt for the existing Makefile shell gates.
Run the checks from the repository root:

```sh
pnpm install --frozen-lockfile
pnpm run check
pnpm run fix
```

The root package manifest references Node-side tooling through the pnpm workspace catalog.
`pnpm-workspace.yaml` defines catalog-backed `markdown-it`, `smol-toml`, and `katex` development overrides.
The lockfile records the resolved dependency graph.
Follow [AGENTS.md](../AGENTS.md) for catalog, version, and override conventions.
`check` runs Markdown lint, custom rule tests, TypeScript validation, Ultracite, code-fence validation, relative-link validation, ShellCheck, and shell formatting checks.
`fix` applies Markdown lint fixes, Ultracite formatting, and the existing Makefile shell formatter.
The rule in `rules/no-box-drawing.ts` rejects Unicode box drawing characters in Markdown text and fenced code.
It reads each full match through the named `boxDrawing` group and reports only the first match on each line.
The CLI configuration respects Git ignore files.
The root `oxfmt.config.ts` ignores Markdown files and sets `trailingComma` to `none`.
Follow the Markdownlint baseline and Ultracite caller requirements in [AGENTS.md](../AGENTS.md).
The fence check supplements Markdown lint with rumdl's built-in MD046 rule, including missing closing fences.
Lychee checks relative file paths and heading fragments offline, excluding network URLs.
Its `.lychee.toml` configuration explicitly excludes generated paths, including when checking Git worktrees.
All checks exclude ignored build outputs and source caches.
These checks run without building targets or invoking release jobs.
Node dependencies are development-only and are excluded from the Docker build context.
Dockerfiles copy explicit target inputs and release packaging selects only metadata-listed target outputs.

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

## YAML filenames

Follow the YAML filename conventions in [AGENTS.md](../AGENTS.md).

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

Follow behavior and control-flow conventions in [AGENTS.md](../AGENTS.md).
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
