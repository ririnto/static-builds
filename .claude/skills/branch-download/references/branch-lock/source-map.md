# Download ownership correspondence

Compare this mapping with the working branch before relying on the model.
The correspondence was inspected against `main` on 2026-11-13.
The model and implementation may drift independently.

| Model action | Production boundary in `cmd/download/main.go` | Correspondence |
| --- | --- | --- |
| Acquire | Successful nonblocking `syscall.Flock` in `download` | A parent gains exclusive ownership of the persistent lock inode. |
| StartTransport | `command.ExtraFiles` and transport start | The transport inherits the lock descriptor. |
| FinishTransport | Transport process exit | Its inherited descriptor closes. |
| Close | Deferred `lock.Close` | The parent releases its descriptor after a cache hit, failure, or completed transport. |
| Stop | Parent process exit | A surviving transport retains its inherited descriptor. |
| Done | Both parents and transports have stopped | No further ownership transition remains. |

The model treats cancellation as parent exit or transport completion, not a timing guarantee.
Normal cancellation waits for transport termination before deferred parent cleanup.
SIGKILL can leave a transport alive after the parent exits.

| Invariant | Required behavior |
| --- | --- |
| NoAbandonedLock | Every held lock has a live parent or transport. |
| ExclusiveOwnership | Every working parent and transport owns the same archive's exclusive lock. |

Never remove a lock file while any downloader may use it.
Removing the pathname can let waiters acquire different inodes for the same archive.
