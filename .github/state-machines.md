# State machines

This inventory covers maintained source, Dockerfiles, contributor checks, and release configurations.
Upstream application internals remain outside this repository's verification scope.

## Downloads and metadata

| Machine | Transitions | Contract and evidence |
| --- | --- | --- |
| Metadata parsing | Input, object or array traversal, decoded value, result or error. | `internal/metadata/parser.go` preserves member order for command output and rejects unsupported input through errors. |
| Target resolution | Arguments, manifest lookup, rendered downloads or query output. | `cmd/download` and `cmd/metadata` reject unknown targets before downloading or returning target metadata. |
| Release download | Cache lookup, existing archive or fetch, temporary file, rename. | A nonempty cached release archive skips the fetch. |
| Branch download | Lock wait, ownership, refresh, deferred cleanup. | Every branch call refreshes its archive while holding the destination lock. |
| Fetch retry | Attempt, failed process, bounded delay, next attempt or exhaustion. | Successful nonempty downloads replace the destination through a sibling-file rename. |
| Cancellation | Signal, cancelled context, child termination or cancelled wait, cleanup. | Go command tests do not establish process-tree termination or all signal interleavings. |

The [branch-download model](../.claude/skills/branch-download/SKILL.md) checks two downloaders' exclusive descriptor ownership, including inherited transport ownership after parent exit, under its stated filesystem assumptions.
The model does not prove download integrity, retry liveness, or concurrent publication.

## Builds and publication

| Machine | Transitions | Contract and evidence |
| --- | --- | --- |
| Build task | Non-root execution, download, metadata arguments, caller-supplied client options, private target context, rootless BuildKit, export and cleanup. | `Taskfile.yaml` propagates failed commands before the next build phase. |
| Docker artifact | Source, configure, compile, install, static verification, final copy. | Each target's final stage depends on its verify stage. |
| GitHub release | Resolve tag, validate version, build, scan, package, upload, publish. | `.github/workflows/template-release.yaml` publishes only after the release job succeeds. |
| Release retry | Existing release or missing release, asset replacement or release creation. | GoReleaser owns release creation and asset replacement through `.goreleaser.yaml`. |
| GitLab package | Eligible branch, generated configuration, manual trigger, child build, artifacts. | `.gitlab-ci.yaml` permits package generation on `main` and `feature/*`. |
| | | Feature package versions end in `-beta`. |
| Contributor hooks | Git operation, checks, success or blocked operation. | Both tracked hooks invoke the complete contributor checks. |

Static verification checks metadata-selected release executables, including Apache HTTPd's `rotatelogs`.
Each released executable must have executable permissions and be identified as a statically linked ELF artifact.
Build verification checks executable startup under strace and module, plugin, or feature listings and rejects ELF interpreters or shared-library dependencies with readelf.
It does not repeat upstream functional tests.
The workflow descriptions reflect source inspection rather than hosted execution evidence.

## Custom healthcheck

The HTTP and stream implementations maintain separate peer state in shared-memory zones.
The [module source](../nginx-upstream-healthcheck/ngx_http_upstream_healthcheck_module/ngx_http_upstream_healthcheck_module.c) implements the following lifecycles.

| Machine | Transitions | Contract |
| --- | --- | --- |
| Checker ownership | Worker initialization, worker-zero scheduling, shutdown. | Shutdown stops timers and drains active probes. |
| Peer health | Initially down, consecutive successes, ready, consecutive failures, down. | `rise` and `fall` thresholds control availability. |
| | | Administratively down peers stay unavailable. |
| Probe | Scheduled, connecting, optional TLS handshake, sending, receiving, completion or deadline. | Completion closes the connection, removes the timer, updates peer state, and releases the probe pool. |
| Peer membership | Discovered, referenced by probes, removed from upstream, reclaimed. | In-flight probes retain the peer state until completion. |
| Status response | Shared state snapshot, status serialization, HTTP response. | Status reports expose health and checker activity without changing the health decision. |

The njs examples return version text and hold no persistent application state.
Markdown and lint rules traverse input syntax within one invocation and retain no cross-invocation lifecycle.
No runtime interleaving proof covers the custom module's shared memory, TLS, timers, reloads, or peer replacement.
