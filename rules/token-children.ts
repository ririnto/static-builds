import type { MarkdownItToken } from "markdownlint";

const { defineProperty } = Object;

export const childTokens = (parent: MarkdownItToken): MarkdownItToken[] => {
  const children = parent.children ?? [];
  if (!Array.isArray(children)) {
    return children;
  }
  const result: MarkdownItToken[] = [];
  const length = Math.min(
    Math.max(Math.floor(children.length), 0),
    Number.MAX_SAFE_INTEGER
  );
  for (let index = 0; index < length; index += 1) {
    if (index in children) {
      defineProperty(result, result.length, {
        configurable: true,
        enumerable: true,
        value: children[index],
        writable: true
      });
    }
  }
  return result;
};
