// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import type { DispatchBoardRow } from "../port/dispatch";
import { DispatchThreadsBar } from "./DispatchThreadsBar";

afterEach(cleanup);

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

function draw(rows: DispatchBoardRow[], onOpen = vi.fn()) {
  const hub = hubWith(rows);
  const view = render(<DispatchThreadsBar hub={hub} onOpen={onOpen} />);
  return { onOpen, hub, container: view.container };
}

describe("the session-collaboration chip", () => {
  it("draws nothing when neither threads nor a board exist", async () => {
    const { container } = draw([]);
    await waitFor(() => expect(container.children).toHaveLength(0));
  });

  it("names the chip and the card", async () => {
    draw([row()]);
    expect(await screen.findByRole("button", { name: "会话协作" })).toBeTruthy();
    expect(screen.getByRole("dialog", { name: /会话协作工作板/ })).toBeTruthy();
  });

  it("shows both ends, the role and the state, grouped into this project", async () => {
    draw([row()]);
    expect(await screen.findByText("组长")).toBeTruthy();
    expect(screen.getByText("组员B · VR面板式")).toBeTruthy();
    expect(screen.getByText("已回信")).toBeTruthy();
    expect(screen.getByText(/本项目/)).toBeTruthy();
  });

  it("groups cross-project dispatches separately", async () => {
    draw([row(), row({ crossProject: true, assignee: { title: "远程组员", sessionPath: "/sessions/remote.jsonl", project: "beta" } })]);
    expect(await screen.findByText(/跨项目/)).toBeTruthy();
  });

  it("opens the assignee's session from its row", async () => {
    const { onOpen } = draw([row()]);
    const target = await screen.findByTitle("/sessions/docs.jsonl");
    await userEvent.click(target);
    expect(onOpen).toHaveBeenCalledExactlyOnceWith("/sessions/docs.jsonl");
  });

  it("uses the unnamed word when the wire gives neither title nor role", async () => {
    draw([row({ assignee: { sessionPath: "/sessions/x.jsonl" }, role: undefined })]);
    expect(await screen.findAllByText(/已派发的会话/)).toBeTruthy();
  });
});
