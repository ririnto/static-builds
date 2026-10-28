import { defineConfig } from "oxlint";
import core from "ultracite/oxlint/core";

/**
 * Configure contributor lint checks.
 */
export default defineConfig({
  env: core.env,
  extends: [core],
  ignorePatterns: core.ignorePatterns,
  jsPlugins: ["./rules/plugins/tsdoc-plugin.ts"],
  overrides: [
    {
      files: ["**/*.{js,jsx,mjs,cjs}"],
      rules: {
        "jsdoc/require-param": "deny",
        "jsdoc/require-param-name": "deny",
        "jsdoc/require-param-type": "deny",
        "jsdoc/require-returns": "deny",
        "jsdoc/require-returns-description": "allow",
        "jsdoc/require-returns-type": "deny"
      }
    }
  ],
  rules: {
    "oxc/no-accumulating-spread": "deny",
    "tsdoc/require-export-tsdoc": "deny",
    "typescript/consistent-type-imports": [
      "error",
      { fixStyle: "separate-type-imports", prefer: "type-imports" }
    ],
    "typescript/no-floating-promises": ["error", { ignoreVoid: false }]
  }
});
