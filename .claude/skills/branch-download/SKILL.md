---
name: branch-download
description: Check branch-download lock interruption and downloader deadline cleanup when changing download.sh.
---

# Verify branch download cleanup

Use this skill when changing branch lock ownership, signal handling, or wget deadlines in `download.sh`.
The source mapping was checked against commit `5f5c3fdb5ae8b2b50dc9d4f6f05da7b57982aafb`.
Move that sync commit forward only after checking the mapping and rerunning both checks against the new source.

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
Exit 1 is expected at the sync commit only when the report identifies `NoAbandonedLock` as broken.

At the sync commit, `NoAbandonedLock` fails.
If source behavior intentionally changes, update the model to match each affected command, rerun it, and record the new result instead of preserving this historical failure.
A branch download can leave its lock after interruption, causing a later download to wait for the lock and eventually fail.
The shortest modeled path creates the lock, receives a signal before recording ownership, exits its signal handler, and skips lock removal because no ownership was recorded.
A controlled replay of the unchanged script reproduced this sequence on the tested macOS shell.
Do not interpret the expected failure as a passing invariant or change signal semantics merely to silence it.

The model follows one branch download with an initially absent lock.
`Mkdir` represents the successful `mkdir` at `download.sh:49`, and `RecordLock` represents the separate assignment at `download.sh:60`.
`Signal` represents delivery at a modeled command boundary, `SignalExit` represents the signal trap at `download.sh:44`, and `CheckLock` and `RemoveLock` represent the exit trap at `download.sh:43`.
`CompleteWork` abstracts the intervening download and its temporary-file cleanup without claiming their correctness.
The model assumes an empty lock directory, successful cleanup commands, and no repeated signals during the traps.
It excludes competing downloaders, lock wait and timeout, failed `mkdir` or `rmdir`, temporary-file and child ownership, retry, publication, SIGKILL, crashes, and external filesystem changes.
A model result establishes only the stated bounded model and requires replay against the real script.

## Check real downloader behavior

Run this existing production-interface verifier from the repository root:

```sh
python3 proxy/tests/verify-download.py
```

It needs Python 3, OpenSSL, wget, and GNU Coreutils or BusyBox `timeout`, or GNU `gtimeout`.
All eight scenarios must pass.
It exercises wget success, deadlines, resistant children, cancellation after wget starts, HTTP failure, missing deadline commands, and fixture cleanup using local HTTPS fixtures.
It uses one attempt and forces the wget path.
It does not cover curl, retry and backoff, concurrent branch writers, destination preservation, archive integrity, or the lock-recording interruption window.

A scratch copy with wget's `-k 1` escalation removed failed the resistant-child scenario by exceeding the six-second driver deadline.
The restored source passed all eight scenarios.
This negative control demonstrates sensitivity to that deadline regression only, independently of the failing lock property.
A separate 100-trial child-signal experiment did not reproduce a child leak, but did not force the spawn-to-PID-assignment window.
That child path remains inconclusive and is outside the lock model.
