import { readdirSync, readFileSync } from "node:fs";
import process from "node:process";

import { isMap, isScalar, isSeq, LineCounter, parseDocument } from "yaml";

import metadata from "../.github/action-inputs.json" with { type: "json" };

interface ApprovedAction {
  readonly commit: string;
  readonly inputs: Readonly<
    Record<string, string | { readonly context: string } | null>
  >;
}

const approvedActions = new Map<string, ApprovedAction>(
  Object.entries(metadata)
);
const workflowsDirectory = ".github/workflows";

/**
 * Read a literal scalar without evaluating GitHub Actions expressions.
 */
const literal = (node: unknown): string | number | boolean | null => {
  if (isScalar(node) && node.anchor === undefined) {
    const value: unknown = node.value;
    if (typeof value === "string" && !value.includes("${{")) {
      return value;
    }
    if (
      typeof value === "boolean" ||
      (typeof value === "number" && Number.isSafeInteger(value))
    ) {
      return value;
    }
  }
  return null;
};

const workflows = new Map(
  readdirSync(workflowsDirectory, { withFileTypes: true })
    .filter(
      (entry) =>
        entry.isFile() &&
        (entry.name.endsWith(".yml") || entry.name.endsWith(".yaml"))
    )
    .map(({ name }) => name)
    .toSorted()
    .map((name) => {
      const path = `${workflowsDirectory}/${name}`;
      const lines = new LineCounter();
      const document = parseDocument(readFileSync(path, "utf-8"), {
        lineCounter: lines
      });
      if (document.errors.length > 0) {
        throw new Error(
          `${path}: ${document.errors.map((error) => error.message).join("\n")}`
        );
      }
      return [`./${path}`, { document, lines }];
    })
);

for (const [path, { document, lines }] of workflows) {
  const jobs = document.get("jobs", true);
  if (!isMap(jobs)) {
    continue;
  }
  for (const { value: job } of jobs.items) {
    if (!isMap(job)) {
      continue;
    }
    const steps = job.get("steps", true);
    for (const invocation of isSeq(steps) ? steps.items : [job]) {
      if (!isMap(invocation)) {
        continue;
      }
      const uses = invocation.get("uses");
      const inputs = invocation.get("with", true);
      if (
        typeof uses !== "string" ||
        !isMap(inputs) ||
        inputs.anchor !== undefined
      ) {
        continue;
      }
      const action = invocation === job ? null : approvedActions.get(uses);
      const reusableInputs =
        invocation === job
          ? workflows
              .get(uses.startsWith("$/") ? `.${uses.slice(1)}` : uses)
              ?.document.getIn(["on", "workflow_call", "inputs"], true)
          : null;
      for (const { key, value } of inputs.items) {
        const input = literal(value);
        if (
          !isScalar(key) ||
          typeof key.value !== "string" ||
          !key.range ||
          key.anchor !== undefined ||
          input === null
        ) {
          continue;
        }
        const declaration = isMap(reusableInputs)
          ? reusableInputs.get(key.value, true)
          : null;
        const actionDefault = action?.inputs[key.value];
        if (
          (action &&
            /^[0-9a-f]{40}$/u.test(action.commit) &&
            typeof actionDefault === "string" &&
            !actionDefault.includes("${{") &&
            actionDefault === String(input)) ||
          (isMap(declaration) &&
            declaration.get("required") !== true &&
            declaration.get("type") === typeof input &&
            literal(declaration.get("default", true)) === input)
        ) {
          const { line, col } = lines.linePos(key.range[0]);
          process.stderr.write(
            `${path.slice(2)}:${line}:${col}: R35/workflow-input-defaults: Omit "${key.value}" for "${uses}". It equals the declared default.\n`
          );
          process.exitCode = 1;
        }
      }
    }
  }
}
