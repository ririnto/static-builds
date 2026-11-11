-------------------------- MODULE BranchLock --------------------------
VARIABLES phase, owner, transports
vars == <<phase, owner, transports>>
Downloaders == {1, 2}

Init == /\ phase = [d \in Downloaders |-> "waiting"]
        /\ owner = 0
        /\ transports = {}

Acquire(d) == /\ phase[d] = "waiting"
              /\ owner = 0
              /\ owner' = d
              /\ phase' = [phase EXCEPT ![d] = "owned"]
              /\ UNCHANGED transports
StartTransport(d) == /\ phase[d] = "owned"
                     /\ phase' = [phase EXCEPT ![d] = "downloading"]
                     /\ transports' = transports \cup {d}
                     /\ UNCHANGED owner
FinishTransport(d) == /\ d \in transports
                      /\ transports' = transports \ {d}
                      /\ phase' = IF phase[d] = "done" THEN phase
                                  ELSE [phase EXCEPT ![d] = "ready"]
                      /\ owner' = IF phase[d] = "done" THEN 0 ELSE owner
Close(d) == /\ phase[d] \in {"owned", "ready"}
            /\ phase' = [phase EXCEPT ![d] = "done"]
            /\ owner' = 0
            /\ UNCHANGED transports
Stop(d) == /\ phase[d] # "done"
           /\ phase' = [phase EXCEPT ![d] = "done"]
           /\ owner' = IF owner = d /\ d \notin transports THEN 0 ELSE owner
           /\ UNCHANGED transports
Done == /\ \A d \in Downloaders : phase[d] = "done"
        /\ transports = {}
        /\ UNCHANGED vars
Next == (\E d \in Downloaders : Acquire(d) \/ StartTransport(d) \/ FinishTransport(d) \/ Close(d) \/ Stop(d)) \/ Done
NoAbandonedLock == owner # 0 => (phase[owner] # "done" \/ owner \in transports)
ExclusiveOwnership == /\ \A d \in Downloaders : phase[d] \in {"owned", "downloading", "ready"} => owner = d
                      /\ \A d \in transports : owner = d
=============================================================================
