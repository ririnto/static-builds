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
Use the repository's `.markdownlint.jsonc` when running Markdown lint.

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
The current GitHub workflows run on release tags or reusable workflow calls, without a push or pull request validation workflow.
Do not create release tags to validate documentation changes.
