---
name: branch-download
description: Check branch-download lock interruption and downloader deadline cleanup when changing cmd/download.
---

# Verify branch download cleanup

Use this skill when changing branch lock ownership, signal handling, or wget deadlines in `cmd/download`.
The model describes lock ownership and context cancellation in `cmd/download/main.go` at commit `97a39c220afb3338ef63583e0a7c31a22ad20fa3`.

## Check the lock model

Install repository dependencies with `pnpm install --frozen-lockfile` and use the installed `/statechart` skill to run `model/BranchLock.tla` with `model/BranchLock.cfg` and `model/view.mjs`.
Set `TLA2TOOLS` to the repository's `node_modules/tla-chaos-loop/tools/tla2tools.jar` so the tool does not bootstrap another checker.
The catalog resolves `tla-chaos-loop` 1.1.0, whose bundled TLC reports build `2026.07.18.145032`.
The recorded run used Node 26.13.0 and Java 25.0.5.
From the repository root, set `STATECHART` to the installed `/statechart` executable and run:

```sh
TLA2TOOLS="$PWD/node_modules/tla-chaos-loop/tools/tla2tools.jar" "$STATECHART" \
  "$PWD/.claude/skills/branch-download/model/BranchLock.tla" \
  --config "$PWD/.claude/skills/branch-download/model/BranchLock.cfg" \
  --view "$PWD/.claude/skills/branch-download/model/view.mjs" \
  --out "$(mktemp -d)/out"
```

Read every result with `/statechart` and compare the generated chart and report with the retained `model/chart.mmd` and `model/report.txt`.
The recorded check passes `NoAbandonedLock` and deadlock checking over 10 states.
The chart hides cancellation steps that do not change its displayed phase.

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
go tool task go -- test ./...
```

These tests cover the retained source-download and manifest interfaces without a separate process-lifecycle harness.
They do not prove signal-race behavior or establish model correspondence by themselves.
Use targeted production-interface evidence when changing lock ownership or cancellation, and report untested runtime behavior explicitly.
