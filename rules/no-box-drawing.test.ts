import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { test } from "node:test";

import MarkdownIt from "markdown-it";
import type { Configuration, Rule } from "markdownlint";
import { applyFixes } from "markdownlint";
import { lint } from "markdownlint/sync";

const { config }: { config: Configuration } = createRequire(import.meta.url)(
  "markdownlint-cli2/parsers/jsonc"
).default(
  readFileSync(new URL("../.markdownlint-cli2.jsonc", import.meta.url), "utf-8")
);
const searchReplace: Rule = createRequire(import.meta.url)(
  "markdownlint-rule-search-replace"
);
const lintOptions = {
  config,
  customRules: [searchReplace],
  markdownItFactory: () => new MarkdownIt()
};

const violations = (input: string) =>
  (lint({ ...lintOptions, strings: { input } }).input ?? [])
    .filter(({ ruleNames }) => ruleNames.includes("search-replace"))
    .map(({ lineNumber, errorRange }) => ({ errorRange, lineNumber }));

test("accepts ASCII tree markers and characters outside the rule range", () => {
  assert.deepEqual(
    violations(
      `ASCII +-- and | ${String.fromCodePoint(0x24_ff)} ${String.fromCodePoint(0x25_80)}`
    ),
    []
  );
});

test("reports every character in the Unicode box drawing range", () => {
  assert.deepEqual(
    violations(
      Array.from(
        { length: 0x80 },
        (_, index) => `x${String.fromCodePoint(0x25_00 + index)}y`
      ).join("\n")
    ),
    Array.from({ length: 0x80 }, (_, index) => ({
      errorRange: [2, 1],
      lineNumber: index + 1
    }))
  );
});

test("reports the first box drawing character on a line", () => {
  assert.deepEqual(
    violations(
      `${String.fromCodePoint(0x25_00)} and ${String.fromCodePoint(0x25_7f)}`
    ),
    [{ errorRange: [1, 1], lineNumber: 1 }]
  );
});

test("reports the location inside fenced code", () => {
  assert.deepEqual(
    violations(
      ["```text", `x ${String.fromCodePoint(0x25_02)} y`, "```"].join("\n")
    ),
    [{ errorRange: [3, 1], lineNumber: 2 }]
  );
});

test("reports matching columns and line numbers across multiline Markdown", () => {
  assert.deepEqual(
    violations(
      `plain +--\n  a${String.fromCodePoint(0x25_01)}\n| safe\n  b${String.fromCodePoint(0x25_7f)}`
    ),
    [
      { errorRange: [4, 1], lineNumber: 2 },
      { errorRange: [4, 1], lineNumber: 4 }
    ]
  );
});

test("reports UTF-16 columns after astral characters", () => {
  assert.deepEqual(
    violations(
      `${String.fromCodePoint(0x1_f6_00)} ${String.fromCodePoint(0x25_00)}`
    ),
    [{ errorRange: [4, 1], lineNumber: 1 }]
  );
});

test("leaves box drawing content unchanged when applying fixes", () => {
  const input = `# Diagnostic only\n\nA box ${String.fromCodePoint(0x25_00)} remains.\n`;
  assert.equal(
    applyFixes(input, lint({ ...lintOptions, strings: { input } }).input ?? []),
    input
  );
});
