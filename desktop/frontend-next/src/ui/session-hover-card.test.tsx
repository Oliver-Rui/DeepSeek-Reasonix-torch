// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import type { DispatchThread } from "../port/hub";
import { SessionHoverCard, useHoverCard, OPEN_DELAY_MS } from "./SessionHoverCard";
import { peerSourceLabel } from "./source";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

const thread = (over: Partial<DispatchThread> = {}): DispatchThread => ({
  key: "s-member",
  role: "组员B",
  sessionPath: "/sessions/member.jsonl",
  targetTitle: "组员B",
  targetProject: "billing",
  state: "waiting",
  since: "2026-11-19T10:00:00Z",
  ...over,
});

const hubWith = (threads: DispatchThread[]) => ({ dispatchThreads: vi.fn(async () => threads) });
const anchor = { top: 100, right: 260, bottom: 130, left: 0, width: 260, height: 30, x: 0, y: 100 } as DOMRect;
const base = { path: "/sessions/lead.jsonl", title: "组长" };

describe("the session hover card", () => {
  it("names the session and the facts the row cannot show", async () => {
    render(<SessionHoverCard hub={hubWith([])} facts={{ ...base, project: "alpha", root: "D:\\work\\alpha" }} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    expect(screen.getByText("组长")).toBeTruthy();
    expect(screen.getByText("alpha")).toBeTruthy();
    expect(screen.getByText("D:\\work\\alpha")).toBeTruthy();
    await act(async () => { await Promise.resolve(); });
  });

  it("asks the hub for the hovered session's own links", async () => {
    const hub = hubWith([thread()]);
    render(<SessionHoverCard hub={hub} facts={base} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    await screen.findByText(/等待「组员B」回信/);
    expect(hub.dispatchThreads).toHaveBeenCalledWith("/sessions/lead.jsonl");
  });

  it("lists a waiting link as 等待「X」回信 and opens it", async () => {
    const onOpen = vi.fn();
    render(<SessionHoverCard hub={hubWith([thread()])} facts={base} anchor={anchor} onOpen={onOpen} onEnter={vi.fn()} onLeave={vi.fn()} />);
    await userEvent.click(await screen.findByRole("button", { name: /等待「组员B」回信/ }));
    expect(onOpen).toHaveBeenCalledWith("/sessions/member.jsonl");
  });

  it("lists an answered link as 处理「X」的消息", async () => {
    render(<SessionHoverCard hub={hubWith([thread({ state: "replied" })])} facts={base} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    expect(await screen.findByText(/处理「组员B」的消息/)).toBeTruthy();
  });

  it("draws no link block when the hub knows nothing", async () => {
    const { container } = render(<SessionHoverCard hub={hubWith([])} facts={base} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    await act(async () => { await Promise.resolve(); });
    expect(container.querySelector(".hovercard-links")).toBeNull();
  });

  it("survives a hub that refuses the read", async () => {
    const hub = { dispatchThreads: vi.fn(async () => { throw new Error("no"); }) };
    const { container } = render(<SessionHoverCard hub={hub} facts={base} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    await act(async () => { await Promise.resolve(); });
    expect(container.querySelector(".hovercard-links")).toBeNull();
    expect(screen.getByText("组长")).toBeTruthy();
  });

  it("opens after the dwell, not on the first pixel", async () => {
    vi.useFakeTimers();
    const seen: string[] = [];
    let fire: (() => void) | null = null;
    function Probe() {
      const hover = useHoverCard();
      if (hover.open) seen.push(hover.open.key);
      fire = () => hover.enter("row-1", { path: "/sessions/x.jsonl", title: "组长" }, document.body);
      return null;
    }
    render(<Probe />);
    await act(async () => { fire?.(); });
    expect(seen).toEqual([]);
    await act(async () => { vi.advanceTimersByTimeAsync(OPEN_DELAY_MS + 20); });
    expect(seen).toEqual(["row-1"]);
  });

  // The card is fixed-positioned, so where it is in the tree decides what it is
  // positioned against. Rendered inside the sidebar it lands in .app, whose
  // overflow:clip makes that box the containing block: the card then draws at
  // the row's offset inside the box instead of at the row, and a scrolled
  // sidebar pushes it further off. It has to hang off the body.
  it("mounts on the document body, not inside the row", async () => {
    const { container } = render(
      <div className="app" style={{ overflow: "clip" }}>
        <SessionHoverCard hub={hubWith([])} facts={base} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />
      </div>,
    );
    const card = document.querySelector(".hovercard");
    expect(card).toBeTruthy();
    expect(container.contains(card)).toBe(false);
    expect(card!.parentElement).toBe(document.body);
    await act(async () => { await Promise.resolve(); });
  });

  it("sits at the row's own viewport coordinates", async () => {
    render(<SessionHoverCard hub={hubWith([])} facts={base} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
    await act(async () => { await Promise.resolve(); });
    const card = document.querySelector(".hovercard") as HTMLElement;
    // anchor.top is 100 and the card is short in jsdom, so it is not clamped.
    expect(card.style.top).toBe("100px");
    expect(card.style.left).toBe("270px");
  });

  // The window scales itself with zoom on the root element, and a fixed box's
  // top/left are multiplied by it. The anchor is measured in scaled viewport
  // pixels, so an undivided offset puts the card a fifth of the page below its
  // row at the default 1.2 scale — the exact drift this card shipped with.
  it("divides out the window's own scale", async () => {
    const before = document.documentElement.style.zoom;
    document.documentElement.style.zoom = "2";
    try {
      render(<SessionHoverCard hub={hubWith([])} facts={base} anchor={anchor} onOpen={vi.fn()} onEnter={vi.fn()} onLeave={vi.fn()} />);
      await act(async () => { await Promise.resolve(); });
      const card = document.querySelector(".hovercard") as HTMLElement;
      expect(card.style.top).toBe("50px");
      expect(card.style.left).toBe("135px");
    } finally {
      document.documentElement.style.zoom = before;
    }
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
