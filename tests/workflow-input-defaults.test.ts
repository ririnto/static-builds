import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import process from "node:process";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

await test("R35 reports only literals equal to declared defaults for the exact reference", () => {
  const checkout = mkdtempSync(path.join(tmpdir(), "workflow-input-defaults-"));
  try {
    const directory = path.join(checkout, ".github", "workflows");
    mkdirSync(directory, { recursive: true });
    const workflow = path.join(directory, "caller.yaml");
    const controls = `jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 1
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0
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
  shared:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with: &checkout-inputs
          fetch-depth: 1
      - uses: actions/checkout@v6
        with: *checkout-inputs
`;
    writeFileSync(
      path.join(directory, "reusable.yml"),
      `on:
  workflow_call:
    inputs:
      publish:
        type: boolean
        default: true
`
    );
    const check = () =>
      spawnSync(
        process.execPath,
        [
          fileURLToPath(
            new URL("../rules/workflow-input-defaults.ts", import.meta.url)
          )
        ],
        { cwd: checkout, encoding: "utf-8" }
      );
    writeFileSync(workflow, controls);
    const passing = check();
    assert.equal(passing.status, 0, passing.stderr);
    assert.equal(passing.stderr, "");
    writeFileSync(
      workflow,
      controls
        .replace("fetch-depth: 0", "fetch-depth: 1")
        .replace("publish: false", "publish: true")
    );
    const failing = check();
    assert.equal(failing.status, 1, failing.stderr);
    const diagnostics = failing.stderr.trim().split("\n");
    assert.equal(diagnostics.length, 2, failing.stderr);
    assert.match(
      diagnostics[0] ?? "",
      /caller\.yaml:10:11: R35\/workflow-input-defaults: Omit "fetch-depth" for "actions\/checkout@v7"/u
    );
    assert.match(
      diagnostics[1] ?? "",
      /caller\.yaml:21:7: R35\/workflow-input-defaults: Omit "publish" for "\.\/\.github\/workflows\/reusable\.yml"/u
    );
  } finally {
    rmSync(checkout, { force: true, recursive: true });
  }
});
