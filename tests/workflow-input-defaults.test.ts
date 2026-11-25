import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";

const check = (checkout: string, ...options: string[]) =>
  spawnSync(
    "pnpm",
    [
      "--silent",
      "run",
      "check:workflow-input-defaults",
      "--repository",
      checkout,
      ...options
    ],
    { cwd: new URL("../", import.meta.url), encoding: "utf-8" }
  );

await test("R35 reports only literals equal to declared defaults for the exact reference", () => {
  const checkout = mkdtempSync(path.join(tmpdir(), "workflow-input-defaults-"));
  try {
    const directory = path.join(checkout, ".github", "workflows");
    mkdirSync(directory, { recursive: true });
    const workflow = path.join(directory, "caller.yaml");
    const controls = `name: Caller
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 1
      - uses: actions/checkout@v7
        with: &unshared
          FETCH-DEPTH: &unused 0
          filter: "null"
          clean: \${{ true }}
      - uses: actions/upload-artifact@v7
        with:
          retention-days: 0
      - uses: actions/setup-go@v7
        with:
          cache-dependency-path: go.mod
      - uses: example/action@v1
        with:
          fetch-depth: 1
  release:
    uses: ./.github/workflows/reusable.yml
    with:
      publish: false
      fraction: 0.25
      implicit-flag: true
      implicit-count: 1
      implicit-text: custom
      midpoint: 100000000000001
      required-text: ""
  actions:
    runs-on: ubuntu-latest
    steps:
      - uses: aquasecurity/trivy-action@v0.36.0
        with:
          exit-code: 0
          hide-progress: false
      - uses: actions/download-artifact@v8
        with:
          digest-mismatch: ignore
      - uses: $/.github/actions/alternate
        with:
          local-label: custom
      - uses: $/.github/actions/fixture
        with:
          scientific-midpoint: 1234567890123445
  safe-local:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - run: echo Ready
      - uses: ./.github/actions/fixture
        with:
          LOCAL-RATE: 0.00002
          local-null: custom
          local-optional: ""
          midpoint: 100000000000000.5
  other-checkout:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          repository: example/other
          ref: older
      - uses: ./.github/actions/fixture
        with:
          LOCAL-RATE: "1E-05"
      - uses: $/.github/actions/fixture@old
        with:
          LOCAL-RATE: "1E-05"
      - uses: $/.github/actions/missing
  invalid-call:
    uses: $/.github/workflows/reusable.yml@v1
    with:
      publish: true
`;
    writeFileSync(
      path.join(directory, "reusable.yml"),
      `name: Reusable
on:
  workflow_call:
    inputs:
      publish:
        type: boolean
        default: true
      fraction:
        type: number
        default: 0.5
      midpoint:
        type: number
        default: 100000000000000.5
      implicit-flag:
        type: boolean
      implicit-count:
        type: number
      implicit-text:
        type: string
      required-text:
        required: true
        type: string
jobs:
  run:
    runs-on: ubuntu-latest
    steps:
      - run: echo Ready
`
    );
    const actions = path.join(checkout, ".github", "actions");
    mkdirSync(path.join(actions, "fixture"), { recursive: true });
    mkdirSync(path.join(actions, "alternate"), { recursive: true });
    const manifest = `name: Fixture
description: Exercise local action defaults
inputs:
  local-rate:
    description: Rate
    default: 1e-5
  local-null:
    description: Empty value
    default: null
  local-optional:
    description: Optional value
  midpoint:
    description: Fixed-point midpoint
    default: "100000000000001"
  scientific-midpoint:
    description: Scientific midpoint
    default: "1.23456789012345E+15"
runs:
  using: composite
  steps:
    - run: echo Ready
      shell: sh
`;
    writeFileSync(path.join(actions, "fixture", "action.yml"), manifest);
    writeFileSync(
      path.join(actions, "alternate", "action.yaml"),
      manifest
        .replace("local-rate:", "local-label:")
        .replace("default: 1e-5", "default: standard")
    );
    writeFileSync(workflow, controls);
    const passing = check(checkout, "--json");
    assert.equal(passing.status, 0, passing.stderr);
    assert.equal(passing.stdout.trim(), "");
    writeFileSync(
      workflow,
      `${controls
        .replace("FETCH-DEPTH: &unused 0", "FETCH-DEPTH: &unused 1")
        .replace('filter: "null"', "filter: null")
        .replace("publish: false", "publish: true")
        .replace("fraction: 0.25", "fraction: 0.5")
        .replace("implicit-flag: true", "implicit-flag: false")
        .replace("implicit-count: 1", "implicit-count: 0")
        .replace("implicit-text: custom", 'implicit-text: ""')
        .replace("midpoint: 100000000000001", "midpoint: 100000000000000")
        .replace("digest-mismatch: ignore", "digest-mismatch: error")
        .replace("LOCAL-RATE: 0.00002", 'LOCAL-RATE: "1E-05"')
        .replace("local-null: custom", "local-null: null")
        .replace(
          "local-label: custom",
          "local-label: standard"
        )}  anchor-source:
    runs-on: ubuntu-latest
    steps: &sequence
      - &step
        &uses-key uses: &reference actions/checkout@v7
        &with-key with: &inputs
          fetch-depth: &depth 1
  alias-step:
    runs-on: ubuntu-latest
    steps:
      - *step
      - ? *uses-key
        : *reference
        ? *with-key
        : *inputs
      - uses: *reference
        with:
          fetch-depth: *depth
  alias-sequence:
    runs-on: ubuntu-latest
    steps: *sequence
`
    );
    const failing = check(checkout, "--json");
    assert.equal(failing.status, 1, failing.stderr);
    const diagnostics = failing.stdout
      .trim()
      .split("\n")
      .map((line) => {
        const record: unknown = JSON.parse(line);
        assert.ok(
          record !== null &&
            typeof record === "object" &&
            "code" in record &&
            "file" in record &&
            "input" in record &&
            "reference" in record &&
            "line" in record &&
            "column" in record
        );
        const { code, file, input, reference, line: row, column } = record;
        return JSON.stringify([code, file, input, reference, row, column]);
      });
    const expected = [
      ["FETCH-DEPTH", "actions/checkout@v7", 12, 11],
      ["filter", "actions/checkout@v7", 13, 11],
      ["publish", "./.github/workflows/reusable.yml", 27, 7],
      ["fraction", "./.github/workflows/reusable.yml", 28, 7],
      ["implicit-flag", "./.github/workflows/reusable.yml", 29, 7],
      ["implicit-count", "./.github/workflows/reusable.yml", 30, 7],
      ["implicit-text", "./.github/workflows/reusable.yml", 31, 7],
      ["midpoint", "./.github/workflows/reusable.yml", 32, 7],
      ["digest-mismatch", "actions/download-artifact@v8", 43, 11],
      ["local-label", "$/.github/actions/alternate", 46, 11],
      ["LOCAL-RATE", "./.github/actions/fixture", 57, 11],
      ["local-null", "./.github/actions/fixture", 58, 11],
      ["fetch-depth", "actions/checkout@v7", 85, 11],
      ["fetch-depth", "actions/checkout@v7", 89, 9],
      ["fetch-depth", "actions/checkout@v7", 93, 11],
      ["fetch-depth", "actions/checkout@v7", 96, 24],
      ["fetch-depth", "actions/checkout@v7", 99, 12]
    ];
    assert.equal(diagnostics.length, expected.length);
    assert.deepEqual(
      new Set(diagnostics),
      new Set(
        expected.map((row) =>
          JSON.stringify([
            "R35/workflow-input-defaults",
            ".github/workflows/caller.yaml",
            ...row
          ])
        )
      )
    );
  } finally {
    rmSync(checkout, { force: true, recursive: true });
  }
});

