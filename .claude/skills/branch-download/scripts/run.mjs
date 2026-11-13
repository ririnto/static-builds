import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import * as view from "./branch-lock/statechart.mjs";
import { parseGraph, renderGraph } from "./graph.mjs";

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
  const graph = renderGraph(
    parseGraph(readFileSync(path.join(temporary, "graph.dot"), "utf-8")),
    view
  );
  const output = path.join(skill, "references/branch-lock/statechart.mmd");
  if (mode === "--write") {
    writeFileSync(output, graph.mermaid);
  } else if (readFileSync(output, "utf-8") !== graph.mermaid) {
    throw new Error(
      "Generated statechart is stale. Run pnpm generate:statechart."
    );
  }
  console.log(graph.summary);
} finally {
  rmSync(temporary, { force: true, recursive: true });
}
