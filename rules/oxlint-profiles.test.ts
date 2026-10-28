import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));

const check = (profile: string, source: string): string => {
  const directory = mkdtempSync(path.join(tmpdir(), "lint-fixture-"));
  try {
    const input = path.join(directory, "fixture.ts");
    writeFileSync(input, source);
    const result = spawnSync(
      path.join(root, "node_modules/.bin/ultracite"),
      ["check", "--config", path.join(root, profile), input],
      { cwd: root, encoding: "utf-8" }
    );
    assert.equal(result.error, undefined);
    return result.stdout + result.stderr;
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

test("base profile requires multiline documentation for exported declarations", () => {
  const invalid = check("oxlint.config.ts", "export const value = 1;\n");
  assert.match(invalid, /tsdoc\(require-export-tsdoc\)/u);
  const valid = check(
    "oxlint.config.ts",
    "/**\n * Provide the value.\n */\nexport const value = 1;\n"
  );
  assert.doesNotMatch(valid, /tsdoc\(require-export-tsdoc\)/u);
});
