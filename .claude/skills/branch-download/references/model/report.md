# BranchLock

`pnpm run check:branch-lock` passed on 2026-11-11 with exit status 0.
TLC generated 50 states and found 20 distinct states, with no states left on the queue.
The complete search depth was 5.

## Rules that hold

`NoAbandonedLock`, `ExclusiveOwnership`, and deadlock checking passed for the bounded two-downloader model.
The model includes inherited transport descriptor ownership after parent exit, but does not prove termination, transport deadlines, or filesystem behavior.
Production-entrypoint subprocess tests separately cover release reuse, branch refresh, failed downloads, concurrent calls, and SIGTERM and SIGKILL reruns.
