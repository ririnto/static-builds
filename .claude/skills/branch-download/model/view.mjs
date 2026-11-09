export const title = "Branch download lock ownership";

/**
 * @param {{phase: string}} value Model state.
 * @returns {string | undefined} Reader-facing state.
 */
export const state = (value) =>
  ({
    beforeMkdir: "Waiting to create the lock",
    beforeRecord: "Lock created, ownership not yet recorded",
    done: "Download has stopped",
    exitTrap: "Exit cleanup checks the lock",
    removeLock: "Cleanup removes the lock",
    signalTrap: "Interruption is handled",
    work: "Download owns its lock"
  })[value.phase];

/**
 * @param {string} action Model action.
 * @returns {string} Reader-facing step.
 */
export const step = (action) =>
  ({
    CheckLock: "Cleanup checks recorded ownership",
    CompleteWork: "Download finishes",
    Done: "Download remains stopped",
    Mkdir: "Lock is created",
    RecordLock: "Cleanup ownership is recorded",
    RemoveLock: "Lock is removed",
    Signal: "Download is interrupted",
    SignalExit: "Signal handler exits"
  })[action] ?? action;
