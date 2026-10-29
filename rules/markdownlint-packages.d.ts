declare module "markdownlint-cli2/parsers/jsonc" {
  import type { Configuration } from "markdownlint";

  export default function parseJsonc(text: string): { config: Configuration };
}

declare module "markdownlint-rule-search-replace" {
  import type { Rule } from "markdownlint";

  const rule: Rule;
  export default rule;
}
