# Branch-lock verification scope

The canonical model covers two downloaders competing for one persistent archive lock.
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

## Latest verification

On 2026-11-13, the canonical model passed both configured invariants and deadlock checking.
TLC generated 50 states, found 20 distinct states, and completed at depth 5 with no queued states.
The chart projects 49 explored transitions into six display states and ten arrows.
It hides 25 transitions within a display state.

The [source mapping](source-map.md) connects model actions to production boundaries.
Production-entrypoint tests provide separate evidence for downloader behavior.
Do not infer process-tree termination or all signal interleavings from either check.
