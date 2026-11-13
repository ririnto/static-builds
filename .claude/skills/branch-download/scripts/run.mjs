import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import * as view from "./branch-lock/statechart.mjs";
/**
 * @typedef {{initial: boolean, values: Record<string, string>}} State
 * @typedef {{action: string, from: string, to: string}} Edge
 */

const [mode] = process.argv.slice(2);
if (!["--check", "--write"].includes(mode) || process.argv.length !== 3) {
  throw new Error(
    "Usage: node .claude/skills/branch-download/scripts/run.mjs --check|--write"
  );
}
const skill = fileURLToPath(new URL("..", import.meta.url));
const require = createRequire(import.meta.url);
const temporary = mkdtempSync(path.join(tmpdir(), "branch-lock-"));
try {
  const result = spawnSync(
    "java",
    [
      "-Xmx512m",
      "-XX:+UseParallelGC",
      "-cp",
      require.resolve("tla-chaos-loop/tools/tla2tools.jar"),
      "tlc2.TLC",
      "-workers",
      "1",
      "-noGenerateSpecTE",
      "-metadir",
      path.join(temporary, "states"),
      "-dump",
      "dot,actionlabels",
      path.join(temporary, "graph.dot"),
      "-config",
      path.join(skill, "assets/branch-lock/BranchLock.cfg"),
      path.join(skill, "assets/branch-lock/BranchLock.tla")
    ],
    { encoding: "utf-8", maxBuffer: 8 * 1024 * 1024, timeout: 60_000 }
  );
  const log = `${result.stdout ?? ""}${result.stderr ?? ""}`;
  process.stdout.write(log);
  if (
    result.error ||
    result.status !== 0 ||
    !log.includes("Model checking completed. No error has been found.")
  ) {
    throw new Error(
      `TLC did not complete successfully (exit ${result.status}). ${result.error ?? ""}`
    );
  }
  /** @type {Map<string, State>} */
  const states = new Map();
  /** @type {Edge[]} */
  const edges = [];
  for (const rawLine of readFileSync(
    path.join(temporary, "graph.dot"),
    "utf-8"
  ).split("\n")) {
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
  const projected = new Map(
    [...states].map(([id, state]) => [id, view.state(state.values)])
  );
  const labels = [...new Set(projected.values())].toSorted();
  for (const label of labels) {
    if (typeof label !== "string" || !/^[\w ,/-]+$/u.test(label)) {
      throw new Error(`Unsupported Mermaid state label: ${label}`);
    }
  }
  const ids = new Map(labels.map((label, index) => [label, `s${index}`]));
  const starts = new Set(
    [...states]
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
  for (const edge of edges) {
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
  const mermaid = [
    "stateDiagram-v2",
    ...labels.map((label) => `  ${ids.get(label)}: ${label}`),
    ...[...starts].toSorted(),
    ...[...arrows].toSorted(),
    ""
  ].join("\n");
  const output = path.join(skill, "references/branch-lock/statechart.mmd");
  if (mode === "--write") {
    writeFileSync(output, mermaid);
  } else if (readFileSync(output, "utf-8") !== mermaid) {
    throw new Error(
      "Generated statechart is stale. Run pnpm generate:statechart."
    );
  }
  console.log(
    `Project ${states.size} model states and ${edges.length} explored transitions into ${labels.length} display states and ${arrows.size} arrows.\nHide ${hidden} transitions within the same display state.`
  );
} finally {
  rmSync(temporary, { force: true, recursive: true });
}
