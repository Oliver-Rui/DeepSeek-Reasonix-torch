import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "node:test";

const verify = fileURLToPath(new URL("verify-studio-artifacts.sh", import.meta.url));
const names = ["amd64", "arm64"].flatMap((arch) => [
  `ReasonixStudio-linux-${arch}.deb`, `ReasonixStudio-linux-${arch}.rpm`, `ReasonixStudio-linux-${arch}.AppImage`,
  `ReasonixStudio-windows-${arch}-installer.exe`, `ReasonixStudio-windows-${arch}.zip`,
  `ReasonixStudio-darwin-${arch}.dmg`, `ReasonixStudio-darwin-${arch}.zip`,
]);

function fixture(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "studio-artifacts-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  for (const name of names) fs.writeFileSync(path.join(dir, name), "package fixture");
  return dir;
}

const run = (dir, ...platforms) => spawnSync("bash", [verify, dir, ...platforms], { encoding: "utf8" });

function fixtureWith(t, list) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "studio-artifacts-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  for (const name of list) fs.writeFileSync(path.join(dir, name), "package fixture");
  return dir;
}

test("all fourteen platform artifacts are required before publication", (t) => {
  assert.equal(run(fixture(t)).status, 0);
});

for (const name of names) {
  test(`missing ${name} cannot be concealed by signatures or extra files`, (t) => {
    const dir = fixture(t);
    fs.unlinkSync(path.join(dir, name));
    for (const present of names) fs.writeFileSync(path.join(dir, present + ".minisig"), "signature");
    fs.writeFileSync(path.join(dir, "ReasonixStudio-linux-aarch64.rpm"), "wrong name");
    const result = run(dir);
    assert.notEqual(result.status, 0);
    assert.ok(result.stderr.includes(name));
  });
}

test("an empty ARM64 package is a failed publication", (t) => {
  const dir = fixture(t);
  fs.truncateSync(path.join(dir, "ReasonixStudio-linux-arm64.deb"));
  assert.notEqual(run(dir).status, 0);
});

test("an empty AppImage is a failed publication", (t) => {
  const dir = fixture(t);
  fs.truncateSync(path.join(dir, "ReasonixStudio-linux-amd64.AppImage"));
  const result = run(dir);
  assert.notEqual(result.status, 0);
  assert.ok(result.stderr.includes("ReasonixStudio-linux-amd64.AppImage"));
});

const windowsNames = names.filter((name) => name.startsWith("ReasonixStudio-windows-"));

test("a release that claims only Windows is checked against the Windows set", (t) => {
  assert.equal(run(fixtureWith(t, windowsNames), "windows").status, 0);
});

test("a Windows-only release is not complete without either architecture", (t) => {
  const dir = fixtureWith(t, windowsNames);
  fs.unlinkSync(path.join(dir, "ReasonixStudio-windows-arm64.zip"));
  const result = run(dir, "windows");
  assert.notEqual(result.status, 0);
  assert.ok(result.stderr.includes("ReasonixStudio-windows-arm64.zip"));
});

test("an unnamed platform set still means every platform the line claims", (t) => {
  assert.notEqual(run(fixtureWith(t, windowsNames)).status, 0);
});

test("a platform the line does not know is refused rather than quietly skipped", (t) => {
  const result = run(fixture(t), "solaris");
  assert.notEqual(result.status, 0);
  assert.ok(result.stderr.includes("solaris"));
});
