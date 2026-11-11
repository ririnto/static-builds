import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

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

void it("enables both spacing rules through the root configuration", () => {
  const root = fileURLToPath(new URL("..", import.meta.url));
  const directory = mkdtempSync(path.join(tmpdir(), "spacing-config-"));
  const fixture = path.join(directory, "fixture.ts");
  try {
    writeFileSync(
      fixture,
      'import { test } from "vitest";\n\ntest("one", () => {\n\n  return 1;\n});\ntest("two", () => {});\n'
    );
    const result = spawnSync(
      "pnpm",
      [
        "exec",
        "ultracite",
        "check",
        "--format",
        "json",
        "--config",
        path.join(root, "oxlint.config.ts"),
        fixture
      ],
      {
        cwd: root,
        encoding: "utf-8"
      }
    );
    assert.equal(result.status, 1, result.stderr);
    const output: unknown = JSON.parse(
      result.stdout.slice(result.stdout.indexOf("{"))
    );
    assert.ok(
      output !== null && typeof output === "object" && "diagnostics" in output
    );
    assert.ok(Array.isArray(output.diagnostics));
    const diagnostics: unknown[] = output.diagnostics;
    const codes = new Set(
      diagnostics.flatMap((diagnostic) => {
        if (
          diagnostic !== null &&
          typeof diagnostic === "object" &&
          "code" in diagnostic &&
          typeof diagnostic.code === "string" &&
          "severity" in diagnostic &&
          diagnostic.severity === "error"
        ) {
          return [diagnostic.code];
        }
        return [];
      })
    );
    assert.ok(codes.has("style-blank-lines(no-blank-lines-in-functions)"));
    assert.ok(codes.has("style-spacing(sibling-test-spacing)"));
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
});
