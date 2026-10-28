import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { stripVTControlCharacters } from "node:util";

const root = fileURLToPath(new URL("../", import.meta.url));

const check = (profile: string, source: string): string => {
  const directory = mkdtempSync(path.join(tmpdir(), "lint-fixture-"));
  try {
    const input = path.join(directory, "fixture.ts");
    writeFileSync(input, source);
    const env: NodeJS.ProcessEnv = {
      ...process.env,
      FORCE_COLOR: "0",
      NO_COLOR: "1"
    };
    delete env.GITHUB_ACTIONS;
    const result = spawnSync(
      path.join(root, "node_modules/.bin/ultracite"),
      ["check", "--config", path.join(root, profile), input],
      { cwd: root, encoding: "utf-8", env }
    );
    assert.equal(result.error, undefined);
    assert.equal(result.signal, null);
    assert.ok(result.status === 0 || result.status === 1);
    const output = stripVTControlCharacters(result.stdout + result.stderr);
    assert.match(output, /Finished in .* on 1 files/u);
    for (const line of output.split("\n").filter(Boolean)) {
      assert.ok(
        line === "Checking formatting..." ||
          line === "All matched files use the correct format." ||
          line ===
            "Format issues found in above 1 files. Run without `--check` to fix." ||
          line.startsWith("Finished in ") ||
          (line.startsWith(input) &&
            (/ \(\d+ms\)$/u.test(line) ||
              /:\d+:\d+: (?:error|warning) [\w/-]+\([^)]+\):/u.test(line))),
        `Unexpected Ultracite output: ${line}`
      );
    }
    return output;
  } finally {
    rmSync(directory, { recursive: true });
  }
};

test("blank-line profile detects gaps but preserves directives and literal contents", () => {
  const profile = "oxlint.style-no-blank-lines-in-functions.config.ts";
  const invalid = check(
    profile,
    "const f = () => {\n\n  return 1;\n};\nf();\n"
  );
  assert.match(invalid, /style\(no-blank-lines-in-functions\)/u);
  const valid = check(
    profile,
    "const f = () => {\n  'use strict';\n\n  const value = `first\n\nlast`;\n  /* SPDX-License-Identifier: MIT\n\n   * Modification notice. */\n  return value;\n};\nf();\n"
  );
  assert.doesNotMatch(valid, /style\(no-blank-lines-in-functions\)/u);
});

test("sibling profile checks proven test registrations without misclassifying local functions", () => {
  const profile = "oxlint.style-sibling-test-spacing.config.ts";
  const invalid = check(
    profile,
    'import { test } from "vitest";\ntest("one", () => {});\ntest("two", () => {});\n'
  );
  assert.match(invalid, /style\(sibling-test-spacing\)/u);
  const valid = check(
    profile,
    'import { test } from "vitest";\ntest("one", () => {});\n\ntest("two", () => {});\n'
  );
  assert.doesNotMatch(valid, /style\(sibling-test-spacing\)/u);
  const local = check(
    profile,
    'const test = (value: string) => value;\ntest("one");\ntest("two");\n'
  );
  assert.doesNotMatch(local, /style\(sibling-test-spacing\)/u);
});
