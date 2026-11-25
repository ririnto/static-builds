import { readdirSync, readFileSync, realpathSync } from "node:fs";
import path from "node:path";
import process from "node:process";
import { parseArgs } from "node:util";

import {
  Composer,
  CST,
  isAlias,
  isMap,
  isNode,
  isScalar,
  isSeq,
  LineCounter,
  parseDocument,
  Parser,
  visit
} from "yaml";
import type { Alias, Document, Node, YAMLMap } from "yaml";

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
    list: { type: "boolean" },
    repository: { type: "string" }
  }
});
const repository = realpathSync(options.repository ?? ".");

const resolve = (document: Document, node: unknown): unknown =>
  isAlias(node) ? node.resolve(document) : node;

const fieldPair = (document: Document, map: unknown, name: string) =>
  isMap(map)
    ? map.items.find(({ key }) => {
        const resolved = resolve(document, key);
        return isScalar(resolved) && resolved.value === name;
      })
    : undefined;

const field = (document: Document, map: unknown, name: string): unknown =>
  fieldPair(document, map, name)?.value;

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

const checkedDocument = (file: string, document: Document) => {
  if (document.errors.length > 0) {
    throw new Error(
      `${file}: ${document.errors.map((error) => error.message).join("\n")}`
    );
  }
  return document;
};

