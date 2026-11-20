# Branch download verification

Use this check when changing download locking, signal handling, or transport deadlines.
A parent or transport working on an archive must hold its exclusive lock.
If another downloader can take the lock early, it can remove an active partial archive.

1. Compare the [source mapping](source-map.md) with the changed code.
2. Run `pnpm run check:formal-verification` and inspect the TLC verdict and invariant results.
3. Use the pstack `statechart` skill to read the explored states or the shortest counterexample.
4. After changing the model or projection, run `pnpm run generate:formal-verification` and review the [generated statechart](statechart.mmd).
5. Run `go tool task test` for evidence from the downloader command.
6. Update the source mapping if the production boundaries change.

When TLC finds a violation, trace the reported actions to the mapped production operations.
Reproduce the sequence through the downloader command to distinguish a code defect from a mismatch in the model.
Preserve the invariants unless the intended contract changes.

The check runs the canonical [TLA+ model](../../assets/branch-download/BranchLock.tla) and [TLC configuration](../../assets/branch-download/BranchLock.cfg).
The model covers two downloaders competing for one persistent archive lock.
It assumes a local filesystem with advisory locks, successful descriptor operations, and no external lock-file removal.

The model omits:

- Bounded polling and retries.
- Temporary-file publication and archive integrity.
- Transport descendants and real network behavior.
- Timing bounds and termination guarantees.

The generated chart follows downloader 1 and preserves its inherited transport ownership after parent exit.
Downloader 2 remains part of every invariant check.
The projection hides only transitions that stay within the same displayed state, including downloader 2's activity.
Chart arrows come from TLC's explored transitions rather than a separately maintained transition list.

## Verification results

The canonical model passed both configured invariants and deadlock checking.
TLC generated 50 states, found 20 distinct states, and completed at depth 5 with no queued states.
The chart projects 49 explored transitions into six display states and ten arrows.
It hides 25 transitions within a display state.

The process tests provide separate evidence for observable downloader behavior.
Use a dedicated reproduction for real transport deadlines, filesystem changes, or process-tree termination.
