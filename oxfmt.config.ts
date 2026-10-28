import { defineConfig } from "oxfmt";
import ultracite from "ultracite/oxfmt";

/**
 * Configure contributor formatting checks.
 */
export default defineConfig({
  ...ultracite,
  ignorePatterns: [...(ultracite.ignorePatterns ?? []), "**/*.md"],
  trailingComma: "none"
});
