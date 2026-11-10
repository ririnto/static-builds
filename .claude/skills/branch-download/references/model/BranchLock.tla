-------------------------- MODULE BranchLock --------------------------
VARIABLES phase, lock, recorded, interrupted
vars == <<phase, lock, recorded, interrupted>>

Init == /\ phase = "beforeMkdir"
        /\ lock = FALSE
        /\ recorded = FALSE
        /\ interrupted = FALSE

Mkdir == /\ phase = "beforeMkdir"
         /\ lock' = TRUE
         /\ phase' = "beforeDefer"
         /\ UNCHANGED <<recorded, interrupted>>
RegisterCleanup == /\ phase = "beforeDefer"
                   /\ recorded' = TRUE
                   /\ phase' = "work"
                   /\ UNCHANGED <<lock, interrupted>>
Signal == /\ phase \in {"beforeMkdir", "beforeDefer", "work", "cleanup"}
          /\ ~interrupted
          /\ interrupted' = TRUE
          /\ UNCHANGED <<phase, lock, recorded>>
Return == /\ phase = "work"
          /\ phase' = "cleanup"
          /\ UNCHANGED <<lock, recorded, interrupted>>
RemoveLock == /\ phase = "cleanup"
              /\ recorded
              /\ lock' = FALSE
              /\ phase' = "done"
              /\ UNCHANGED <<recorded, interrupted>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Mkdir \/ RegisterCleanup \/ Signal \/ Return \/ RemoveLock \/ Done
NoAbandonedLock == (phase = "done" /\ interrupted) => ~lock
=============================================================================
