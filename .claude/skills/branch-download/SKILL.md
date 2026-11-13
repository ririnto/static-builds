---
name: branch-download
description: Verify exclusive download ownership and cleanup when changing downloader locking, signals, or transport deadlines.
---

# Verify download cleanup

Use this skill when changing lock ownership, signal handling, or transport deadlines.

1. Compare the [source mapping](references/branch-lock/source-map.md) with the changed implementation.
2. Preserve the canonical model's assumptions and invariants unless the intended contract changes.
3. Run `pnpm check:branch-lock` and inspect its exit status and invariant results.
4. After changing the model or display projection, run `pnpm generate:statechart` and review the [generated statechart](references/branch-lock/statechart.mmd).
5. Run `go tool task test` for production-entrypoint evidence.
6. Obtain separate evidence for real transport deadlines or filesystem behavior when those contracts change.

Do not treat a model result as proof of signal timing, termination, or transport descendants.
Keep both downloaders in the explored model even though the chart follows one downloader.
Do not edit generated arrows or weaken an invariant to make validation pass.
See the [verification scope](references/branch-lock/README.md) for assumptions and omitted behavior.
