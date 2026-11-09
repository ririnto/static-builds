import { readFileSync, realpathSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import MarkdownIt from "markdown-it";
import type { Rule } from "markdownlint";

import { childTokens } from "./token-children.ts";

const headingParser = new MarkdownIt({ html: true });
const htmlAnchorPattern =
  /<[a-z][^>]*?\s(?:id|name)\s*=\s*["']?(?<anchor>[^"'\s>]+)/giu;
const htmlImagePattern = /^<img(?=\s|\/?>)/iu;
const htmlAttributePattern =
  /(?<attribute>\s+[^\s=/>]+(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s"'=<>`]+))?)/gu;
const htmlSourcePattern =
  /^\s+src\s*=\s*(?:"(?<double>[^"]*)"|'(?<single>[^']*)'|(?<unquoted>[^\s"'=<>`]+))/iu;
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
      let slug = anchor;
      let count = counts.get(anchor) ?? 0;
      while (counts.has(slug)) {
        count += 1;
        slug = `${anchor}-${count}`;
      }
      counts.set(anchor, count);
      counts.set(slug, 0);
      anchors.add(slug);
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
 * Require existing relative Markdown link and Markdown or HTML image destinations, with link
 * fragments on Markdown files matching rendered heading anchors, without
 * changing source or imposing fragment semantics on images or other files.
 */
export default {
  description:
    "Relative link destinations must exist and fragments must match heading anchors",
  function: (params, onError) => {
    const destinations = params.parsers.markdownit.tokens.flatMap((parent) => {
      const result = childTokens(parent).flatMap((token) => {
        const destination = token.attrs?.find(
          ([name]) =>
            (token.type === "link_open" && name === "href") ||
            (token.type === "image" && name === "src")
        )?.[1];
        return [
          {
            destination,
            lineNumber: token.lineNumber,
            link: token.type === "link_open"
          }
        ];
      });
      let offset = 0;
      const html =
        parent.type === "html_block"
          ? (headingParser.parseInline(parent.content, {})[0].children ?? [])
          : childTokens(parent);
      for (const token of html) {
        if (
          token.type === "html_inline" &&
          htmlImagePattern.test(token.content)
        ) {
          const source = [...token.content.matchAll(htmlAttributePattern)]
            .map(
              (match) =>
                htmlSourcePattern.exec(match.groups?.attribute ?? "")?.groups
            )
            .find(Boolean);
          if (source) {
            const index = parent.content.indexOf(token.content, offset);
            offset = index + token.content.length;
            result.push({
              destination: headingParser.utils.unescapeAll(
                source.double ?? source.single ?? source.unquoted ?? ""
              ),
              lineNumber:
                parent.lineNumber +
                parent.content.slice(0, index).split("\n").length -
                1,
              link: false
            });
          }
        }
      }
      return result;
    });
    for (const { destination, lineNumber, link } of destinations) {
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
              lineNumber
            });
          } else if (
            link &&
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
              lineNumber
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
              lineNumber
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
