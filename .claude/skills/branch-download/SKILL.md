---
name: branch-download
description: Check branch-download lock interruption and downloader deadline cleanup when changing cmd/download.
---

# Verify branch download cleanup

Run these checks when changing branch lock ownership, signal handling, or wget deadlines in `cmd/download`.
The model describes lock ownership and context cancellation in `cmd/download/main.go`.
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
The retained `references/model/chart.mmd` shows the model's phase transitions in Mermaid.
The chart omits cancellation steps that leave the displayed phase unchanged.
Update the chart and `references/model/report.md` when the model changes.
The recorded check passes `NoAbandonedLock` and deadlock checking over 10 states.

The model follows one branch download with an initially absent lock.
`Mkdir` represents successful `os.Mkdir`, and `RegisterCleanup` represents the subsequent deferred removal registration.
`Signal` represents context cancellation from SIGHUP, SIGINT, or SIGTERM without interrupting ordinary Go execution.
`Return` abstracts all paths returning after registration, and `RemoveLock` represents the deferred `os.Remove`.
Cancellation may occur before lock creation, between creation and defer registration, during work, or during cleanup.
The model assumes successful lock creation and removal and eventual return from work.
It excludes competing downloaders, lock wait and timeout, filesystem errors, temporary files, child processes, retries, publication, SIGKILL, crashes, and external filesystem changes.
It does not prove termination or verify that cancellation reaches every blocking operation.
The invariant remains `NoAbandonedLock == (phase = "done" /\ interrupted) => ~lock`.
A model result establishes only the stated bounded model and requires correspondence checks against the implementation.

## Check real downloader behavior

Run the surviving command tests from the repository root:

```sh
go tool task test
```

These tests cover the retained source-download and manifest interfaces without a separate process-lifecycle harness.
They do not prove signal-race behavior or establish model correspondence by themselves.
Use targeted production-interface evidence when changing lock ownership or cancellation, and report untested runtime behavior explicitly.