const localActionDefaults = (file: string): Map<string, string> => {
  const defaults = new Map<string, string>();
  const document = checkedDocument(
    file,
    parseDocument(readFileSync(file, "utf-8"))
  );
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

const manifest = (directory: string) => {
  try {
    const root = realpathSync(path.join(repository, directory));
    if (root === repository || root.startsWith(`${repository}${path.sep}`)) {
      const [filename, alternate] = readdirSync(root).filter(
        (name) => name === "action.yml" || name === "action.yaml"
      );
      if (alternate !== undefined) {
        return { kind: "ambiguous" } as const;
      }
      if (filename !== undefined) {
        const file = realpathSync(path.join(root, filename));
        if (file.startsWith(`${repository}${path.sep}`)) {
          return {
            kind: "available",
            defaults: localActionDefaults(file)
          } as const;
        }
      }
    }
  } catch {
    return { kind: "unavailable" } as const;
  }
  return { kind: "unavailable" } as const;
};

interface Location {
  readonly file: string;
  readonly line: number;
  readonly column: number;
}

interface Anchor {
  readonly name: string;
  readonly definition: Location;
  readonly uses: readonly Location[];
}

type Advice = "omit" | "hand-edit" | "detach";

const location = (
  file: string,
  lines: LineCounter,
  offset: number
): Location => {
  const { line, col } = lines.linePos(offset);
  return { file: file.slice(2), line, column: col };
};

const locationText = ({ file, line, column }: Location) =>
  `${file}:${line}:${column}`;

const report = (
  finding: Location & {
    readonly input: string;
    readonly reference: string;
    readonly advice: Advice;
    readonly actions: readonly string[];
    readonly anchors: readonly Anchor[];
  }
) => {
  const code = "R35/workflow-input-defaults";
  if (options.json === true) {
    process.stdout.write(`${JSON.stringify({ code, ...finding })}\n`);
  } else {
    const { input, reference, advice, actions, anchors } = finding;
    const subject = `"${input}" for "${reference}" equals the default.`;
    const fix = {
      omit: `Omit "${input}".`,
      "hand-edit": `Hand edit "${input}" and preserve its alias references.`,
      detach:
        "Hand edit this call by detaching only this step's mapping. Preserve the shared definition."
    }[advice];
    const consumers =
      actions.length > 1
        ? ` Actions using the mapping: ${actions.join(", ")}.`
        : "";
    const details = anchors
      .map(({ name, definition, uses }) =>
        uses.length > 0
          ? ` &${name} is defined at ${locationText(definition)} and used at ${uses.map(locationText).join(", ")}.`
          : ` Removing the input also removes &${name} at ${locationText(definition)}.`
      )
      .join("");
    process.stderr.write(
      `${locationText(finding)}: ${code}: ${subject} ${fix}${consumers}${details}\n`
    );
  }
  process.exitCode = 1;
};

const listEntry = (entry: object) => {
  process.stdout.write(`${JSON.stringify(entry)}\n`);
};

if (options.list === true) {
  for (const [reference, { inputs }] of Object.entries<ApprovedAction>(
    metadata
  )) {
    for (const [input, value] of Object.entries(inputs)) {
      if (value !== null && typeof value === "object" && "context" in value) {
        listEntry({
          kind: "context-dependent",
          reference,
          input,
          condition: value.context
        });
      }
    }
  }
}

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
      const source = readFileSync(path.join(repository, file), "utf-8");
      const parser = new Parser((offset) => {
        lines.addNewLine(offset);
      });
      const tokens = [...parser.parse(source)];
      const [document, extra] = new Composer({ keepSourceTokens: true })
        .compose(tokens, true, source.length)
        .toArray();
      if (document === undefined || extra !== undefined) {
        throw new Error(`${file}: expected one YAML document`);
      }
      return [
        `./${file}`,
        { document: checkedDocument(file, document), lines, tokens }
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

type MatchDefault = (
  input: string,
  value: ReturnType<typeof literal>
) => boolean;

interface Invocation {
  readonly reference: string | null;
  readonly path: readonly unknown[];
  readonly mappingAlias: Alias | undefined;
  readonly matchesDefault: MatchDefault;
}

const workspaceRisk = (
  document: Document,
  step: unknown,
  shell: unknown,
  reference: string | null
): string | null => {
  if (literal(readField(document, step, "if")) === false) {
    return null;
  }
  if (reference === "actions/checkout@v7") {
    const inputs = readField(document, step, "with");
    const defaultInputs = [
      ["repository", `\${{ github.repository }}`],
      ["ref", `\${{ github.sha }}`],
      ["github-server-url", `\${{ github.server_url }}`],
      ["path", "."],
      ["sparse-checkout", ""]
    ] as const;
    const unchanged =
      !isMap(inputs) ||
      defaultInputs.every(([name, expected]) => {
        const entry = inputs.items.find(({ key }) => {
          const value = literal(resolve(document, key));
          return typeof value === "string" && value.toLowerCase() === name;
        });
        const node = resolve(document, entry?.value);
        const value: unknown = isScalar(node) ? node.value : node;
        return (
          value === undefined ||
          value === null ||
          (typeof value === "string" &&
            (value.trim() === "" || value.trim() === expected))
        );
      });
    return unchanged ? null : "Earlier checkout can change workspace contents";
  }
  const command = literal(readField(document, step, "run"));
  const interpreter = readField(document, step, "shell") ?? shell;
  const shellName = literal(interpreter);
  if (
    typeof command === "string" &&
    (interpreter === undefined ||
      interpreter === null ||
      shellName === "bash" ||
      shellName === "sh") &&
    fieldPair(document, step, "env") === undefined &&
    /^(?:echo(?: +[\w.,!:/-]+)*|true|:)$/u.test(command.trim())
  ) {
    return null;
  }
  return fieldPair(document, step, "uses")
    ? "Earlier action can change workspace contents"
    : "Earlier script can change workspace contents";
};

for (const [file, { document, lines, tokens }] of workflows) {
  const jobsEntry = fieldPair(document, document.contents, "jobs");
  const jobs = resolve(document, jobsEntry?.value);
  if (!isMap(jobs)) {
    continue;
  }
  const aliases = new Map<Node, Alias[]>();
  visit(document, {
    Alias(_key, alias) {
      const definition = alias.resolve(document);
      if (definition) {
        const uses = aliases.get(definition);
        if (uses) {
          uses.push(alias);
        } else {
          aliases.set(definition, [alias]);
        }
      }
    }
  });

  const anchorOffsets = new Map<CST.Token, number>();
  for (const token of tokens) {
    if (token.type === "document") {
      CST.visit(token, ({ start, key, sep, value }) => {
        const keyAnchor = start.find(({ type }) => type === "anchor");
        const valueAnchor = sep?.find(({ type }) => type === "anchor");
        if (key && keyAnchor) {
          anchorOffsets.set(key, keyAnchor.offset);
        }
        if (value) {
          const anchor = sep ? valueAnchor : keyAnchor;
          if (anchor) {
            anchorOffsets.set(value, anchor.offset);
          }
        }
      });
    }
  }

  const consumers = new Map<YAMLMap, Invocation[]>();
  const workflowDefaults = readField(document, document.contents, "defaults");
  const workflowRun = readField(document, workflowDefaults, "run");
  for (const { key: jobKey, value: rawJob } of jobs.items) {
    const job = resolve(document, rawJob);
    const stepsEntry = fieldPair(document, job, "steps");
    const steps = resolve(document, stepsEntry?.value);
    const reusable = !isSeq(steps);
    const invocations = reusable ? [rawJob] : steps.items;
    const jobDefaults = readField(document, job, "defaults");
    const jobRun = readField(document, jobDefaults, "run");
    const shell =
      readField(document, jobRun, "shell") ??
      readField(document, workflowRun, "shell");
    let priorRisk: string | null = null;
    for (const rawInvocation of invocations) {
      const invocation = resolve(document, rawInvocation);
      const uses = fieldPair(document, invocation, "uses");
      const resolvedReference = literal(resolve(document, uses?.value));
      const reference =
        typeof resolvedReference === "string" ? resolvedReference : null;
      const precedingRisk = priorRisk;
      if (!reusable) {
        priorRisk ??= workspaceRisk(document, invocation, shell, reference);
      }
      const withEntry = fieldPair(document, invocation, "with");
      const inputs = resolve(document, withEntry?.value);
      const structure = [
        jobsEntry?.value,
        rawJob,
        stepsEntry?.value,
        rawInvocation,
        withEntry?.value
      ];
      const mappingAlias = structure.find(isAlias);
      const workspace = reference?.startsWith("./") === true;
      const local =
        reference !== null && (workspace || reference.startsWith("$/"));
      const directory =
        local && reference !== null ? localPath(reference) : null;
      const target =
        reusable && local && reference !== null
          ? reusableDefaults(reference)
          : null;
      let action: ReadonlyMap<string, unknown> | null | undefined;
      let reason: string | null = null;
      if (!reusable && reference !== null) {
        if (!local) {
          action = approvedActions.get(reference);
        } else if (directory !== null) {
          const localManifest = manifest(directory);
          if (localManifest.kind === "ambiguous") {
            reason = "Directory has both action.yml and action.yaml";
          } else if (localManifest.kind === "available") {
            reason = workspace ? precedingRisk : null;
            if (reason === null) {
              action = localManifest.defaults;
            }
          }
        }
      }
      if (options.list === true) {
        if (reference !== null && !target && !action) {
          const use = structure.slice(0, -1).find(isAlias) ?? uses?.value;
          if (isNode(use) && use.range) {
            listEntry({
              kind: "left-to-review",
              ...location(file, lines, use.range[0]),
              reference,
              reason:
                reason ??
                (local
                  ? "Local input metadata is unavailable"
                  : "No approved metadata for this external reference")
            });
          }
        }
        continue;
      }
      if (!isMap(inputs) || inputs.items.length === 0) {
        continue;
      }

      const caller: Invocation = {
        reference,
        mappingAlias,
        path: [
          document.contents,
          jobsEntry?.key,
          jobKey,
          stepsEntry?.key,
          withEntry?.key,
          ...structure
        ],
        matchesDefault(name, input) {
          if (input === undefined) {
            return false;
          }
          if (target) {
            return matchesReusableDefault(target, name, input);
          }
          const value = action?.get(name.toLowerCase());
          return (
            typeof value === "string" &&
            !value.includes("${{") &&
            value === inputText(input)
          );
        }
      };
      const callers = consumers.get(inputs);
      if (callers) {
        callers.push(caller);
      } else {
        consumers.set(inputs, [caller]);
      }
    }
  }

  for (const [inputs, callers] of consumers) {
    const actions = [
      ...new Set(
        callers.map(({ reference }) => reference ?? "unresolved action")
      )
    ];
    const knownUses = new Set(
      callers.flatMap((caller) => caller.path.filter(isAlias))
    );
    for (const { key, value } of inputs.items) {
      const inputKey = resolve(document, key);
      const inputValue = resolve(document, value);
      const name = literal(inputKey);
      if (typeof name !== "string" || !isScalar(inputKey)) {
        continue;
      }
      const input = literal(inputValue);
      const matching = callers.filter(({ matchesDefault }) =>
        matchesDefault(name, input)
      );
      for (const caller of matching) {
        const { reference, mappingAlias } = caller;
        const inputAlias = [value, key].find(isAlias);
        const offset = (mappingAlias ?? inputAlias ?? inputKey).range?.[0];
        if (reference === null || offset === undefined) {
          continue;
        }
        const scalarIsShared =
          (isNode(inputValue) && aliases.has(inputValue)) ||
          aliases.has(inputKey);
        const anchors: Anchor[] = [];
        const nodes = new Set(
          [...caller.path, key, value].map((node) => resolve(document, node))
        );
        const outsideConsumer = [...nodes].some(
          (node) =>
            (isMap(node) || isSeq(node)) &&
            aliases.get(node)?.some((alias) => !knownUses.has(alias)) === true
        );
        let advice: Advice = "omit";
        if (mappingAlias !== undefined) {
          advice = "detach";
        } else if (
          inputAlias !== undefined ||
          scalarIsShared ||
          outsideConsumer ||
          matching.length !== callers.length
        ) {
          advice = "hand-edit";
        }
        for (const node of nodes) {
          if (!isNode(node) || isAlias(node) || node.anchor === undefined) {
            continue;
          }
          const uses = aliases.get(node);
          if (
            !uses &&
            (advice !== "omit" || (node !== inputKey && node !== inputValue))
          ) {
            continue;
          }
          const definitionOffset =
            (node.srcToken && anchorOffsets.get(node.srcToken)) ??
            node.range?.[0];
          if (definitionOffset !== undefined) {
            anchors.push({
              name: node.anchor,
              definition: location(file, lines, definitionOffset),
              uses: (uses ?? []).flatMap((alias) =>
                alias.range ? [location(file, lines, alias.range[0])] : []
              )
            });
          }
        }
        report({
          ...location(file, lines, offset),
          input: name,
          reference,
          advice,
          actions,
          anchors
        });
      }
    }
  }
}
