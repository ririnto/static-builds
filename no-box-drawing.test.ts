import assert from "node:assert/strict";
import { test } from "node:test";

import { lint } from "markdownlint/sync";

import noBoxDrawing from "./no-box-drawing.ts";

const violations = (input: string) => {
  const result = lint({
    config: { default: false, "docs/no-box-drawing": true },
    customRules: [noBoxDrawing],
    strings: { input }
  });
  return (result.input ?? []).map(({ lineNumber, errorRange }) => ({
    errorRange,
    lineNumber
  }));
};

test("accepts ASCII tree markers and characters outside the rule range", () => {
  const beforeRange = String.fromCodePoint(0x24_ff);
  const afterRange = String.fromCodePoint(0x25_80);
  assert.deepEqual(
    violations(`ASCII +-- and | ${beforeRange} ${afterRange}`),
    []
  );
});

test("reports every character in the Unicode box drawing range", () => {
  const codePoints = Array.from({ length: 0x80 }, (_, index) =>
    String.fromCodePoint(0x25_00 + index)
  );
  const input = codePoints.map((character) => `x${character}y`).join("\n");
  const expected = codePoints.map((_, index) => ({
    errorRange: [2, 1],
    lineNumber: index + 1
  }));

  assert.deepEqual(violations(input), expected);
});

test("reports the first box drawing character on a line", () => {
  const first = String.fromCodePoint(0x25_00);
  const second = String.fromCodePoint(0x25_7f);
  assert.deepEqual(violations(`${first} and ${second}`), [
    { errorRange: [1, 1], lineNumber: 1 }
  ]);
});

test("reports the location inside fenced code", () => {
  const boxCharacter = String.fromCodePoint(0x25_02);
  const input = ["```text", `x ${boxCharacter} y`, "```"].join("\n");
  assert.deepEqual(violations(input), [{ errorRange: [3, 1], lineNumber: 2 }]);
});

test("reports matching columns and line numbers across multiline Markdown", () => {
  const first = String.fromCodePoint(0x25_01);
  const last = String.fromCodePoint(0x25_7f);
  assert.deepEqual(violations(`plain +--\n  a${first}\n| safe\n  b${last}`), [
    { errorRange: [4, 1], lineNumber: 2 },
    { errorRange: [4, 1], lineNumber: 4 }
  ]);
});
