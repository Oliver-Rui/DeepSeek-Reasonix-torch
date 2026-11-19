// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import type { DispatchBoardRow } from "../port/dispatch";
import { DispatchThreadsBar } from "./DispatchThreadsBar";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

const row = (over: Partial<DispatchBoardRow> = {}): DispatchBoardRow => ({
  dispatcher: { title: "组长", sessionPath: "/sessions/lead.jsonl", project: "alpha" },
  assignee: { title: "组员B · VR面板式", sessionPath: "/sessions/docs.jsonl", project: "alpha" },
  role: "组员B",
  state: "replied",
  since: "2026-10-16T10:00:00Z",
  crossProject: false,
  ...over,
});

function hubWith(rows: DispatchBoardRow[]) {
  return { dispatchBoard: vi.fn(async () => rows) };
}

function draw(rows: DispatchBoardRow[], onOpen = vi.fn(), active = true) {
  const hub = hubWith(rows);
  const view = render(<DispatchThreadsBar hub={hub} onOpen={onOpen} active={active} />);
  return { onOpen, hub, container: view.container };
}

describe("the session-collaboration chip", () => {
  it("draws nothing when neither threads nor a board exist", async () => {
    const { container } = draw([]);
    await waitFor(() => expect(container.children).toHaveLength(0));
  });

  it("names the chip and the card", async () => {
    draw([row()]);
    expect(await screen.findByRole("button", { name: "会话互聊" })).toBeTruthy();
    expect(screen.getByRole("dialog", { name: /会话互聊工作板/ })).toBeTruthy();
  });

  it("shows both ends, the role and the state, grouped into this project", async () => {
    draw([row()]);
    expect(await screen.findByText("组长")).toBeTruthy();
    expect(screen.getByText("组员B · VR面板式")).toBeTruthy();
    expect(screen.getByText("已回信")).toBeTruthy();
    expect(screen.getByText(/本项目/)).toBeTruthy();
  });

  it("groups cross-project dispatches separately and names both projects", async () => {
    draw([row(), row({ crossProject: true, dispatcher: { title: "组长", sessionPath: "/sessions/lead.jsonl", project: "alpha" }, assignee: { title: "远程组员", sessionPath: "/sessions/remote.jsonl", project: "beta" } })]);
    expect(await screen.findByText(/跨项目/)).toBeTruthy();
    expect(screen.getByText("alpha → beta")).toBeTruthy();
  });

  it("keeps the full name behind the ellipsis", async () => {
    draw([row()]);
    const target = await screen.findByRole("button", { name: "组员B · VR面板式" });
    expect(target.getAttribute("title")).toBe("组员B · VR面板式");
  });

  it("gives the dispatch a stable time element", async () => {
    const { container } = draw([row()]);
    await screen.findByText("组员B · VR面板式");
    const time = container.querySelector('time[datetime="2026-10-16T10:00:00Z"]');
    expect(time).toBeTruthy();
  });

  it("opens the assignee's session from its row", async () => {
    const { onOpen } = draw([row()]);
    const target = await screen.findByRole("button", { name: "组员B · VR面板式" });
    await userEvent.click(target);
    expect(onOpen).toHaveBeenCalledExactlyOnceWith("/sessions/docs.jsonl");
  });

  it("uses the unnamed word when the wire gives neither title nor role", async () => {
    draw([row({ assignee: { sessionPath: "/sessions/x.jsonl" }, role: undefined })]);
    expect(await screen.findAllByText(/已派发的会话/)).toBeTruthy();
  });

  it("does not read while its pane is not visible", async () => {
    vi.useFakeTimers();
    const hub = hubWith([row()]);
    render(<DispatchThreadsBar hub={hub} active={false} />);
    await act(async () => { vi.advanceTimersByTime(30_000); });
    expect(hub.dispatchBoard).not.toHaveBeenCalled();
  });

  it("waits for the previous read before starting another", async () => {
    vi.useFakeTimers();
    let release: ((rows: DispatchBoardRow[]) => void) | undefined;
    const hub = {
      dispatchBoard: vi.fn(() => new Promise<DispatchBoardRow[]>((resolve) => {
        if (release) { resolve([row()]); return; }
        release = resolve;
      })),
    };
    const view = render(<DispatchThreadsBar hub={hub} active />);
    expect(hub.dispatchBoard).toHaveBeenCalledTimes(1);
    await act(async () => { vi.advanceTimersByTime(30_000); });
    expect(hub.dispatchBoard).toHaveBeenCalledTimes(1);
    await act(async () => { release?.([row()]); await Promise.resolve(); });
    await act(async () => { vi.advanceTimersByTime(10_000); });
    expect(hub.dispatchBoard).toHaveBeenCalledTimes(2);
    view.unmount();
  });
});
