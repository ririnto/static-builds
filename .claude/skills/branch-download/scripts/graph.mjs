/**
 * @typedef {{initial: boolean, values: Record<string, string>}} State
 * @typedef {{action: string, from: string, to: string}} Edge
 * @typedef {{states: Map<string, State>, edges: Edge[]}} Graph
 */

/**
 * Parse TLC's scalar fields without evaluating TLA+ expressions.
 * @param {string} dot TLC export.
 * @returns {Graph} Explored graph.
 */
export const parseGraph = (dot) => {
  /** @type {Map<string, State>} */
  const states = new Map();
  /** @type {Edge[]} */
  const edges = [];
  for (const rawLine of dot.split("\n")) {
    const line = rawLine.trim();
    const edge =
      /^(?<from>-?\d+) -> (?<to>-?\d+) \[label=(?<label>"(?:\\.|[^"\\])*")/u.exec(
        line
      );
    const state =
      /^(?<id>-?\d+) \[label=(?<label>"(?:\\.|[^"\\])*")(?<attributes>.*)$/u.exec(
        line
      );
    if (edge?.groups) {
      edges.push({
        action: String(JSON.parse(edge.groups.label)),
        from: edge.groups.from,
        to: edge.groups.to
      });
    } else if (state?.groups) {
      const values = Object.fromEntries(
        [
          ...String(JSON.parse(state.groups.label)).matchAll(
            /^\/\\ (?<name>\w+) = (?<value>.*)$/gmu
          )
        ].map((match) => [match.groups?.name ?? "", match.groups?.value ?? ""])
      );
      states.set(state.groups.id, {
        initial: state.groups.attributes.includes("style = filled"),
        values
      });
    } else if (/^-?\d+ (?:->|\[)/u.test(line)) {
      throw new Error(`Unsupported TLC graph entry: ${line}`);
    }
  }
  if (
    states.size === 0 ||
    ![...states.values()].some((state) => state.initial)
  ) {
    throw new Error("TLC graph has no states or initial state.");
  }
  for (const edge of edges) {
    if (!states.has(edge.from) || !states.has(edge.to)) {
      throw new Error("TLC graph contains an edge with an unknown state.");
    }
  }
  return { edges, states };
};

/**
 * Project explored states and edges into a deterministic Mermaid graph.
 * Only transitions within the same display state are hidden.
 * @param {Graph} graph Explored graph.
 * @param {{state: (values: Record<string, string>) => string, actions: Record<string, string>}} view Display mapping.
 * @returns {{mermaid: string, summary: string}} Generated text.
 */
export const renderGraph = (graph, view) => {
  const projected = new Map(
    [...graph.states].map(([id, state]) => [id, view.state(state.values)])
  );
  const labels = [...new Set(projected.values())].toSorted();
  for (const label of labels) {
    if (typeof label !== "string" || !/^[\w ,/-]+$/u.test(label)) {
      throw new Error(`Unsupported Mermaid state label: ${label}`);
    }
  }
  const ids = new Map(labels.map((label, index) => [label, `s${index}`]));
  const starts = new Set(
    [...graph.states]
      .filter(([, state]) => state.initial)
      .map(([id]) => {
        const label = projected.get(id);
        if (label === undefined) {
          throw new Error("Missing initial display state.");
        }
        return `  [*] --> ${ids.get(label)}`;
      })
  );
  /** @type {Set<string>} */
  const arrows = new Set();
  let hidden = 0;
  for (const edge of graph.edges) {
    const from = projected.get(edge.from);
    const to = projected.get(edge.to);
    if (from === undefined || to === undefined) {
      throw new Error("Missing display state for explored transition.");
    }
    const [action] = edge.action.split("(");
    if (!Object.hasOwn(view.actions, action)) {
      throw new Error(`Unmapped TLC action: ${action}`);
    }
    if (from === to) {
      hidden += 1;
    } else {
      arrows.add(
        `  ${ids.get(from)} --> ${ids.get(to)}: ${view.actions[action]}`
      );
    }
  }
  return {
    mermaid: [
      "stateDiagram-v2",
      ...labels.map((label) => `  ${ids.get(label)}: ${label}`),
      ...[...starts].toSorted(),
      ...[...arrows].toSorted(),
      ""
    ].join("\n"),
    summary: `Project ${graph.states.size} model states and ${graph.edges.length} explored transitions into ${labels.length} display states and ${arrows.size} arrows.\nHide ${hidden} transitions within the same display state.`
  };
};
