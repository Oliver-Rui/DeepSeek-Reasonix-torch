import { describe, expect, it } from "vitest";
import { blockRange, firstImplicit, insertionAt, pinnedEnd, placeAt, pointerInsertion, previewKeys, slotAt, stepInsertion } from "./reorder";
import type { ReorderItem } from "./reorder";

const item = (key: string, pinned = false, rememberable = true): ReorderItem => ({ key, pinned, rememberable });

describe("slotAt", () => {
  const boxes = [{ top: 0, height: 40 }, { top: 40, height: 40 }];
  it("returns the first row whose middle is below the pointer", () => {
    expect(slotAt(boxes, 5)).toBe(0);
    expect(slotAt(boxes, 30)).toBe(1);
    expect(slotAt(boxes, 45)).toBe(1);
    expect(slotAt(boxes, 200)).toBe(2);
  });
});

describe("insertionAt", () => {
  it("removes the carried row from the slot count", () => {
    expect(insertionAt(1, 0)).toBe(0);
    expect(insertionAt(1, 2)).toBe(1);
    expect(insertionAt(1, 3)).toBe(2);
  });
});

describe("blockRange", () => {
  const items = [item("p1", true), item("p2", true), item("u1"), item("u2"), item("i", false, false)];
  it("stops at the pinned and implicit boundaries", () => {
    expect(blockRange(items, 1)).toEqual({ start: 0, end: 2, inside: true });
    expect(blockRange(items, 2)).toEqual({ start: 2, end: 4, inside: true });
  });
  it("lets an implicit project join only the remembered unpinned block", () => {
    expect(blockRange(items, 4)).toEqual({ start: 2, end: 4, inside: false });
  });
});

describe("pointerInsertion", () => {
  const items = [item("p1", true), item("p2", true), item("u1"), item("u2")];
  it("is the slot without the carried row, and null where the row already sits", () => {
    expect(pointerInsertion(items, 2, 0)).toBeNull();
    expect(pointerInsertion(items, 2, 2)).toBeNull();
    expect(pointerInsertion(items, 2, 3)).toBeNull();
    expect(pointerInsertion(items, 2, 4)).toBe(3);
    expect(pointerInsertion(items, 1, 0)).toBe(0);
  });
  it("never crosses the pinned boundary", () => {
    expect(pointerInsertion(items, 2, 0)).toBeNull();
    expect(pointerInsertion(items, 1, 4)).toBeNull();
  });
});

describe("placeAt", () => {
  it("names the next remembered project and paints the line before it", () => {
    const full = [item("a"), item("b"), item("i", false, false)];
    expect(placeAt(full, full, "b", 0)).toEqual({ before: "a", previewAt: 0, line: { key: "a", edge: "before" } });
  });
  it("turns a drop among implicit projects into the end of the remembered order", () => {
    const full = [item("a"), item("b"), item("i", false, false)];
    const placed = placeAt(full, full, "a", 2);
    expect(placed.before).toBe("");
    expect(placed.previewAt).toBe(1);
    expect(placed.line).toEqual({ key: "i", edge: "before" });
    expect(previewKeys(full, "a", placed.previewAt)).toEqual(["b", "a", "i"]);
  });
  it("keeps a hidden session as the target when the visible list is truncated", () => {
    const full = [item("a"), item("b"), item("c")];
    const visible = full.slice(0, 2);
    const placed = placeAt(visible, full, "a", 1);
    expect(placed.before).toBe("c");
    expect(placed.previewAt).toBe(1);
    expect(placed.line).toEqual({ key: "b", edge: "after" });
  });
  it("ends a pinned row at the pinned partition and paints the boundary", () => {
    const full = [item("p1", true), item("p2", true), item("u1")];
    const placed = placeAt(full, full, "p1", 2);
    expect(placed.before).toBe("");
    expect(placed.previewAt).toBe(1);
    expect(placed.line).toEqual({ key: "u1", edge: "before" });
    expect(previewKeys(full, "p1", placed.previewAt)).toEqual(["p2", "p1", "u1"]);
  });
  it("changes nothing when a carried row is put back where it was", () => {
    const full = [item("a"), item("b"), item("c")];
    expect(placeAt(full, full, "b", 1).line).toBeNull();
  });
});

describe("stepInsertion", () => {
  const items = [item("a"), item("b"), item("c")];
  it("moves one visible slot and stops at the edges", () => {
    expect(stepInsertion(items, 1, -1)).toBe(0);
    expect(stepInsertion(items, 1, 1)).toBe(2);
    expect(stepInsertion(items, 0, -1)).toBeNull();
    expect(stepInsertion(items, 2, 1)).toBeNull();
  });
  it("has no menu step for a project the kernel never stored", () => {
    const withImplicit = [item("a"), item("i", false, false)];
    expect(stepInsertion(withImplicit, 1, -1)).toBeNull();
  });
});

describe("partition helpers", () => {
  it("finds the pinned end and the first implicit row", () => {
    const items = [item("p", true), item("u"), item("i", false, false)];
    expect(pinnedEnd(items)).toBe(1);
    expect(firstImplicit(items)).toBe(2);
    expect(firstImplicit([item("u")])).toBe(-1);
  });
});
