import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";

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
      path.join(actions, "fixture", "action.yaml"),
      manifest.replace("default: 1e-5", "default: 0.00002")
    );
    writeFileSync(
      path.join(actions, "alternate", "action.yaml"),
      manifest
        .replace("local-rate:", "local-label:")
        .replace("default: 1e-5", "default: standard")
    );
    const check = () =>
      spawnSync(
        "pnpm",
        [
          "--silent",
          "run",
          "check:workflow-input-defaults",
          "--repository",
          checkout,
          "--json"
        ],
        { cwd: new URL("../", import.meta.url), encoding: "utf-8" }
      );
    writeFileSync(workflow, controls);
    const passing = check();
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
    const failing = check();
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
      ["LOCAL-RATE", "./.github/actions/fixture", 55, 11],
      ["local-null", "./.github/actions/fixture", 56, 11],
      ["fetch-depth", "actions/checkout@v7", 83, 11],
      ["fetch-depth", "actions/checkout@v7", 87, 9],
      ["fetch-depth", "actions/checkout@v7", 91, 11],
      ["fetch-depth", "actions/checkout@v7", 94, 24],
      ["fetch-depth", "actions/checkout@v7", 97, 12]
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
