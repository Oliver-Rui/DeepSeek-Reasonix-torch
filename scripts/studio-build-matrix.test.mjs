import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { test } from "node:test";

const root = fileURLToPath(new URL("..", import.meta.url));
const script = (...args) =>
  spawnSync("bash", ["scripts/studio-build-matrix.sh", ...args], { cwd: root, encoding: "utf8" });
const legs = (...args) => JSON.parse(script(...args).stdout).include;

test("the whole line is six legs, one per platform and architecture", (t) => {
  assert.deepEqual(legs("all").map((l) => l.platform), [
    "windows/amd64", "windows/arm64",
    "darwin/amd64", "darwin/arm64",
    "linux/amd64", "linux/arm64",
  ]);
});

test("a Windows-only release is the two Windows legs and nothing else", (t) => {
  const win = legs("windows");
  assert.deepEqual(win.map((l) => l.platform), ["windows/amd64", "windows/arm64"]);
  assert.deepEqual(win.map((l) => l.artifact), [
    "studio-build-windows-amd64", "studio-build-windows-arm64",
  ]);
  assert.deepEqual(win.map((l) => l.runner), ["windows-latest", "windows-11-arm"]);
});

test("naming no platform set means the whole line", (t) => {
  assert.deepEqual(legs(), legs("all"));
});

test("every leg names the runner and the artifact publish downloads it by", (t) => {
  for (const leg of legs("all")) {
    assert.ok(leg.runner, leg.platform);
    assert.ok(leg.artifact.startsWith("studio-build-"), leg.platform);
    if (leg.platform.startsWith("windows/")) {
      assert.ok(leg.goarch && leg.unpacked, `${leg.platform} needs its architecture and bundle directory`);
    }
  }
});

test("a platform set the line does not know is refused, not read as everything", (t) => {
  const result = script("darwin");
  assert.notEqual(result.status, 0);
  assert.ok(result.stderr.includes("usage"));
});
