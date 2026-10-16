import assert from "node:assert/strict";
import { test } from "node:test";

import { lint } from "markdownlint/sync";

import noBoxDrawing from "./no-box-drawing.ts";

const violations = (input: string) =>
  (
    lint({
      config: { default: false, "docs/no-box-drawing": true },
      customRules: [noBoxDrawing],
      strings: { input }
    }).input ?? []
  ).map(({ lineNumber, errorRange }) => ({
    errorRange,
    lineNumber
  }));

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
