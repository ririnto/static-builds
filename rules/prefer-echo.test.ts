import { equal, ok } from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { appendFileSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import process from "node:process";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

await test("reports plain static printf lines and preserves other output", () => {
  const directory = mkdtempSync(
    path.join(tmpdir(), "static-builds-prefer-echo-")
  );
  try {
    const fixture = path.join(directory, "output.sh");
    const runRule = () =>
      spawnSync(
        process.execPath,
        [fileURLToPath(new URL("prefer-echo.ts", import.meta.url)), fixture],
        { encoding: "utf-8" }
      );
    const controls = String.raw`echo 'Ready'
printf '%s' 'Ready'
printf 'Ready\tset\n'
printf 'Value: %d\n' 7
printf '%s\n' "$message"
printf '%s\n' '-n'
printf '%s\n' 'a\b'
printf -- 'Ready\n'
printf 'Ready\n' > "$output"
> "$output" \
  printf 'Ready\n'
cat <<'OUTPUT'
printf 'Ready\n'
OUTPUT
`;
    writeFileSync(fixture, controls);
    const clean = runRule();
    equal(clean.status, 0, clean.stderr);
    equal(clean.stderr, "");
    appendFileSync(
      fixture,
      String.raw`printf 'Ready\n'
printf '%s\n' "All done"
`
    );
    const violation = runRule();
    equal(violation.status, 1, violation.stderr);
    const line = controls.split("\n").length;
    const diagnostics = violation.stderr.trimEnd().split("\n");
    equal(diagnostics.length, 2, violation.stderr);
    ok(
      diagnostics[0]?.startsWith(`${fixture}:${line}: R32/prefer-echo:`),
      violation.stderr
    );
    ok(
      diagnostics[1]?.startsWith(`${fixture}:${line + 1}: R32/prefer-echo:`),
      violation.stderr
    );
  } finally {
    rmSync(directory, { recursive: true });
  }
});