const recordLines = (output: string) =>
  output
    .trim()
    .split("\n")
    .filter(Boolean)
    .map((line) => {
      const entry: unknown = JSON.parse(line);
      assert.ok(entry !== null && typeof entry === "object");
      return entry;
    });

await test("R35 traces shared YAML definitions and advises edits at the caller", () => {
  const checkout = mkdtempSync(path.join(tmpdir(), "workflow-input-aliases-"));
  try {
    const directory = path.join(checkout, ".github", "workflows");
    mkdirSync(directory, { recursive: true });
    const file = ".github/workflows/aliases.yaml";
    writeFileSync(
      path.join(checkout, file),
      `jobs:
  source:
    steps: &steps
      - &step
        &use uses: &checkout actions/checkout@v7
        &params with: &shared
          &key clean: &value true
      - uses: actions/setup-go@v7
        with: &equal
          cache: &unused true
  callers:
    steps:
      - *step
      - ? *use
        : *checkout
        ? *params
        : *shared
      - uses: actions/setup-go@v7
        with: *shared
      - uses: actions/checkout@v7
        with:
          ? *key
          : *value
      - uses: *checkout
        with:
          clean: *value
      - uses: aquasecurity/trivy-action@v0.36.0
        with: *equal
  sequence:
    steps: *steps
  nested:
    steps:
      - uses: actions/checkout@v7
        with:
          &credentials-key persist-credentials: &enabled true
      - uses: actions/checkout@v7
        with: &nested
          clean: *enabled
          ? *credentials-key
          : *enabled
      - uses: actions/checkout@v7
        with: *nested
`
    );
    const result = check(checkout, "--json");
    assert.equal(result.status, 1, result.stderr);
    const records = recordLines(result.stdout);
    const at = (
      line: number,
      input: string,
      column: number,
      reference = "actions/checkout@v7"
    ) => {
      const record = records.find(
        (entry) =>
          "line" in entry &&
          entry.line === line &&
          "input" in entry &&
          entry.input === input
      );
      assert.ok(record);
      assert.ok("file" in record && record.file === file);
      assert.ok("reference" in record && record.reference === reference);
      assert.ok("column" in record && record.column === column);
      assert.ok(
        "advice" in record && "actions" in record && "anchors" in record
      );
      return record;
    };
    const definition = at(7, "clean", 16);
    assert.equal(definition.advice, "hand-edit");
    assert.deepEqual(
      new Set(Array.isArray(definition.actions) ? definition.actions : []),
      new Set(["actions/checkout@v7", "actions/setup-go@v7"])
    );
    assert.ok(Array.isArray(definition.anchors));
    assert.deepEqual(
      new Set(
        definition.anchors.map((entry: unknown) => {
          assert.ok(
            entry !== null &&
              typeof entry === "object" &&
              "name" in entry &&
              "definition" in entry &&
              "uses" in entry
          );
          assert.ok(
            entry.definition !== null &&
              typeof entry.definition === "object" &&
              "file" in entry.definition &&
              entry.definition.file === file &&
              "line" in entry.definition
          );
          assert.ok(Array.isArray(entry.uses));
          return JSON.stringify([
            entry.name,
            entry.definition.line,
            entry.uses
              .map((use: unknown) => {
                assert.ok(
                  use !== null &&
                    typeof use === "object" &&
                    "file" in use &&
                    use.file === file &&
                    "line" in use &&
                    typeof use.line === "number"
                );
                return use.line;
              })
              .toSorted((a, b) => a - b)
          ]);
        })
      ),
      new Set(
        [
          ["steps", 3, [30]],
          ["step", 4, [13]],
          ["params", 6, [16]],
          ["shared", 6, [17, 19]],
          ["key", 7, [22]],
          ["value", 7, [23, 26]]
        ].map((entry) => JSON.stringify(entry))
      )
    );
    assert.equal(at(13, "clean", 9).advice, "detach");
    assert.equal(at(17, "clean", 11).advice, "detach");
    assert.equal(at(23, "clean", 13).advice, "hand-edit");
    assert.equal(at(26, "clean", 18).advice, "hand-edit");
    assert.equal(at(30, "clean", 12).advice, "detach");
    assert.partialDeepStrictEqual(at(10, "cache", 11, "actions/setup-go@v7"), {
      advice: "omit",
      anchors: [{ name: "unused", uses: [] }]
    });
    assert.equal(
      at(28, "cache", 15, "aquasecurity/trivy-action@v0.36.0").advice,
      "detach"
    );
    assert.equal(at(30, "cache", 12, "actions/setup-go@v7").advice, "detach");
    assert.deepEqual(
      {
        clean: at(42, "clean", 15).advice,
        "persist-credentials": at(42, "persist-credentials", 15).advice
      },
      { clean: "hand-edit", "persist-credentials": "hand-edit" }
    );
  } finally {
    rmSync(checkout, { force: true, recursive: true });
  }
});

