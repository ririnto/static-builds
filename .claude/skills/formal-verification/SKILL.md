---
name: formal-verification
description: Run and maintain the repository's formal checks when changing modeled behavior, including download lock ownership and cleanup.
---

# Formal verification

- [Branch downloads](references/branch-download/README.md) covers archive lock ownership and descriptor cleanup.

Compare the model's source mapping and assumptions with the implementation you changed.
Run `pnpm run check:formal-verification` from the repository root to check the models and their statecharts.
After changing a model or its projection, run `pnpm run generate:formal-verification` and review the resulting statechart.
Use the pstack `statechart` skill to inspect explored states or trace a counterexample.
Replay counterexamples through the affected production command and update the source mapping when production boundaries change.
