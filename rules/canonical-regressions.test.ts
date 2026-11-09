import assert from "node:assert/strict";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { test } from "node:test";

import MarkdownIt from "markdown-it";
import { lint } from "markdownlint/sync";

import noEmDash from "./no-em-dash.ts";
import relativeLinks from "./relative-links.ts";

void test("ignores em dashes in inline HTML code but reports prose", () => {
  const result = lint({
    config: {
      default: false,
      "docs/no-em-dash": true,
      "docs/relative-links": true
    },
    customRules: [noEmDash],
    markdownItFactory: () => new MarkdownIt({ html: true }),
    strings: {
      input: `Literal <code>${String.fromCodePoint(0x20_14)}</code>.\n\nProse ${String.fromCodePoint(0x20_14)} outside.`
    }
  });
  assert.deepEqual(
    (result.input ?? []).map(({ lineNumber }) => lineNumber),
    [3]
  );
});

void test("checks HTML image destinations and ignores examples", () => {
  const directory = mkdtempSync(path.join(process.cwd(), ".tmp-markdownlint-"));
  try {
    writeFileSync(path.join(directory, "image.svg"), "<svg></svg>");
    const source = path.join(directory, "source.md");
    writeFileSync(
      source,
      '# Fixture\n\n<img src="image.svg">\n\n<img src="missing.png">\n\n<!-- <img src="ignored.png"> -->\n\n```html\n<img src="example.png">\n```\n'
    );
    const result = lint({
      config: {
        default: false,
        "docs/no-em-dash": true,
        "docs/relative-links": true
      },
      customRules: [relativeLinks],
      files: [source],
      markdownItFactory: () => new MarkdownIt({ html: true })
    });
    assert.deepEqual(
      (result[source] ?? []).map(({ lineNumber }) => lineNumber),
      [5]
    );
  } finally {
    rmSync(directory, { recursive: true });
  }
});

void test("resolves colliding duplicate heading fragments", () => {
  const directory = mkdtempSync(path.join(process.cwd(), ".tmp-markdownlint-"));
  try {
    writeFileSync(
      path.join(directory, "anchors.md"),
      "# Foo\n\n## Foo\n\n## Foo-1\n"
    );
    const source = path.join(directory, "source.md");
    writeFileSync(
      source,
      "# Fixture\n\n[Valid](anchors.md#foo-1-1)\n\n[Missing](anchors.md#foo-2)\n"
    );
    const result = lint({
      config: {
        default: false,
        "docs/no-em-dash": true,
        "docs/relative-links": true
      },
      customRules: [relativeLinks],
      files: [source],
      markdownItFactory: () => new MarkdownIt({ html: true })
    });
    assert.deepEqual(
      (result[source] ?? []).map(({ lineNumber }) => lineNumber),
      [5]
    );
  } finally {
    rmSync(directory, { recursive: true });
  }
});
