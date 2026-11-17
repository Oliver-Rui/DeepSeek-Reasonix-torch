// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import "./testkit";
import { SessionHoverCard } from "./SessionHoverCard";
import type { DispatchThread } from "../port/hub";

afterEach(cleanup);

const ws = { root: "/w", name: "w", sessions: [] };
const session = { path: "/w/s/a.jsonl", name: "a", title: "alpha" };
const anchor = { right: 200, top: 100 } as DOMRect;

function draw(threads: DispatchThread[]) {
  const onClose = vi.fn();
  const onOpenSession = vi.fn();
  render(<SessionHoverCard hub={{ dispatchThreads: () => Promise.resolve(threads) } as never}
    ws={ws as never} session={session as never} anchor={anchor}
    onClose={onClose} onKeepOpen={() => {}} onOpenSession={onOpenSession} />);
  return { onClose, onOpenSession };
}

it("shows each dispatch target with its state and time from the audit replay", async () => {
  draw([
    { role: "美术", targetTitle: "poster", targetProject: "p", sessionPath: "/p/1.jsonl", state: "working", since: "2026-11-16T10:00:00Z" },
    { role: "测试", targetTitle: "qa", sessionPath: "/p/2.jsonl", state: "replied" },
  ]);
  await waitFor(() => expect(document.querySelectorAll('[data-action="peer.thread-open"]')).toHaveLength(2));
  const working = document.querySelector('[data-action="peer.thread-open"][data-state="working"]');
  const replied = document.querySelector('[data-action="peer.thread-open"][data-state="replied"]');
  expect(working).toBeTruthy();
  expect(replied).toBeTruthy();
  expect(working!.querySelector("small")?.textContent).toMatch(/\d{1,2}:\d{2}/);
  expect(working!.textContent).toContain("poster");
});

it("still names the role when an older hub has no state", async () => {
  draw([{ role: "策划", key: "s-abc" }]);
  await waitFor(() => expect(screen.getByText(/策划/)).toBeTruthy());
  expect(document.querySelector('[data-action="peer.thread-open"]')?.getAttribute("data-state")).toBeNull();
});

it("opens the target and carries no dispatch entry of its own", async () => {
  const { onOpenSession } = draw([{ role: "美术", sessionPath: "/p/1.jsonl", state: "replied" }]);
  await waitFor(() => expect(document.querySelector('[data-action="peer.thread-open"]')).toBeTruthy());
  fireEvent.click(document.querySelector('[data-action="peer.thread-open"]')!);
  expect(onOpenSession).toHaveBeenCalledWith("/p/1.jsonl");
  expect(document.querySelector('[data-action="session.dispatch-start"]')).toBeNull();
});
