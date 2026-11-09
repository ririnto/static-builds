---
name: branch-download
description: Verify downloader deadline and cleanup behavior when changing download.sh or its process lifecycle.
---

# Verify downloader deadlines and cleanup

Use this skill when changing `download.sh`, especially its wget deadline, temporary files, branch lock, or signal cleanup.
The source mapping was last checked against commit `5f5c3fdb5ae8b2b50dc9d4f6f05da7b57982aafb`.
Move that sync commit forward only after checking the mapping against the new source and rerunning the verification.

## Run

From the repository root, run the existing production-interface verifier:

```sh
python3 proxy/tests/verify-download.py
```

It needs Python 3, OpenSSL, wget, and GNU Coreutils or BusyBox `timeout`, or GNU `gtimeout`.
It copies the current production scripts into isolated temporary fixtures and uses a local HTTPS server.
All eight scenarios must pass.
A failure identifies a scenario, and its process and fixture cleanup runs even after a driver timeout.

The verifier checks successful branch downloads, slow and signal-resistant wget deadlines, `gtimeout`, interruption after wget starts, HTTP failure, a missing deadline command, and fixture cleanup.
It uses one attempt and forces the wget path.
It does not establish curl behavior, retry and backoff, concurrent branch writers, destination preservation, archive integrity, or signals between lock acquisition and ownership recording.

## Source mapping and limits

`download.sh` creates the branch lock with `mkdir` and later records its path for the exit trap.
The signal trap terminates and waits for the recorded child before the exit trap removes recorded temporary and lock paths.
The wget path uses `timeout -k 1`, and a successful nonempty temporary file is moved into place.
The verifier exercises those production interfaces but does not explore every signal delivery point.
A signal between successful lock creation and ownership recording is an unconfirmed model hypothesis, not a demonstrated source defect.

The branch-download TLA+ model is an unchecked draft, not a formal verification result.
The available statechart frontend failed on the SANY XML emitted by `tlaplus-mcp` 1.0.0 TLC 2.0.
Do not claim model properties or generate a substitute chart from this draft.
When a compatible checker is available through an approved repository dependency channel, use `/statechart` to inspect every result, validate each model action against its separate source command, and replay any counterexample against the real script.
The draft omits retries, timeouts, failures, parent signal forwarding, and external actors, and assumes shell signals can arrive between its modeled commands.

## Negative control

The existing verifier detected a scratch mutation that removed `-k 1` from the wget deadline command.
The resistant-child scenario exceeded the six-second driver deadline, and the unchanged source passed again after restoration.
This demonstrates sensitivity to that deadline regression only.
Do not alter the production checkout to perform a negative control; use an isolated scratch copy and preserve its cleanup.
