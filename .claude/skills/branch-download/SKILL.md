---
name: branch-download
description: Check branch-download lock interruption and downloader deadline cleanup when changing cmd/download.
---

# Verify branch download cleanup

Use this skill when changing branch lock ownership, signal handling, or wget deadlines in `cmd/download`.
The retained model describes the historical shell implementation rather than the current Go command.
The Go command records lock cleanup ownership immediately after successful `os.Mkdir` and handles signals through context cancellation.
Do not present the historical model result as verification of the Go implementation.
Update the model and replay its transitions against the Go command before making current lock-model correspondence claims.

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
Exit 1 is expected for the retained historical model only when the report identifies `NoAbandonedLock` as broken.

For the retained historical model, `NoAbandonedLock` fails.
If source behavior intentionally changes, update the model to match each affected command, rerun it, and record the new result instead of preserving this historical failure.
The former shell branch downloader could leave its lock after interruption, causing a later download to wait for the lock and eventually fail.
The shortest modeled path creates the lock, receives a signal before recording ownership, exits its signal handler, and skips lock removal because no ownership was recorded.
A controlled replay of the former script reproduced this sequence on the tested macOS shell.
Do not interpret the expected failure as a passing invariant or change signal semantics merely to silence it.

The model follows one branch download with an initially absent lock.
`Mkdir` represents historical shell lock creation, and `RecordLock` represents its separate ownership assignment.
`Signal` represents delivery at a historical shell command boundary, `SignalExit` represents its signal trap, and `CheckLock` and `RemoveLock` represent its exit trap.
`CompleteWork` abstracts the intervening download and its temporary-file cleanup without claiming their correctness.
The model assumes an empty lock directory, successful cleanup commands, and no repeated signals during the traps.
It excludes competing downloaders, lock wait and timeout, failed `mkdir` or `rmdir`, temporary-file and child ownership, retry, publication, SIGKILL, crashes, and external filesystem changes.
A model result establishes only the stated bounded model and requires replay against the corresponding implementation.

## Check real downloader behavior

Run the surviving command tests from the repository root:

```sh
go tool task test-go
```

These tests cover the retained source-download and manifest interfaces without a separate process-lifecycle harness.
They do not establish the historical model's correspondence with the current Go downloader or prove signal-race behavior.
Use targeted production-interface evidence when changing lock ownership or cancellation, and report untested runtime behavior explicitly.