await test("R35 lists unavailable action metadata without diagnosing the calls", () => {
  const checkout = mkdtempSync(
    path.join(tmpdir(), "workflow-input-unavailable-")
  );
  try {
    const directory = path.join(checkout, ".github", "workflows");
    mkdirSync(directory, { recursive: true });
    const file = ".github/workflows/unavailable.yaml";
    writeFileSync(
      path.join(checkout, file),
      `jobs:
  external:
    runs-on: ubuntu-latest
    steps:
      - uses: example/action@v1
        with:
          label: standard
  local:
    runs-on: ubuntu-latest
    steps:
      - uses: ./unreadable
        with:
          label: standard
`
    );
    mkdirSync(path.join(checkout, "unreadable"));
    writeFileSync(
      path.join(checkout, "unreadable", "action.yaml"),
      "inputs: [\n"
    );
    const result = check(checkout, "--json");
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout.trim(), "");
    assert.equal(result.stderr.trim(), "");
    const listing = check(checkout, "--list");
    assert.equal(listing.status, 0, listing.stderr);
    assert.deepEqual(
      new Set(
        recordLines(listing.stdout)
          .filter((entry) => "kind" in entry && entry.kind === "left-to-review")
          .map((entry) => {
            assert.ok(
              "file" in entry && "line" in entry && "reference" in entry
            );
            return JSON.stringify([entry.file, entry.line, entry.reference]);
          })
      ),
      new Set(
        [
          [file, 5, "example/action@v1"],
          [file, 11, "./unreadable"]
        ].map((entry) => JSON.stringify(entry))
      )
    );
  } finally {
    rmSync(checkout, { force: true, recursive: true });
  }
});

