import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  copyFileSync,
  mkdirSync,
  mkdtempSync,
  rmSync,
  symlinkSync,
  writeFileSync
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

await test("R35 reports only literals equal to declared defaults for the exact reference", () => {
  const checkout = mkdtempSync(path.join(tmpdir(), "workflow-input-defaults-"));
  try {
    const directory = path.join(checkout, ".github", "workflows");
    mkdirSync(directory, { recursive: true });
    mkdirSync(path.join(checkout, "rules"));
    for (const file of [
      "package.json",
      "rules/workflow-input-defaults.ts",
      ".github/action-inputs.json"
    ]) {
      copyFileSync(
        new URL(`../${file}`, import.meta.url),
        path.join(checkout, file)
      );
    }
    symlinkSync(
      fileURLToPath(new URL("../node_modules", import.meta.url)),
      path.join(checkout, "node_modules"),
      "junction"
    );
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
        with:
          FETCH-DEPTH: 0
          filter: "null"
          clean: \${{ true }}
      - uses: actions/upload-artifact@v7
        with:
          retention-days: 0
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
      required-text: ""
  shared:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/checkout@v7
        with: &checkout-inputs
          fetch-depth: 1
      - uses: actions/checkout@v6
        with: *checkout-inputs
      - uses: aquasecurity/trivy-action@v0.36.0
        with:
          exit-code: 0
          hide-progress: false
      - uses: actions/download-artifact@v8
        with:
          digest-mismatch: ignore
      - uses: ./.github/actions/fixture
        with:
          LOCAL-RATE: 0.00002
          local-null: custom
          local-optional: ""
      - uses: $/.github/actions/alternate
        with:
          local-label: custom
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
      spawnSync("pnpm", ["run", "check:workflow-input-defaults"], {
        cwd: checkout,
        encoding: "utf-8"
      });
    writeFileSync(workflow, controls);
    const passing = check();
    assert.equal(passing.status, 0, passing.stderr);
    assert.doesNotMatch(passing.stderr, /R35\//u);
    writeFileSync(
      workflow,
      controls
        .replace("FETCH-DEPTH: 0", "FETCH-DEPTH: 1")
        .replace('filter: "null"', "filter: null")
        .replace("publish: false", "publish: true")
        .replace("fraction: 0.25", "fraction: 0.5")
        .replace("implicit-flag: true", "implicit-flag: false")
        .replace("implicit-count: 1", "implicit-count: 0")
        .replace("implicit-text: custom", 'implicit-text: ""')
        .replace("digest-mismatch: ignore", "digest-mismatch: error")
        .replace("LOCAL-RATE: 0.00002", 'LOCAL-RATE: "1E-05"')
        .replace("local-null: custom", "local-null: null")
        .replace("local-label: custom", "local-label: standard")
        .replace(
          "uses: ./.github/workflows/reusable.yml",
          "uses: $/.github/workflows/reusable.yml"
        )
    );
    const failing = check();
    assert.equal(failing.status, 1, failing.stderr);
    assert.deepEqual(
      [
        ...failing.stderr.matchAll(
          /R35\/workflow-input-defaults:[^"\n]*"(?<input>[^"\n]+)"[^"\n]*"(?<reference>[^"\n]+)"/gu
        )
      ]
        .map(({ groups }) => [groups?.input, groups?.reference].join(" "))
        .toSorted(),
      [
        "FETCH-DEPTH actions/checkout@v7",
        "filter actions/checkout@v7",
        "publish $/.github/workflows/reusable.yml",
        "fraction $/.github/workflows/reusable.yml",
        "implicit-flag $/.github/workflows/reusable.yml",
        "implicit-count $/.github/workflows/reusable.yml",
        "implicit-text $/.github/workflows/reusable.yml",
        "digest-mismatch actions/download-artifact@v8",
        "LOCAL-RATE ./.github/actions/fixture",
        "local-null ./.github/actions/fixture",
        "local-label $/.github/actions/alternate"
      ].toSorted(),
      failing.stderr
    );
    assert.equal(
      [...failing.stderr.matchAll(/R35\//gu)].length,
      11,
      failing.stderr
    );
    writeFileSync(
      workflow,
      `on: push
jobs:
  invalid-workflow:
    uses: $/.github/workflows/reusable.yml@v1
  invalid-action:
    runs-on: ubuntu-latest
    steps:
      - uses: $/.github/actions/missing
`
    );
    const invalid = check();
    assert.equal(invalid.status, 1, invalid.stderr);
    assert.deepEqual(
      [
        ...invalid.stderr.matchAll(
          /R35\/invalid-self-reference:[^"\n]*"(?<reference>[^"\n]+)"/gu
        )
      ]
        .map(({ groups }) => groups?.reference ?? "")
        .toSorted(),
      ["$/.github/actions/missing", "$/.github/workflows/reusable.yml@v1"],
      invalid.stderr
    );
    assert.equal(
      [...invalid.stderr.matchAll(/R35\//gu)].length,
      2,
      invalid.stderr
    );
  } finally {
    rmSync(checkout, { force: true, recursive: true });
  }
});
