import { execFileSync, spawnSync } from "node:child_process";
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
const gitEnv = { ...process.env };
delete gitEnv.GIT_DIR;
delete gitEnv.GIT_WORK_TREE;
delete gitEnv.GIT_INDEX_FILE;
delete gitEnv.GIT_OBJECT_DIRECTORY;
delete gitEnv.GIT_ALTERNATE_OBJECT_DIRECTORIES;
delete gitEnv.GIT_COMMON_DIR;
const gitOptions = {
  env: gitEnv,
  cwd: new URL("../../../../", import.meta.url),
  encoding: "utf-8" as const
};
const tree =
  revision === undefined
    ? undefined
    : execFileSync(
        "git",
        ["rev-parse", "--verify", "--end-of-options", `${revision}^{tree}`],
        gitOptions
      ).trim();
const modeAndPath = "--format=%(objectmode) %(path)";
const tracked = new Map(
  execFileSync(
    "git",
    tree === undefined
      ? ["ls-files", "--cached", "-z", modeAndPath]
      : ["ls-tree", "-rtz", modeAndPath, tree],
    gitOptions
  )
    .split("\0")
    .filter(Boolean)
    .map((entry) => [entry.slice(7), entry.slice(0, 6)])
);
const files = [...tracked.keys()].filter(
  (file) => tracked.get(file) !== "040000"
);
const indexDirectories = new Set<string>();
if (tree === undefined) {
  for (const file of files) {
    for (let folder = dirname(file); folder !== "."; folder = dirname(folder)) {
      indexDirectories.add(folder);
    }
  }
}
const readBlob = (file: string): string =>
  execFileSync("git", ["show", `${tree ?? ""}:${file}`], gitOptions);
const isRegular = (file: string): boolean =>
  tracked.get(file) === "100644" || tracked.get(file) === "100755";
const isDirectory = (folder: string): boolean =>
  tree === undefined
    ? indexDirectories.has(folder)
    : tracked.get(folder) === "040000";
const sourcePath = (file: string): string | undefined => {
  if (isRegular(file)) {
    return file;
  }
  if (tracked.get(file) !== "120000") {
    return undefined;
  }
  const target = readBlob(file);
  if (target.startsWith("/")) {
    return undefined;
  }
  const parts = target.split("/");
  const parent = file.split("/").slice(0, -1);
  const leaf = parts.pop();
  if (
    leaf === undefined ||
    leaf === "" ||
    leaf === "." ||
    leaf === ".." ||
    leaf.toLowerCase() === ".git"
  ) {
    return undefined;
  }
  for (const part of parts) {
    if (part === "..") {
      if (parent.pop() === undefined) {
        return undefined;
      }
    } else if (part !== "" && part !== ".") {
      if (part.toLowerCase() === ".git") {
        return undefined;
      }
      parent.push(part);
      if (!isDirectory(parent.join("/"))) {
        return undefined;
      }
    }
  }
  const resolved = [...parent, leaf].join("/");
  return isRegular(resolved) ? resolved : undefined;
};
const sourceOf = (file: string): string => {
  const resolved = sourcePath(file);
  return resolved === undefined ? "" : readBlob(resolved);
};
const normalizeName = (name: string): string =>
  name.toLowerCase().replaceAll(/[-_]/gu, "");
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
  /(?<![\w$.])(?:import|require)\s*\(\s*(?:\/\*[\s\S]*?\*\/\s*)*(?<quote>["'`])pele\k<quote>\s*(?:\/\*[\s\S]*?\*\/\s*)*\)/u;
const outsideScriptExtension =
  /^\.(?:sh|bash|zsh|ksh|fish|py|rb|pl|php|lua|ps1|bat|cmd|r|jl|kts|gradle|groovy|js|mjs|cjs|jsx|ts|mts|cts|tsx)$/u;
const shebangMatches = spawnSync(
  "git",
  [
    "grep",
    "--text",
    "--line-number",
    "--null",
    "--only-matching",
    "--max-count=1",
    "-e",
    "^#!",
    tree ?? "--cached",
    "--"
  ],
  gitOptions
);
if (
  shebangMatches.error ||
  (shebangMatches.status !== 0 && shebangMatches.status !== 1)
) {
  throw shebangMatches.error ?? new Error(shebangMatches.stderr.trim());
}
const shebangFiles = new Set(
  shebangMatches.stdout.split("\0#!\n").flatMap((match) => {
    const [file, line] = match.split("\0");
    return line === "1"
      ? [tree === undefined ? file : file.slice(tree.length + 1)]
      : [];
  })
);
const isFormalScript = (file: string, extension: string): boolean => {
  if (
    !outsideScriptExtension.test(extension) &&
    !shebangFiles.has(sourcePath(file) ?? "")
  ) {
    return false;
  }
  const source = sourceOf(file);
  return (
    [...source.matchAll(modelFile)].some(({ groups }) =>
      modelNames.has(normalizeName(groups?.stem ?? ""))
    ) ||
    tlcTool.test(source) ||
    peleImport.test(source) ||
    peleImportCall.test(source)
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
