import { defineConfig } from "oxlint";
import core from "ultracite/oxlint/core";

/**
 * Configure contributor lint checks.
 */
export default defineConfig({
  env: core.env,
  extends: [core],
  ignorePatterns: core.ignorePatterns,
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
    "typescript/no-floating-promises": ["error", { ignoreVoid: false }]
  }
});
