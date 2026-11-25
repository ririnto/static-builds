import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import process from "node:process";

const literalLine =
  /^[ \t]*printf[ \t]+(?<quote>['"])[A-Za-z0-9][A-Za-z0-9 .,:_/-]*\\n\k<quote>[ \t]*$/u;
const stringLine =
  /^[ \t]*printf[ \t]+(?<format>['"])%s\\n\k<format>[ \t]+(?<argument>['"])[A-Za-z0-9][A-Za-z0-9 .,:_/-]*\k<argument>[ \t]*$/u;

const files = process.argv.slice(2);
for (const file of files.length > 0
  ? files
  : execFileSync("git", ["ls-files", "-z", "--", "*.sh", ".husky/pre-*"], {
      encoding: "utf-8"
    })
      .split("\0")
      .filter(Boolean)) {
  const source = readFileSync(file, "utf-8");
  const candidates = source
    .split("\n")
    .map((line) => literalLine.test(line) || stringLine.test(line));
  if (candidates.includes(true)) {
    JSON.parse(
      execFileSync("go", ["tool", "shfmt", "--to-json", "--filename", file], {
        encoding: "utf-8",
        input: source,
        maxBuffer: Number.POSITIVE_INFINITY
      }),
      (_key, statement: unknown) => {
        if (
          typeof statement === "object" &&
          statement !== null &&
          "Cmd" in statement &&
          typeof statement.Cmd === "object" &&
          statement.Cmd !== null &&
          "Type" in statement.Cmd &&
          statement.Cmd.Type === "CallExpr" &&
          "Pos" in statement &&
          typeof statement.Pos === "object" &&
          statement.Pos !== null &&
          "Line" in statement.Pos &&
          typeof statement.Pos.Line === "number" &&
          candidates[statement.Pos.Line - 1]
        ) {
          process.stderr.write(
            `${file}:${statement.Pos.Line}: R32/prefer-echo: Use echo for this plain printf line.\n`
          );
          process.exitCode = 1;
        }
        return statement;
      }
    );
  }
}
