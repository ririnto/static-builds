# Formal verification

TLC explores every reachable state within each TLA+ model's bounds and checks the configured invariants and deadlock condition.
Compare the model's source mapping and assumptions with the implementation before relying on its result.

- [Branch downloads](references/branch-download/README.md) covers exclusive archive ownership, descriptor cleanup, and transport ownership after parent exit.

Run these commands from the repository root:

| Command | Result |
| --- | --- |
| `pnpm run check:formal-verification` | Check the models and compare the generated statecharts with the committed copies. |
| `pnpm run generate:formal-verification` | Regenerate statecharts after changing a model or its projection. |

Use the pstack `statechart` skill to read the explored states or trace the shortest counterexample.
Replay a counterexample through the affected command to distinguish a code defect from a mismatch in the model.
Update the source mapping when production boundaries change.
