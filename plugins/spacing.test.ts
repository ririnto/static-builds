import { describe, it } from "node:test";

import { RuleTester } from "oxlint/plugins-dev";

import blankLines from "./no-blank-lines-in-functions.ts";
import siblingSpacing from "./sibling-test-spacing.ts";

RuleTester.describe = (name, run) => {
  void describe(name, run);
};
RuleTester.it = (name, run) => {
  void it(name, run);
};

const tester = new RuleTester({
  languageOptions: { parserOptions: { lang: "ts" }, sourceType: "module" }
});

tester.run(
  "no-blank-lines-in-functions",
  blankLines.rules["no-blank-lines-in-functions"],
  {
    invalid: [
      {
        code: "const f = () => {\n\n  return 1;\n};\nf();\n",
        errors: [{ messageId: "noBlankLines" }]
      }
    ],
    valid: [
      {
        code: "const f = () => {\n  'use strict';\n\n  const value = `first\n\nlast`;\n  /* SPDX-License-Identifier: MIT\n\n   * Modification notice. */\n  return value;\n};\nf();\n",
        name: "preserves directives, literals, and comments"
      },
      {
        code: 'import { describe, test } from "vitest";\ndescribe("suite", () => {\n  test("one", () => {});\n\n  test("two", () => {});\n});\n',
        name: "preserves spacing between proven sibling registrations"
      }
    ]
  }
);

tester.run(
  "sibling-test-spacing",
  siblingSpacing.rules["sibling-test-spacing"],
  {
    invalid: [
      {
        code: 'import { test } from "vitest";\ntest("one", () => {});\ntest("two", () => {});\n',
        errors: [{ messageId: "siblingSpacing" }]
      }
    ],
    valid: [
      {
        code: 'import { test } from "vitest";\ntest("one", () => {});\n\ntest("two", () => {});\n',
        name: "accepts one blank line between imported registrations"
      },
      {
        code: 'const test = (value: string) => value;\ntest("one");\ntest("two");\n',
        name: "does not classify local functions as test registrations"
      },
      {
        code: 'import { test } from "vitest";\nfunction local(test: (value: string) => void) {\n  test("one");\n  test("two");\n}\n',
        name: "does not classify a shadowed import as a test registration"
      }
    ]
  }
);
