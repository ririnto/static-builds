import { readdirSync, readFileSync, realpathSync } from "node:fs";
import path from "node:path";
import process from "node:process";
import { parseArgs } from "node:util";

import {
  isAlias,
  isMap,
  isScalar,
  isSeq,
  LineCounter,
  parseDocument,
  visit
} from "yaml";
import type { Document } from "yaml";

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

const { values: options } = parseArgs({
  options: {
    json: { type: "boolean" },
    repository: { type: "string" }
  }
});
const repository = realpathSync(options.repository ?? ".");

const resolve = (document: Document, node: unknown): unknown =>
  isAlias(node) ? node.resolve(document) : node;

const field = (document: Document, map: unknown, name: string): unknown =>
  isMap(map)
    ? map.items.find(({ key }) => {
        const resolved = resolve(document, key);
        return isScalar(resolved) && resolved.value === name;
      })?.value
    : undefined;

const readField = (document: Document, map: unknown, name: string): unknown =>
  resolve(document, field(document, map, name));

const literal = (
  node: unknown
): string | number | boolean | null | undefined => {
  if (node === null) {
    return null;
  }
  if (isScalar(node)) {
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

const isMidpoint = (value: number, digits: bigint, exponent: number) => {
  const midpoint = digits * 10n - 5n;
  const places = 15 - exponent;
  if (places <= 0) {
    return (
      Number.isInteger(value) &&
      BigInt(value) === midpoint * 10n ** BigInt(-places)
    );
  }
  const fives = 5n ** BigInt(places);
  const scaled = value * 2 ** places;
  return (
    midpoint % fives === 0n &&
    Number.isInteger(scaled) &&
    BigInt(scaled) === midpoint / fives
  );
};

const runnerNumber = (value: number): string => {
  const magnitude = Math.abs(value);
  const [mantissa = "", power] = magnitude.toExponential(14).split("e");
  const exponent = Number(power);
  const rounded = BigInt(mantissa.replace(".", ""));
  // GitHub's runner uses .NET G15, which rounds exact midpoints to even.
  const digits =
    rounded % 2n === 1n && isMidpoint(magnitude, rounded, exponent)
      ? rounded - 1n
      : rounded;
  const sign = value < 0 || Object.is(value, -0) ? "-" : "";
  const significant = String(digits).padStart(15, "0");
  if (exponent >= 15 || exponent < -4) {
    const fraction = significant.slice(1).replace(/0+$/u, "");
    const significand =
      fraction === "" ? significant[0] : `${significant[0]}.${fraction}`;
    const powerDigits = String(Math.abs(exponent)).padStart(2, "0");
    return `${sign}${significand}E${exponent < 0 ? "-" : "+"}${powerDigits}`;
  }
  return `${sign}${Number(`${digits}e${exponent - 14}`)}`;
};

const inputText = (value: string | number | boolean | null): string => {
  if (typeof value === "number") {
    return runnerNumber(value);
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

const localActionDefaults = (file: string): Map<string, string> => {
  const defaults = new Map<string, string>();
  const document = readYaml(file);
  const inputs = readField(document, document.contents, "inputs");
  if (isMap(inputs)) {
    for (const { key, value } of inputs.items) {
      const name = literal(resolve(document, key));
      const declaration = resolve(document, value);
      if (typeof name === "string" && isMap(declaration)) {
        const entry = declaration.items.find(({ key: fieldName }) => {
          const fieldValue = literal(resolve(document, fieldName));
          return (
            typeof fieldValue === "string" &&
            fieldValue.toLowerCase() === "default"
          );
        });
        const input = literal(resolve(document, entry?.value));
        if (input !== undefined) {
          defaults.set(name.toLowerCase(), inputText(input));
        }
      }
    }
  }
  return defaults;
};

const manifestDefaults = (directory: string): Map<string, string> | null => {
  for (const filename of ["action.yml", "action.yaml"]) {
    try {
      const file = realpathSync(path.join(repository, directory, filename));
      return file.startsWith(`${repository}${path.sep}`)
        ? localActionDefaults(file)
        : null;
    } catch (error) {
      if (
        !(error instanceof Error && "code" in error && error.code === "ENOENT")
      ) {
        return null;
      }
    }
  }
  return null;
};

const report = (
  file: string,
  lines: LineCounter,
  offset: number,
  input: string,
  reference: string,
  shared: boolean,
  removedAnchors: readonly string[]
) => {
  const { line, col } = lines.linePos(offset);
  const code = "R35/workflow-input-defaults";
  const filename = file.slice(2);
  if (options.json === true) {
    process.stdout.write(
      `${JSON.stringify({ code, file: filename, input, reference, line, column: col })}\n`
    );
  } else {
    const advice = shared ? "Review the shared definition of" : "Omit";
    const anchors =
      removedAnchors.length > 0
        ? ` Removing the input also removes ${removedAnchors.join(", ")}.`
        : "";
    process.stderr.write(
      `${filename}:${line}:${col}: ${code}: ${advice} "${input}" for "${reference}". It equals the default.${anchors}\n`
    );
  }
  process.exitCode = 1;
};

const workflows = new Map(
  readdirSync(path.join(repository, workflowsDirectory), {
    withFileTypes: true
  })
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
      return [
        `./${file}`,
        { document: readYaml(path.join(repository, file), lines), lines }
      ];
    })
);

const localPath = (reference: string): string | null => {
  const relative = reference.slice(2);
  if (
    relative.includes("\\") ||
    relative.includes("\0") ||
    relative.startsWith("/") ||
    relative.split("/").includes("..") ||
    (reference.startsWith("$/") && relative.includes("@"))
  ) {
    return null;
  }
  return `./${relative}`;
};

const reusableDefaults = (reference: string) => {
  const filename = localPath(reference);
  const document = filename === null ? null : workflows.get(filename)?.document;
  if (document) {
    const trigger = readField(document, document.contents, "on");
    const call = readField(document, trigger, "workflow_call");
    const inputs = readField(document, call, "inputs");
    if (isMap(inputs)) {
      return { document, inputs };
    }
  }
  return null;
};

const matchesReusableDefault = (
  target: NonNullable<ReturnType<typeof reusableDefaults>>,
  name: string,
  input: string | number | boolean | null
): boolean => {
  const { document, inputs } = target;
  const declaration = readField(document, inputs, name);
  const type = literal(readField(document, declaration, "type"));
  const caller = input === null && type === "string" ? "" : input;
  if (
    !isMap(declaration) ||
    literal(readField(document, declaration, "required")) === true ||
    type !== typeof caller
  ) {
    return false;
  }
  const node = field(document, declaration, "default");
  const value =
    node === undefined
      ? implicitDefaults.get(type)
      : literal(resolve(document, node));
  if (typeof value === "number" && typeof caller === "number") {
    return runnerNumber(value) === runnerNumber(caller);
  }
  return (value === null && type === "string" ? "" : value) === caller;
};

for (const [file, { document, lines }] of workflows) {
  const jobs = readField(document, document.contents, "jobs");
  if (!isMap(jobs)) {
    continue;
  }
  const aliased = new Set<unknown>();
  visit(document, {
    Alias(_key, alias) {
      const definition = alias.resolve(document);
      if (definition) {
        aliased.add(definition);
      }
    }
  });
  for (const { value: rawJob } of jobs.items) {
    const job = resolve(document, rawJob);
    const rawSteps = field(document, job, "steps");
    const steps = resolve(document, rawSteps);
    const reusable = !isSeq(steps);
    const invocations = reusable ? [rawJob] : steps.items;
    for (const [index, rawInvocation] of invocations.entries()) {
      const invocation = resolve(document, rawInvocation);
      const rawInputs = field(document, invocation, "with");
      const inputs = resolve(document, rawInputs);
      const reference = literal(readField(document, invocation, "uses"));
      if (
        !isMap(inputs) ||
        inputs.items.length === 0 ||
        typeof reference !== "string"
      ) {
        continue;
      }

      const workspace = reference.startsWith("./");
      const self = reference.startsWith("$/");
      const local = workspace || self;
      const target = reusable && local ? reusableDefaults(reference) : null;
      let action: ReadonlyMap<string, unknown> | null | undefined;
      if (!reusable) {
        if (local) {
          // Prior steps can replace the runner workspace behind a ./ reference.
          const directory =
            workspace && index > 0 ? null : localPath(reference);
          action = directory === null ? null : manifestDefaults(directory);
        } else {
          action = approvedActions.get(reference);
        }
      }
      if (!target && !action) {
        continue;
      }

      const sharedCollection =
        aliased.has(jobs) ||
        aliased.has(job) ||
        aliased.has(steps) ||
        aliased.has(invocation) ||
        aliased.has(inputs);
      for (const { key, value } of inputs.items) {
        const inputKey = resolve(document, key);
        const inputValue = resolve(document, value);
        const name = literal(inputKey);
        const input = literal(inputValue);
        if (typeof name !== "string" || input === undefined) {
          continue;
        }
        const actionDefault = action?.get(name.toLowerCase());
        const matches = target
          ? matchesReusableDefault(target, name, input)
          : typeof actionDefault === "string" &&
            !actionDefault.includes("${{") &&
            actionDefault === inputText(input);
        if (matches) {
          const alias = [rawSteps, rawInvocation, rawInputs, key, value].find(
            isAlias
          );
          const location =
            alias?.range ?? (isScalar(inputKey) ? inputKey.range : null);
          if (location) {
            const shared =
              sharedCollection ||
              aliased.has(inputKey) ||
              aliased.has(inputValue);
            const removedAnchors = [inputKey, inputValue].flatMap((node) =>
              isScalar(node) && node.anchor !== undefined && !shared
                ? [node.anchor]
                : []
            );
            report(
              file,
              lines,
              location[0],
              name,
              reference,
              shared,
              removedAnchors
            );
          }
        }
      }
    }
  }
}
