import { execFileSync, spawnSync } from "node:child_process";
import path from "node:path";

const skill = ".claude/skills/formal-verification";
const topic = /^[a-z0-9]+(?:-[a-z0-9]+)*$/u;
const [revision, ...extra] = process.argv.slice(2);
if (extra.length > 0 || revision?.startsWith("-")) {
  throw new Error(
    "Usage: pnpm --filter @static-builds/formal-verification run check:layout [revision]"
  );
}
const gitOptions = {
  cwd: new URL("../../../../", import.meta.url),
  encoding: "utf-8" as const
};
const files = execFileSync(
  "git",
  revision === undefined
    ? ["ls-files", "--cached", "-z"]
    : ["ls-tree", "-rz", "--name-only", revision],
  gitOptions
)
  .split("\0")
  .filter(Boolean);
const formalToolPatterns = [
  "[\"']tlc2[.]TLC[\"']|require[.]resolve[(][[:blank:]]*[\"'][^\"']*tla2tools[.]jar[\"']",
  "^[[:blank:]]*import[[:blank:]]+[{][[:blank:]]*render([[:blank:]]+as[[:blank:]]+[$_[:alnum:]]+)?[[:blank:]]*[}][[:blank:]]+from[[:blank:]]+[\"']pele[\"']",
  "^[[:blank:]]*import[[:blank:]]+[{][[:blank:]]*runStatechart[[:blank:]]*[}][[:blank:]]+from[[:blank:]]+[\"'][^\"']*/statechart[.][cm]?[jt]s[\"']"
];
const matches = spawnSync(
  "git",
  [
    "grep",
    "-IlzE",
    ...formalToolPatterns.flatMap((pattern) => ["-e", pattern]),
    revision ?? "--cached",
    "--",
    "*.ts",
    "*.mts",
    "*.cts",
    "*.js",
    "*.mjs",
    "*.cjs",
    `:(exclude,glob)${skill}/**`
  ],
  gitOptions
);
if (matches.error || (matches.status !== 0 && matches.status !== 1)) {
  throw matches.error ?? new Error(matches.stderr.trim() || "git grep failed");
}
const formalScripts = new Set(
  matches.stdout
    .split("\0")
    .filter(Boolean)
    .map((file) =>
      revision === undefined ? file : file.slice(revision.length + 1)
    )
);
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
    ((inSkill && script.test(name)) ||
      legacyScript.test(name) ||
      formalScripts.has(file)) &&
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
