import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { stripVTControlCharacters } from "node:util";

const root = fileURLToPath(new URL("../", import.meta.url));

const check = (source: string): string[] => {
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
      [
        "check",
        "--format",
        "json",
        "--config",
        path.join(root, "oxlint.config.ts"),
        input
      ],
      { cwd: root, encoding: "utf-8", env }
    );
    assert.equal(result.error, undefined);
    assert.equal(result.signal, null);
    assert.ok(result.status === 0 || result.status === 1);
    assert.equal(result.stderr.trim(), "");
    const output = stripVTControlCharacters(result.stdout);
    const jsonStart = output.indexOf('{ "diagnostics"');
    assert.ok(jsonStart !== -1, output);
    const formatter = output.slice(0, jsonStart);
    assert.match(formatter, /^Checking formatting\.\.\./u);
    assert.match(formatter, /Finished in .* on 1 files[^\n]*\s*$/u);
    const parsed: unknown = JSON.parse(output.slice(jsonStart));
    assert.ok(typeof parsed === "object" && parsed !== null);
    assert.ok("number_of_files" in parsed && parsed.number_of_files === 1);
    assert.ok("diagnostics" in parsed && Array.isArray(parsed.diagnostics));
    return parsed.diagnostics.map((diagnostic: unknown) => {
      assert.ok(typeof diagnostic === "object" && diagnostic !== null);
      assert.ok("filename" in diagnostic && diagnostic.filename === input);
      assert.ok("code" in diagnostic && typeof diagnostic.code === "string");
      return diagnostic.code;
    });
  } finally {
    rmSync(directory, { recursive: true });
  }
};

void test("blank-line profile detects gaps but preserves directives and literal contents", () => {
  const invalid = check("const f = () => {\n\n  return 1;\n};\nf();\n");
  assert.ok(invalid.includes("style-blank-lines(no-blank-lines-in-functions)"));
  const valid = check(
    "const f = () => {\n  'use strict';\n\n  const value = `first\n\nlast`;\n  /* SPDX-License-Identifier: MIT\n\n   * Modification notice. */\n  return value;\n};\nf();\n"
  );
  assert.ok(!valid.includes("style-blank-lines(no-blank-lines-in-functions)"));
});

void test("sibling profile checks proven test registrations without misclassifying local functions", () => {
  const invalid = check(
    'import { test } from "vitest";\ntest("one", () => {});\ntest("two", () => {});\n'
  );
  assert.ok(invalid.includes("style-spacing(sibling-test-spacing)"));
  const valid = check(
    'import { test } from "vitest";\ntest("one", () => {});\n\ntest("two", () => {});\n'
  );
  assert.ok(!valid.includes("style-spacing(sibling-test-spacing)"));
  const local = check(
    'const test = (value: string) => value;\ntest("one");\ntest("two");\n'
  );
  assert.ok(!local.includes("style-spacing(sibling-test-spacing)"));
});
