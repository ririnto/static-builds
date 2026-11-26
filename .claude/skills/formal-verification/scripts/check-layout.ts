import { execFileSync } from "node:child_process";
import { basename, dirname, extname } from "node:path/posix";

import { isAlias, isScalar, parseDocument } from "yaml";

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
const sourceOf = (file: string): string =>
  execFileSync("git", ["show", `${revision ?? ""}:${file}`], gitOptions);
const normalizeName = (name: string): string =>
  name.toLowerCase().replaceAll(/[\s_-]/gu, "");
const models = files.filter((file) => extname(file) === ".tla");
const modelFolders = new Set(models.map((file) => dirname(file)));
const modelNames = new Set(
  models.map((file) => normalizeName(basename(file, ".tla")))
);
const isInSkill = (file: string): boolean => file.startsWith(`${skill}/`);
const belongsToModel = (file: string): boolean =>
  modelFolders.has(dirname(file)) ||
  modelNames.has(normalizeName(basename(file, extname(file))));
const frontMatter =
  /^(?:\uFEFF)?---[ \t]*\r?\n(?<fields>[\s\S]*?)(?<=\n)(?:---|\.\.\.)[ \t]*(?:\r?\n|$)/u;
const chartTitle = (file: string): string | undefined => {
  const fields = frontMatter.exec(sourceOf(file))?.groups?.fields;
  if (fields === undefined) {
    return undefined;
  }
  const document = parseDocument(fields);
  if (document.errors.length > 0) {
    return undefined;
  }
  const title = document.get("title", true);
  const resolved = isAlias(title) ? title.resolve(document) : title;
  return isScalar(resolved) && typeof resolved.value === "string"
    ? normalizeName(resolved.value)
    : undefined;
};
const chartSources = files.filter((file) => {
  const extension = extname(file);
  if (extension !== ".mmd" && extension !== ".mnd") {
    return false;
  }
  if (isInSkill(file) || belongsToModel(file)) {
    return true;
  }
  const title = chartTitle(file);
  return title !== undefined && modelNames.has(title);
});
const formalCharts = new Set(chartSources);
const chartSvgs = new Set(
  chartSources.map((file) => `${file.slice(0, -4)}.svg`)
);
const script = /\.[cm]?[jt]s$/u;
const extraScript = /\.(?:bash|fish|jsx|ksh|py[iw]?|sh|tsx|zsh)$/u;
const skillScriptMetadata = new Set([".json", ".md", ".yaml", ".yml"]);
const skillScriptHeader =
  /^(?:#!|import[ \t]+(?:[*{"']|[\w.]+(?:[ \t]+(?:as|from)[ \t]+|[ \t]*(?:[,;\r\n]|$)))|export[ \t]+(?:async|class|const|default|function)\b|from[ \t]+[\w.]+[ \t]+import[ \t]+|(?:async[ \t]+)?def[ \t]+\w+[ \t]*\(|set[ \t]+-[a-z]|(?:print|console\.log)[ \t]*\(|printf[ \t]+["'])/u;
const isSkillScript = (
  file: string,
  section: string | undefined,
  extension: string
): boolean =>
  script.test(extension) ||
  extraScript.test(extension) ||
  (section === "scripts" && !skillScriptMetadata.has(extension)) ||
  skillScriptHeader.test(sourceOf(file).trimStart());
const modelFile =
  /(?<![\p{L}\p{N}_.-])(?<stem>[\p{L}\p{N}_-][\p{L}\p{N}_.-]*)\.tla(?![\p{L}\p{N}_.-])/gu;
const tlcTool =
  /(?<![\w.-])(?:tlc2\.TLC|tla2tools(?:-[\w.-]+)?\.jar)(?![\w.-])/u;
const peleImport =
  /^\s*import\b(?:[^"'();]*?\bfrom)?\s*(?<quote>["'])pele\k<quote>/mu;
const peleImportCall =
  /(?<![\w$.])(?:import|require)\s*\(\s*(?<quote>["'])pele\k<quote>\s*\)/u;
const isFormalScript = (file: string, extension: string): boolean => {
  const hasScriptExtension =
    script.test(extension) || extraScript.test(extension);
  if (!hasScriptExtension && extension !== "") {
    return false;
  }
  const source = sourceOf(file);
  return (
    (hasScriptExtension || source.startsWith("#!")) &&
    ([...source.matchAll(modelFile)].some(({ groups }) =>
      modelNames.has(normalizeName(groups?.stem ?? ""))
    ) ||
      tlcTool.test(source) ||
      peleImport.test(source) ||
      peleImportCall.test(source))
  );
};
for (const file of files) {
  const folder = dirname(file);
  const extension = extname(file);
  const inSkill = isInSkill(file);
  const [section, model, leaf, ...nested] = inSkill
    ? file.slice(skill.length + 1).split("/")
    : [];
  let expected: string | undefined;
  if (
    extension === ".tla" ||
    (extension === ".cfg" && (inSkill || belongsToModel(file)))
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
    formalCharts.has(file) ||
    (extension === ".svg" &&
      (inSkill || modelFolders.has(folder) || chartSvgs.has(file)))
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
    (section !== "scripts" ||
      extension !== ".ts" ||
      (leaf !== undefined && !topic.test(model ?? "")) ||
      nested.length > 0) &&
    (inSkill
      ? isSkillScript(file, section, extension)
      : isFormalScript(file, extension))
  ) {
    expected = "scripts/ or scripts/<model>/ with a .ts filename";
  }
  if (expected !== undefined) {
    process.stderr.write(`${file}: expected ${skill}/${expected}\n`);
    process.exitCode = 1;
  }
}
