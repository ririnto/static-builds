declare module "markdownlint-cli2/parsers/jsonc" {
  export default function parseJsonc(text: string): unknown;
}

declare module "markdownlint-rule-search-replace" {
  import type { Rule } from "markdownlint";

  const rule: Rule;
  export default rule;
}
