// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { useStatusPoll } from "./useStatusPoll";
import "./testkit";

afterEach(() => { cleanup(); vi.useRealTimers(); });

const HIDDEN = "hidden";
const VISIBLE = "visible";

function setHidden(hidden: boolean) {
  Object.defineProperty(document, HIDDEN, { configurable: true, get: () => hidden });
  Object.defineProperty(document, "visibilityState", { configurable: true, get: () => (hidden ? "hidden" : VISIBLE) });
}

// A board read that a hidden page skips must still happen when the page comes
// back. The timer alone would get there eventually, but a window left in the
// background for an hour would draw an empty board for a whole tick after the
// person returns to it — and the board is the one surface that says whether
// work is still running.
it("reads immediately when the page becomes visible again", async () => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  setHidden(true);
  const read = vi.fn(async () => undefined);
  renderHook(() => useStatusPoll(true, read, 10_000));

  await act(async () => { await vi.advanceTimersByTimeAsync(0); });
  expect(read).toHaveBeenCalledTimes(1);

  // While hidden the cadence carries on; the reader itself decides to skip.
  setHidden(false);
  await act(async () => { document.dispatchEvent(new Event("visibilitychange")); });
  expect(read).toHaveBeenCalledTimes(2);
});

// Coming back while still hidden (a pane behind another) is not a reason to
// read: the page cannot draw it, and the reader would skip anyway.
it("ignores the event while the page is still hidden", async () => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  setHidden(true);
  const read = vi.fn(async () => undefined);
  renderHook(() => useStatusPoll(true, read, 10_000));

  await act(async () => { await vi.advanceTimersByTimeAsync(0); });
  await act(async () => { document.dispatchEvent(new Event("visibilitychange")); });
  expect(read).toHaveBeenCalledTimes(1);
});

// A pane that is not on screen must not poll at all, so the listener it would
// have registered is never installed.
it("does not listen when the poll is off", async () => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  setHidden(true);
  const read = vi.fn(async () => undefined);
  renderHook(() => useStatusPoll(false, read, 10_000));
  setHidden(false);
  await act(async () => { document.dispatchEvent(new Event("visibilitychange")); });
  expect(read).not.toHaveBeenCalled();
});
