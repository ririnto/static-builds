import { readdirSync, readFileSync, realpathSync, statSync } from "node:fs";
import path from "node:path";
import process from "node:process";

import { isMap, isScalar, isSeq, LineCounter, parseDocument } from "yaml";

import metadata from "../.github/action-inputs.json" with { type: "json" };

interface ApprovedAction {
  readonly commit: string;
  readonly inputs: Readonly<
    Record<
      string,
      string | { readonly context: string } | { readonly default: null } | null
    >
  >;
}

const approvedActions = new Map(
  Object.entries<ApprovedAction>(metadata).map(
    ([reference, { commit, inputs }]) => [
      reference,
      /^[0-9a-f]{40}$/u.test(commit)
        ? new Map(
            Object.entries(inputs).map(([name, value]) => [
              name.toLowerCase(),
              value !== null && typeof value === "object" && "default" in value
                ? ""
                : value
            ])
          )
        : null
    ]
  )
);
const workflowsDirectory = ".github/workflows";
const implicitDefaults = new Map<string, boolean | number | string>([
  ["boolean", false],
  ["number", 0],
  ["string", ""]
]);

const repository = realpathSync(".");

const literal = (
  node: unknown
): string | number | boolean | null | undefined => {
  if (node === null) {
    return null;
  }
  if (isScalar(node) && node.anchor === undefined) {
    const value: unknown = node.value;
    if (typeof value === "string" && !value.includes("${{")) {
      return value;
    }
    if (value === null || typeof value === "boolean") {
      return value;
    }
    if (typeof value === "number" && Number.isFinite(value)) {
      if (node.format !== "HEX" && node.format !== "OCT") {
        return value;
      }
      if (value <= 0xff_ff_ff_ff) {
        return value > 0x7f_ff_ff_ff ? value - 0x1_00_00_00_00 : value;
      }
    }
  }
  return undefined;
};

const actionInput = (value: string | number | boolean | null): string => {
  if (typeof value === "number") {
    if (Object.is(value, -0)) {
      return "-0";
    }
    const [mantissa = "", power] = value.toExponential(14).split("e");
    const exponent = Number(power);
    if (exponent >= 15 || exponent < -4) {
      const trimmed = mantissa.replace(/0+$/u, "").replace(/\.$/u, "");
      const digits = String(Math.abs(exponent)).padStart(2, "0");
      return `${trimmed}E${exponent < 0 ? "-" : "+"}${digits}`;
    }
    return String(Number(value.toPrecision(15)));
  }
  return value === null ? "" : String(value);
};

const readYaml = (file: string, lineCounter?: LineCounter) => {
  const document = parseDocument(readFileSync(file, "utf-8"), { lineCounter });
  if (document.errors.length > 0) {
    throw new Error(
      `${file}: ${document.errors.map((error) => error.message).join("\n")}`
    );
  }
  return document;
};

const repositoryFile = (file: string): string | null => {
  if (statSync(file, { throwIfNoEntry: false })?.isFile() === true) {
    const resolved = realpathSync(file);
    if (resolved.startsWith(`${repository}${path.sep}`)) {
      return resolved;
    }
  }
  return null;
};

const localActionDefaults = (file: string): Map<string, string> => {
  const defaults = new Map<string, string>();
  const inputs = readYaml(file).get("inputs", true);
  if (isMap(inputs)) {
    for (const { key, value: declaration } of inputs.items) {
      if (
        isScalar(key) &&
        typeof key.value === "string" &&
        isMap(declaration)
      ) {
        const entry = declaration.items.find(
          ({ key: field }) =>
            isScalar(field) &&
            typeof field.value === "string" &&
            field.value.toLowerCase() === "default"
        );
        const value = literal(entry?.value);
        if (value !== undefined) {
          defaults.set(key.value.toLowerCase(), actionInput(value));
        }
      }
    }
  }
  return defaults;
};

