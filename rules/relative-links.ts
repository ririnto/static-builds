import { readFileSync, realpathSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import MarkdownIt from "markdown-it";
import type { Rule } from "markdownlint";

import { childTokens } from "./token-children.ts";

const headingParser = new MarkdownIt({ html: true });
const htmlAnchorPattern =
  /<[a-z][^>]*?\s(?:id|name)\s*=\s*["']?(?<anchor>[^"'\s>]+)/giu;
const slugExcluded =
  /[^\p{Letter}\p{Mark}\p{Number}\p{Connector_Punctuation}\- ]/gu;

const headingAnchors = (file: string): Set<string> => {
  const anchors = new Set(["top"]);
  const counts = new Map<string, number>();
  const tokens = headingParser.parse(readFileSync(file, "utf-8"), {});
  for (const [index, token] of tokens.entries()) {
    const anchor =
      token.type === "heading_open"
        ? (tokens[index + 1]?.children ?? [])
            .filter(
              (child) => child.type === "text" || child.type === "code_inline"
            )
            .map((child) => child.content)
            .join("")
            .toLowerCase()
            .replaceAll(slugExcluded, "")
            .replaceAll(" ", "-")
        : "";
    if (anchor) {
      const count = counts.get(anchor) ?? 0;
      if (count) {
        anchors.add(`${anchor}-${count}`);
      }
      counts.set(anchor, count + 1);
      anchors.add(anchor);
    }
    for (const html of [token, ...(token.children ?? [])].filter(
      (item) => item.type === "html_block" || item.type === "html_inline"
    )) {
      for (const match of html.content
        .replaceAll(/<!--[\s\S]*?(?:-->|$)/gu, "")
        .matchAll(htmlAnchorPattern)) {
        anchors.add(match.groups?.anchor ?? "");
      }
    }
  }
  return anchors;
};

/**
 * Require existing relative Markdown link and image destinations, with link
 * fragments on Markdown files matching rendered heading anchors, without
 * changing source or imposing fragment semantics on images or other files.
 */
export default {
  description:
    "Relative link destinations must exist and fragments must match heading anchors",
  function: (params, onError) => {
    for (const token of params.parsers.markdownit.tokens.flatMap(childTokens)) {
      const destination = token.attrs?.find(
        ([name]) =>
          (token.type === "link_open" && name === "href") ||
          (token.type === "image" && name === "src")
      )?.[1];
      if (
        Boolean(destination) &&
        destination !== undefined &&
        !destination.startsWith("/") &&
        !destination.startsWith("#") &&
        !URL.canParse(destination)
      ) {
        try {
          const destinationURL = new URL(
            destination,
            pathToFileURL(params.name)
          );
          const destinationPath = fileURLToPath(destinationURL);
          if (
            [destinationPath, realpathSync(destinationPath)].some((target) => {
              const relative = path.relative(process.cwd(), target);
              return (
                relative === ".." ||
                relative.startsWith(`..${path.sep}`) ||
                path.isAbsolute(relative)
              );
            })
          ) {
            onError({
              context: destination,
              lineNumber: token.lineNumber
            });
          } else if (
            token.type === "link_open" &&
            destinationURL.hash.length > 1 &&
            destinationPath.toLowerCase().endsWith(".md") &&
            !headingAnchors(destinationPath).has(
              decodeURIComponent(destinationURL.hash.slice(1))
            )
          ) {
            onError({
              context: destination,
              detail:
                "Fragment does not match a rendered Markdown heading anchor.",
              lineNumber: token.lineNumber
            });
          }
        } catch (error) {
          if (
            error instanceof URIError ||
            (error instanceof Error &&
              "code" in error &&
              [
                "ERR_INVALID_FILE_URL_PATH",
                "ERR_INVALID_ARG_VALUE",
                "ENOENT",
                "ENAMETOOLONG",
                "ENOTDIR",
                "EISDIR",
                "ELOOP",
                "EACCES"
              ].some((code) => code === error.code))
          ) {
            onError({
              context: destination,
              lineNumber: token.lineNumber
            });
          } else {
            throw error;
          }
        }
      }
    }
  },
  names: ["docs/relative-links"],
  parser: "markdownit",
  tags: ["docs", "links"]
} satisfies Rule;
