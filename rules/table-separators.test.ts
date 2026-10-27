import assert from "node:assert/strict";
import { test } from "node:test";

import { applyFixes } from "markdownlint";
import { lint } from "markdownlint/sync";

import rule from "./table-separators.ts";

const check = (input: string) =>
  lint({
    config: { default: false, "docs/table-separators": true },
    customRules: [rule],
    strings: { input }
  }).input ?? [];

test("accepts three hyphens and all table alignment markers", () => {
  const input =
    "| A | B | C | D |\n| --- | :--- | ---: | :---: |\n| 1 | 2 | 3 | 4 |\n";
  assert.deepEqual(check(input), []);
});

test("reports and fixes short and long delimiters while preserving alignment", () => {
  const input =
    "| A | B | C | D |\n| -- | ----- | :-: | ---: |\n| 1 | 2 | 3 | 4 |\n";
  const errors = check(input);
  assert.deepEqual(
    errors.map(({ lineNumber, errorRange }) => ({ errorRange, lineNumber })),
    [
      { errorRange: [3, 2], lineNumber: 2 },
      { errorRange: [8, 5], lineNumber: 2 },
      { errorRange: [17, 1], lineNumber: 2 }
    ]
  );
  const fixed = applyFixes(input, errors);
  assert.equal(
    fixed,
    "| A | B | C | D |\n| --- | --- | :---: | ---: |\n| 1 | 2 | 3 | 4 |\n"
  );
  assert.equal(applyFixes(fixed, check(fixed)), fixed);
});

test("ignores data rows and fenced table examples", () => {
  const input =
    "| A | B |\n| --- | --- |\n| - | - |\n| -- | ---- |\n\n```markdown\n| A |\n| - |\n```\n";
  assert.deepEqual(check(input), []);
});

test("checks pipeless and blockquoted tables", () => {
  const input = "A | B\n-- | ---\n1 | 2\n\n> | A |\n> | - |\n> | 1 |\n";
  assert.deepEqual(
    check(input).map(({ lineNumber, errorRange }) => ({
      errorRange,
      lineNumber
    })),
    [
      { errorRange: [1, 2], lineNumber: 2 },
      { errorRange: [5, 1], lineNumber: 6 }
    ]
  );
});
