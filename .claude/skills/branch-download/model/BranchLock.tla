-------------------------- MODULE BranchLock --------------------------
VARIABLES phase, lock, recorded, interrupted
vars == <<phase, lock, recorded, interrupted>>

Init == /\ phase = "beforeMkdir"
        /\ lock = FALSE
        /\ recorded = FALSE
        /\ interrupted = FALSE

Mkdir == /\ phase = "beforeMkdir"
         /\ lock' = TRUE
         /\ phase' = "beforeRecord"
         /\ UNCHANGED <<recorded, interrupted>>
RecordLock == /\ phase = "beforeRecord"
              /\ recorded' = TRUE
              /\ phase' = "work"
              /\ UNCHANGED <<lock, interrupted>>
CompleteWork == /\ phase = "work"
                /\ phase' = "exitTrap"
                /\ UNCHANGED <<lock, recorded, interrupted>>
Signal == /\ phase \in {"beforeMkdir", "beforeRecord", "work"}
          /\ phase' = "signalTrap"
          /\ interrupted' = TRUE
          /\ UNCHANGED <<lock, recorded>>
SignalExit == /\ phase = "signalTrap"
              /\ phase' = "exitTrap"
              /\ UNCHANGED <<lock, recorded, interrupted>>
CheckLock == /\ phase = "exitTrap"
             /\ phase' = IF recorded THEN "removeLock" ELSE "done"
             /\ UNCHANGED <<lock, recorded, interrupted>>
RemoveLock == /\ phase = "removeLock"
              /\ lock' = FALSE
              /\ phase' = "done"
              /\ UNCHANGED <<recorded, interrupted>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Mkdir \/ RecordLock \/ CompleteWork \/ Signal \/ SignalExit \/
        CheckLock \/ RemoveLock \/ Done
NoAbandonedLock == (phase = "done" /\ interrupted) => ~lock
=============================================================================