const reusableInputsFor = (trigger: unknown) => {
  if (isMap(trigger)) {
    const call = trigger.get("workflow_call", true);
    if (isMap(call)) {
      const inputs = call.get("inputs", true);
      return isMap(inputs) ? inputs : true;
    }
    if (literal(call) === null) {
      return true;
    }
  }
  if (
    literal(trigger) === "workflow_call" ||
    (isSeq(trigger) &&
      trigger.items.some((item) => literal(item) === "workflow_call"))
  ) {
    return true;
  }
  return null;
};

const report = (
  file: string,
  lines: LineCounter,
  offset: number,
  message: string
) => {
  const { line, col } = lines.linePos(offset);
  process.stderr.write(`${file.slice(2)}:${line}:${col}: ${message}\n`);
  process.exitCode = 1;
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
      const file = `${workflowsDirectory}/${name}`;
      const lines = new LineCounter();
      return [`./${file}`, { document: readYaml(file, lines), lines }];
    })
);

for (const [file, { document, lines }] of workflows) {
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
      const reference = invocation.get("uses", true);
      if (
        !isScalar(reference) ||
        typeof reference.value !== "string" ||
        !reference.range
      ) {
        continue;
      }
      const uses = reference.value;
      const self = uses.startsWith("$/");
      const local = self || uses.startsWith("./");
      const relative = uses.slice(2);
      const localPath = `./${relative}`;
      const safeLocalPath =
        local &&
        !uses.includes("${{") &&
        !uses.includes("\\") &&
        !relative.includes("\0") &&
        !relative.startsWith("/") &&
        !relative.split("/").includes("..") &&
        (!self || (!uses.includes("@") && relative.length > 0));
      const reusable = invocation === job;
      const reusableInputs =
        reusable && safeLocalPath
          ? reusableInputsFor(
              workflows.get(localPath)?.document.get("on", true)
            )
          : null;
      const manifest =
        !reusable && safeLocalPath
          ? (repositoryFile(`${localPath}/action.yml`) ??
            repositoryFile(`${localPath}/action.yaml`))
          : null;
      if (
        self &&
        !(reusable
          ? reusableInputs !== null
          : manifest !== null ||
            (safeLocalPath &&
              (repositoryFile(`${localPath}/Dockerfile`) !== null ||
                repositoryFile(`${localPath}/dockerfile`) !== null)))
      ) {
        report(
          file,
          lines,
          reference.range[0],
          `R35/invalid-self-reference: Invalid same-repository reference "${uses}".`
        );
        continue;
      }
      const inputs = invocation.get("with", true);
      if (!isMap(inputs) || inputs.anchor !== undefined) {
        continue;
      }
      const externalAction =
        reusable || local ? null : approvedActions.get(uses);
      const action =
        manifest === null ? externalAction : localActionDefaults(manifest);
      for (const { key, value } of inputs.items) {
        const input = literal(value);
        if (
          !isScalar(key) ||
          typeof key.value !== "string" ||
          !key.range ||
          key.anchor !== undefined ||
          input === undefined
        ) {
          continue;
        }
        const name = key.value;
        const declaration = isMap(reusableInputs)
          ? reusableInputs.get(name, true)
          : null;
        const actionDefault = action?.get(name.toLowerCase());
        const reusableType = isMap(declaration)
          ? declaration.get("type")
          : null;
        const reusableInput =
          input === null && reusableType === "string" ? "" : input;
        const defaultValue =
          isMap(declaration) && declaration.has("default")
            ? literal(declaration.get("default", true))
            : implicitDefaults.get(typeof reusableInput);
        if (
          (typeof actionDefault === "string" &&
            !actionDefault.includes("${{") &&
            actionDefault === actionInput(input)) ||
          (isMap(declaration) &&
            declaration.get("required") !== true &&
            reusableType === typeof reusableInput &&
            (defaultValue === null && reusableType === "string"
              ? ""
              : defaultValue) === reusableInput)
        ) {
          report(
            file,
            lines,
            key.range[0],
            `R35/workflow-input-defaults: Omit "${name}" for "${uses}". It equals the declared default.`
          );
        }
      }
    }
  }
}
