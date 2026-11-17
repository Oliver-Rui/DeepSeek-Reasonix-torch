// @vitest-environment jsdom
import { expect, it } from "vitest";
import { monthBuckets } from "./timeBuckets";
import "./testkit";

it("cuts a newest-first list into day sections, one mark per section", () => {
  const now = Date.now();
  // Fixed points against the local midnight, so the fixtures never drift
  // across a day boundary depending on when the suite runs.
  const dayStart = new Date().setHours(0, 0, 0, 0);
  // 呼叫方的口径：缺失的活动戳按"现在"计。
  const items = [
    { key: "a", at: now },                          // 正在写入
    { key: "b", at: dayStart + Math.floor((now - dayStart) / 2) }, // 今天过半处 — 永远是今天
    { key: "c", at: now - 2 * 86400_000 },          // 前天 — 7 天内
  ];
  const { marks, counts } = monthBuckets(items);
  expect([...marks.entries()]).toEqual([["a", "今天"], ["c", "7 天内"]]);
  expect(counts.get("今天")).toBe(2);
  expect(counts.get("7 天内")).toBe(1);
});

it("yesterday sits between today and the week tail", () => {
  const now = Date.now();
  const dayStart = new Date().setHours(0, 0, 0, 0);
  const { marks } = monthBuckets([
    { key: "a", at: now },
    { key: "b", at: dayStart - 60_000 },            // 昨天最后一分钟 — 昨天
    { key: "c", at: now - 3 * 86400_000 },
  ]);
  expect([...marks.entries()]).toEqual([["a", "今天"], ["b", "昨天"], ["c", "7 天内"]]);
});