await test("R35 lists contextual defaults and local calls left to review", async (t) => {
  const checkout = mkdtempSync(path.join(tmpdir(), "workflow-input-review-"));
  try {
    const directory = path.join(checkout, ".github", "workflows");
    mkdirSync(directory, { recursive: true });
    const file = ".github/workflows/local.yaml";
    writeFileSync(
      path.join(checkout, file),
      `jobs:
  checkout:
    steps:
      - uses: actions/checkout@v7
        with:
          repository: example/other
      - uses: ./local
        with:
          label: standard
  ref:
    steps:
      - uses: actions/checkout@v7
        with:
          ref: other
      - uses: ./local
        with:
          label: standard
  write:
    steps:
      - run: echo replacement > local/action.yaml
      - uses: ./local
        with:
          label: standard
  script:
    steps:
      - run: ./unreadable.sh
      - uses: ./local
        with:
          label: standard
  dual:
    steps:
      - uses: ./dual
        with:
          label: standard
`
    );
    const manifest = "inputs:\n  label:\n    default: standard\n";
    mkdirSync(path.join(checkout, "local"));
    writeFileSync(path.join(checkout, "local", "action.yaml"), manifest);
    mkdirSync(path.join(checkout, "dual"));
    writeFileSync(path.join(checkout, "dual", "action.yml"), manifest);
    writeFileSync(path.join(checkout, "dual", "action.yaml"), manifest);
    const result = check(checkout, "--json");
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout.trim(), "");
    const listing = check(checkout, "--list");
    assert.equal(listing.status, 0, listing.stderr);
    const entries = recordLines(listing.stdout);
    const contextual = entries.filter(
      (entry) => "kind" in entry && entry.kind === "context-dependent"
    );
    const reviews = entries.filter(
      (entry) => "kind" in entry && entry.kind === "left-to-review"
    );
    assert.equal(contextual.length, 23);
    assert.equal(reviews.length, 5);
    assert.ok(
      contextual.some(
        (entry) =>
          "reference" in entry &&
          entry.reference === "actions/setup-go@v7" &&
          "input" in entry &&
          entry.input === "cache-dependency-path" &&
          "condition" in entry &&
          entry.condition ===
            "With caching enabled, falls back to go.mod at the GitHub workspace root"
      )
    );
    assert.deepEqual(
      reviews
        .filter(
          (entry) => "reference" in entry && entry.reference === "./local"
        )
        .map((entry) => {
          assert.ok(
            "file" in entry &&
              entry.file === file &&
              "line" in entry &&
              "reference" in entry
          );
          return JSON.stringify([entry.line, entry.reference]);
        })
        .toSorted(),
      [
        [7, "./local"],
        [15, "./local"],
        [21, "./local"],
        [27, "./local"]
      ]
        .map((entry) => JSON.stringify(entry))
        .toSorted()
    );
    await t.test("a directory with both manifests is left to review", () => {
      assert.ok(
        reviews.some(
          (entry) =>
            "file" in entry &&
            entry.file === file &&
            "line" in entry &&
            entry.line === 32 &&
            "reference" in entry &&
            entry.reference === "./dual"
        )
      );
    });
  } finally {
    rmSync(checkout, { force: true, recursive: true });
  }
});

