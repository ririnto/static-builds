import { execFileSync } from "node:child_process";
import path from "node:path";

const skill = ".claude/skills/formal-verification";
const topic = /^[a-z0-9]+(?:-[a-z0-9]+)*$/u;
const [revision, ...extra] = process.argv.slice(2);
if (extra.length > 0 || revision?.startsWith("-")) {
  throw new Error("Usage: pnpm run check:formal-layout [revision]");
}
const files = execFileSync(
  "git",
  revision === undefined
    ? ["ls-files", "--cached", "-z"]
    : ["ls-tree", "-rz", "--name-only", revision],
  { cwd: new URL("../../../../", import.meta.url), encoding: "utf-8" }
)
  .split("\0")
  .filter(Boolean);
const modelFolders = new Set(
  files
    .filter((file) => file.endsWith(".tla"))
    .map((file) => path.posix.dirname(file))
);
const legacyScript =
  /^(?:statechart(?:-[a-z]+)?|render-statecharts)\.(?:[cm]?js|ts)$/u;
const script = /\.(?:[cm]?js|[cm]?ts)$/u;
for (const file of files) {
  const folder = path.posix.dirname(file);
  const name = path.posix.basename(file);
  const extension = path.posix.extname(file);
  const inSkill = file.startsWith(`${skill}/`);
  const [section, model, leaf, ...nested] = inSkill
    ? file.slice(skill.length + 1).split("/")
    : [];
  let expected: string | undefined;
  if (
    extension === ".tla" ||
    (extension === ".cfg" &&
      (inSkill || modelFolders.has(folder) || name === "BranchLock.cfg"))
  ) {
    if (
      section !== "assets" ||
      !topic.test(model ?? "") ||
      !leaf ||
      nested.length > 0
    ) {
      expected = "assets/<model>/";
    }
  } else if (
    [".mmd", ".mnd", ".svg"].includes(extension) &&
    (inSkill ||
      modelFolders.has(folder) ||
      name === "statechart.mmd" ||
      name === "statechart.mnd" ||
      name === "statechart.svg")
  ) {
    if (
      extension === ".mnd" ||
      section !== "references" ||
      !topic.test(model ?? "") ||
      !leaf ||
      nested.length > 0
    ) {
      expected = "references/<model>/ with a .mmd or .svg filename";
    }
  } else if (
    ((inSkill && script.test(name)) || legacyScript.test(name)) &&
    (section !== "scripts" ||
      extension !== ".ts" ||
      (leaf !== undefined && !topic.test(model ?? "")) ||
      nested.length > 0)
  ) {
    expected = "scripts/ or scripts/<model>/ with a .ts filename";
  }
  if (expected !== undefined) {
    process.stderr.write(`${file}: expected ${skill}/${expected}\n`);
    process.exitCode = 1;
  }
}
