export const title = "Branch download lock cleanup";

/**
 * @param {{phase: string}} value Model state.
 * @returns {string | undefined} Reader-facing state.
 */
export const state = (value) =>
  ({
    beforeDefer: "Lock created, cleanup not yet registered",
    beforeMkdir: "Waiting to create the lock",
    cleanup: "Download is returning",
    done: "Download has stopped",
    work: "Download owns its lock"
  })[value.phase];

/**
 * @param {string} action Model action.
 * @returns {string} Reader-facing step.
 */
export const step = (action) =>
  ({
    Done: "Download remains stopped",
    Mkdir: "Lock is created",
    RegisterCleanup: "Lock cleanup is registered",
    RemoveLock: "Lock is removed",
    Return: "Download returns",
    Signal: "Cancellation is requested"
  })[action] ?? action;
