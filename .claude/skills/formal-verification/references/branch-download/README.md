# Branch download verification

Use this check when changing download locking, signal handling, or transport deadlines.
A parent or transport working on an archive must hold its exclusive lock.
If another downloader can take the lock early, it can remove an active partial archive.

## Model scope

The [TLA+ model](../../assets/branch-download/BranchLock.tla) covers two downloaders competing for one persistent archive lock.
The [TLC configuration](../../assets/branch-download/BranchLock.cfg) checks `NoAbandonedLock`, `ExclusiveOwnership`, and deadlocks.
The [source mapping](source-map.md) describes both invariants and maps the model actions to the downloader.
The model assumes a local filesystem with advisory locks, successful descriptor operations, and no external lock-file removal.

The model omits:

- Bounded polling and retries.
- Temporary-file publication and archive integrity.
- Transport descendants and real network behavior.
- Timing bounds and termination guarantees.

The [generated statechart](../../assets/branch-download/statechart.mmd) follows downloader 1 and shows that a transport can retain the lock after parent exit.
Downloader 2 remains part of every invariant check.
The projection hides only transitions that stay within the same displayed state, including downloader 2's activity.
The [generation script](../../scripts/branch-download/statechart.mjs) projects TLC's explored transitions into the Mermaid chart.

## Verification results

The canonical model passed both configured invariants and deadlock checking.
TLC generated 50 states, found 20 distinct states, and completed at depth 5 with no queued states.
The chart projects 49 explored transitions into six display states and ten arrows.
It hides 25 transitions within a display state.

Run `go tool task test` to exercise observable downloader behavior through the downloader command.
Use a dedicated reproduction for real transport deadlines, filesystem changes, or process-tree termination.
