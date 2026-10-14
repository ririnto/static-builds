# Contributing

Read [AGENTS.md](../AGENTS.md) and the affected target's README before changing the repository.
Document the problem, affected targets, and acceptance criteria in an issue.
Use the bug report template for failures and the change request template for improvements.
Link related issues instead of expanding a completed issue's scope.

## Documentation

Write one complete sentence per source line.
Separate paragraphs with a blank line.
Keep spaces between words and use lists for separate instructions.
Do not join independent instructions with semicolons or other punctuation.
Apply these rules to maintained Markdown, contributor instructions, AGENTS.md, and SKILL.md files when present.

Preserve technical meaning, permission requirements, code, metadata, exact quotations, licenses, and Markdown table syntax.
Keep code blocks and recorded command output intact when editing prose.
Exclude generated files and vendored material from mechanical prose changes.
Check Markdown rendering, relative links, and instruction loading paths after editing.
Use the repository's `.markdownlint-cli2.jsonc` when running Markdown lint.

### Markdown validation

The [Markdown validation workflow](workflows/markdown-validation.yaml) reports lint, code-fence, and relative-link checks on every pull request.
It also supports manual runs.
Every pull request is checked because deleting a linked file can break documentation without changing Markdown.
The checks use maintained markdownlint-cli2, rumdl, and Lychee releases.

Install Node.js 24 LTS and pnpm 11.28.2 for Node-side tooling.
Use rumdl 0.2.79 and Lychee 0.24.2, matching the workflow's tool versions.
Install ShellCheck and shfmt for the existing Makefile shell gates.
Run the checks from the repository root:

```sh
pnpm install --frozen-lockfile
pnpm run check
pnpm run fix
```

The root package manifest owns Node-side tooling and its lockfile records dependency resolution.
`pnpm-workspace.yaml` selects patched smol-toml and KaTeX versions for development tooling.
Remove these overrides when upstream dependencies select supported patched versions themselves.
`check` runs Markdown lint, code-fence validation, relative-link validation, ShellCheck, and shell formatting checks.
`fix` applies Markdown lint fixes and the existing Makefile shell formatter.
The existing Markdown rules remain enabled through `.markdownlint-cli2.jsonc`.
The CLI configuration respects Git ignore files and excludes the vendored healthcheck module.
The fence check supplements Markdown lint with rumdl's built-in MD046 rule, including missing closing fences.
Lychee checks relative file paths and heading fragments offline, excluding network URLs.
Its `.lychee.toml` configuration explicitly excludes generated and vendored paths, including when checking Git worktrees.
All checks exclude ignored build outputs, source caches, and the vendored healthcheck module.
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

Use `.yaml` when the consuming platform supports that extension.
Check filename discovery rules and repository references before renaming YAML files.
GitHub workflows and Dependabot configuration already use `.yaml` here.

Keep `.gitlab-ci.yml` as GitLab's default pipeline entrypoint.
Renaming it would require changing the project's configured CI/CD path.
See [GitLab's pipeline tutorial](https://docs.gitlab.com/ci/quick_start/).

Keep `templates/static-release.yml` because GitLab CI/CD components require `.yml` component filenames.
See [GitLab's component directory structure](https://docs.gitlab.com/ci/components/#directory-structure).
The parent pipeline and child pipeline generator consume the `static-release` component.

## Commits and pull requests

Create a feature branch from the current `main` branch.
Keep each change focused and reviewable.
Use a commit title followed by a blank line and a body.
Explain the reason, main changes, verification results, and material limitations in the body.
Use one sentence per source line in commit bodies, issues, and pull requests.

Open an ordinary pull request against `main` using the repository's pull request template.
Link the issue and report the commands and results that cover the change.
Mark failed and unrun checks as such.
Review the complete posted change before merging.
Publish through the feature branch and pull request rather than pushing changes directly to `main`.

## Verification

For documentation changes, check Markdown syntax, rendering, relative links, and preservation of code and recorded output.
For build changes, run `make build <target>` for each affected target before creating release tags.
Report unavailable tools and checks that could not run.
The release workflows run on release tags or reusable workflow calls.
The separate Markdown validation workflow runs on pull requests or manual requests with read-only repository access.
Do not create release tags to validate documentation changes.
