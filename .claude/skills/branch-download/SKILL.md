---
name: branch-download
description: Check download lock interruption and downloader deadline cleanup when changing cmd/download.
---

# Verify download cleanup

Run these checks when changing lock ownership, signal handling, or wget deadlines in `cmd/download`.
The model describes descriptor ownership in `cmd/download/main.go` for both release and branch archives.
Compare its transitions with the current source before relying on a model result.

## Check the lock model

Install repository dependencies with `pnpm install --frozen-lockfile`.
Install Java 25 or a compatible Java runtime.
The `tla-chaos-loop` dependency supplies the TLC executable JAR.
Run TLC from the repository root:

```sh
pnpm run check:branch-lock
```

Check the command's exit status and invariant report.
The retained `references/model/chart.mmd` illustrates one downloader's ownership phases.
Update the chart and `references/model/report.md` when the model changes.

The model follows two downloaders competing for one persistent lock file.
`Acquire` represents successful nonblocking `flock`, and `StartTransport` represents the transport inheriting the lock descriptor through `ExtraFiles`.
`FinishTransport` represents closing the transport's descriptor, while `Close` represents the parent's deferred close after a cache hit, failure, or completed transport.
`Stop` represents parent exit, including SIGKILL, without closing any descriptor still held by its transport.
Cancellation through SIGHUP, SIGINT, or SIGTERM normally waits for transport termination before the parent's deferred close.
The lock stays owned until its final descriptor closes, and lock files must not be removed while any downloader may use them.
The model assumes a local filesystem supporting advisory locks, successful descriptor operations, and no external removal of lock files.
It abstracts bounded polling, temporary files, retries, publication, and transport descendants.
It does not prove termination or establish cancellation deadlines.
`NoAbandonedLock` requires a live parent or transport for every held lock.
`ExclusiveOwnership` requires every working parent and transport to own the same archive's exclusive lock.
A model result establishes only these bounded transitions and requires correspondence checks against the implementation.

## Check real downloader behavior

Run the command tests from the repository root:

```sh
go tool task test
```

The downloader tests invoke the production `main` entrypoint in subprocesses with a controlled curl executable at the external transport boundary.
They cover cache location, release reuse, branch refresh, failed and empty refresh preservation, concurrent processes, and reruns following SIGTERM and SIGKILL.
The real transport's network behavior and wget deadlines require separate evidence when changed.
