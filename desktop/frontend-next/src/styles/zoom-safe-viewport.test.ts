import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

// The shell lays out at 100vw / --zoom and scales by --zoom, so a viewport
// subtraction that skips the division is --zoom times too wide: the context
// card, the model menu foot and the error bars each escaped narrow windows
// by exactly the zoom factor before this contract existed.
const dir = fileURLToPath(new URL(".", import.meta.url));
const offenders: string[] = [];
for (const file of readdirSync(dir).filter((f) => f.endsWith(".css"))) {
  const css = readFileSync(join(dir, file), "utf8");
  for (const m of css.matchAll(/calc\(\s*(100vw|100vh|100dvh)\s+([^)]*)\)/g)) {
    if (!m[2].includes("/ var(--zoom")) {
      offenders.push(`${file}: calc(${m[1]} ${m[2].slice(0, 40)})`);
    }
  }
}

describe("every viewport subtraction divides by --zoom", () => {
  it("keeps clamps honest under interface zoom", () => {
    expect(offenders).toEqual([]);
  });
});
