// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import type { PeerThread } from "../port/boundary";
import { SessionHoverCard, useHoverCard, OPEN_DELAY_MS } from "./SessionHoverCard";
import { peerSourceLabel } from "./source";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

const thread = (over: Partial<PeerThread> = {}): PeerThread => ({
  key: "s-member",
  title: "组员B",
  project: "billing",
  sessionPath: "/sessions/member.jsonl",
  state: "waiting",
  since: "2026-11-19T10:00:00Z",
  messageId: "m1",
  ...over,
});

const anchor = { top: 100, right: 260, bottom: 130, left: 0, width: 260, height: 30, x: 0, y: 100 } as DOMRect;

describe("the session hover card", () => {
  it("names the session and the facts the row cannot show", () => {
    render(<SessionHoverCard facts={{ title: "组长", project: "alpha", root: "D:\\work\\alpha", threads: [] }} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    expect(screen.getByText("组长")).toBeTruthy();
    expect(screen.getByText("alpha")).toBeTruthy();
    expect(screen.getByText("D:\\work\\alpha")).toBeTruthy();
  });

  it("lists a waiting link as 等待「X」回信 and opens it", async () => {
    const onOpen = vi.fn();
    render(<SessionHoverCard facts={{ title: "组长", threads: [thread()] }} anchor={anchor} onOpen={onOpen} onEnter={vi.fn()} onLeave={vi.fn()} />);
    const link = screen.getByRole("button", { name: /等待「组员B」回信/ });
    await userEvent.click(link);
    expect(onOpen).toHaveBeenCalledWith("/sessions/member.jsonl");
  });

  it("lists an answered link as 处理「X」的消息", () => {
    render(<SessionHoverCard facts={{ title: "组长", threads: [thread({ state: "replied" })] }} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    expect(screen.getByText(/处理「组员B」的消息/)).toBeTruthy();
  });

  it("draws no link block when nothing is talking", () => {
    const { container } = render(<SessionHoverCard facts={{ title: "组长" }} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    expect(container.querySelector(".hovercard-links")).toBeNull();
  });

  it("opens after the dwell, not on the first pixel", async () => {
    vi.useFakeTimers();
    const seen: string[] = [];
    let fire: (() => void) | null = null;
    function Probe() {
      const hover = useHoverCard();
      if (hover.open) seen.push(hover.open.key);
      fire = () => hover.enter("row-1", { title: "组长" }, document.body);
      return null;
    }
    render(<Probe />);
    await act(async () => { fire?.(); });
    expect(seen).toEqual([]);
    await act(async () => { vi.advanceTimersByTime(OPEN_DELAY_MS + 20); });
    expect(seen).toEqual(["row-1"]);
  });
});

describe("the relayed-message label", () => {
  it("says 来自「X」 for a handed-over task", () => {
    expect(peerSourceLabel({ sessionId: "s-a", title: "甲" })).toBe("来自「甲」");
  });

  it("says 「X」的回信 for a member's answer", () => {
    expect(peerSourceLabel({ sessionId: "s-a", title: "甲", reply: true })).toBe("「甲」的回信");
  });

  it("says nothing when the host declared no source", () => {
    expect(peerSourceLabel(undefined)).toBe("");
    expect(peerSourceLabel({ sessionId: "" })).toBe("");
  });
});
