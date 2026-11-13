export const actions = {
  Acquire: "Parent acquires the lock",
  Close: "Parent closes its descriptor",
  Done: "Both downloaders remain stopped",
  FinishTransport: "Transport closes its descriptor",
  StartTransport: "Transport inherits the descriptor",
  Stop: "Parent exits"
};

/**
 * @param {Record<string, string>} values TLC scalar fields.
 * @returns {string} First downloader's visible ownership.
 */
export const state = (values) => {
  const phases = /^<<(?<items>.*)>>$/u.exec(values.phase);
  if (!phases) {
    throw new Error(`Unsupported downloader phases: ${values.phase}`);
  }
  const phase =
    /^"(?<phase>\w+)"/u.exec(phases.groups?.items ?? "")?.groups?.phase ?? "";
  /** @type {Record<string, string>} */
  const labels = {
    downloading: "Parent and transport hold the lock",
    owned: "Parent holds the lock",
    ready: "Transport stopped, parent holds the lock",
    waiting: "Parent waits for the lock"
  };
  let label = labels[phase];
  if (phase === "done") {
    label =
      values.transports === "{1}"
        ? "Parent stopped, transport holds the lock"
        : "Parent stopped, no descriptor remains";
  }
  if (!label) {
    throw new Error(`Unmapped downloader phase: ${phase}`);
  }
  return label;
};
