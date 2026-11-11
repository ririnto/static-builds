import assert from "node:assert/strict";
import { test } from "node:test";

import MarkdownIt from "markdown-it";
import { lint } from "markdownlint/sync";

import noEmDash from "./no-em-dash.ts";

void test("ignores em dashes in inline HTML code but reports prose", () => {
  assert.deepEqual(
    (
      lint({
        config: {
          default: false,
          "docs/no-em-dash": true
        },
        customRules: [noEmDash],
        markdownItFactory: () => new MarkdownIt({ html: true }),
        strings: {
          input: `Literal <code>${String.fromCodePoint(0x20_14)}</code>.\n\nProse ${String.fromCodePoint(0x20_14)} outside.`
        }
      }).input ?? []
    ).map(({ lineNumber }) => lineNumber),
    [3]
  );
});