await test("R35 continues after documents without invocations", async (t) => {
  const checkout = mkdtempSync(path.join(tmpdir(), "workflow-input-empty-"));
  try {
    const directory = path.join(checkout, ".github", "workflows");
    mkdirSync(directory, { recursive: true });
    const documents = [
      ["empty", ""],
      ["comments-only", "# no workflow\n"],
      ["no-jobs", "name: No jobs\n"],
      ["no-steps", "jobs:\n  absent:\n    runs-on: ubuntu-latest\n"]
    ];
    for (const [name, contents] of documents) {
      assert.ok(name !== undefined && contents !== undefined);
      writeFileSync(path.join(directory, `${name}.yaml`), contents);
    }
    writeFileSync(
      path.join(directory, "z-control.yaml"),
      "jobs:\n  check:\n    steps:\n      - uses: actions/setup-go@v7\n        with:\n          cache: true\n"
    );
    const result = check(checkout, "--json");
    assert.equal(result.status, 1, result.stderr);
    const entries = recordLines(result.stdout);
    await Promise.all(
      documents.map(async ([name]) => {
        await t.test(`${name} produces no diagnostic`, () => {
          assert.ok(
            !entries.some(
              (entry) =>
                "file" in entry &&
                entry.file === `.github/workflows/${name}.yaml`
            )
          );
        });
      })
    );
    assert.equal(entries.length, 1);
    const [entry] = entries;
    assert.ok(
      "file" in entry &&
        entry.file === ".github/workflows/z-control.yaml" &&
        "input" in entry &&
        entry.input === "cache" &&
        "reference" in entry &&
        entry.reference === "actions/setup-go@v7" &&
        "line" in entry &&
        entry.line === 6
    );
  } finally {
    rmSync(checkout, { force: true, recursive: true });
  }
});
