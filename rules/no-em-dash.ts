import type { Rule } from "markdownlint";

import { childTokens } from "./token-children.ts";

/**
 * Markdownlint rule rejecting em dashes (U+2014) in Markdown prose while
 * preserving code spans, code blocks, and HTML.
 */
export default {
  description: "Em dashes are not allowed in Markdown prose",
  function: (params, onError) => {
    for (const token of params.parsers.markdownit.tokens.flatMap(childTokens)) {
      if (token.type === "text" && token.content.includes("—")) {
        onError({
          context: token.content.trim(),
          detail: "Use a comma, colon, or a separate sentence instead.",
          lineNumber: token.lineNumber
        });
      }
    }
  },
  names: ["docs/no-em-dash"],
  parser: "markdownit",
  tags: ["docs", "unicode"]
} satisfies Rule;
