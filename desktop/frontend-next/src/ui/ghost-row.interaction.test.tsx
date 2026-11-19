// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import "./testkit";
import { Workspaces } from "./Workspaces";
import type { TreeWorkspace } from "../port/hub";

afterEach(cleanup);

// A ghost row is a removed project whose archive is still shown. The kernel
// sends it only where the archive is the view — the Archived scope and search
// results — so the everyday list stays clean. The row exists so the archive
// stays reachable, and it offers the way back.
const ghost: TreeWorkspace = {
  root: "/gone", name: "gone", remembered: true, removed: true, archived: 1, total: 1,
  sessions: [{ path: "/gone/sessions/old.jsonl", name: "old", title: "Old work", archived: true }],
};

function tree(ws: TreeWorkspace, hub: Record<string, unknown> = {}, reload = vi.fn(async () => {}), scope = "all") {
  return <Workspaces hub={hub as never} tree={[ws]} treeRead runtimes={[]} active="" folded={new Set()}
    onFold={() => {}} reload={reload} onOpen={vi.fn(async () => {})} onFocus={() => {}} onClose={async () => {}}
    liveIds={() => []} runs={{}} onRename={() => {}} onError={() => {}}
    adder={{ add: () => {}, close: () => {}, at: null } as never} scope={scope as never} />;
}

it("shows a ghost row's archived conversations in the Archived scope", () => {
  render(tree(ghost, {}, vi.fn(async () => {}), "archived"));
  expect(screen.getByText("gone")).toBeTruthy();
  expect(screen.getByText("已移除")).toBeTruthy();
  expect(screen.getByText("Old work")).toBeTruthy();
});

it("a live conversation keeps a ghost row in the everyday list", () => {
  const live: TreeWorkspace = {
    ...ghost,
    sessions: [{ path: "/gone/sessions/now.jsonl", name: "now", title: "Current work" }],
  };
  render(tree(live));
  expect(screen.getByText("gone")).toBeTruthy();
  expect(screen.getByText("Current work")).toBeTruthy();
});

it("offers the way back instead of a second removal", () => {
  const hub = { addWorkspace: vi.fn(async () => ({ ...ghost, removed: false })) };
  const reload = vi.fn(async () => {});
  const { container } = render(tree(ghost, hub, reload, "archived"));
  fireEvent.click(container.querySelector(".wsmore") as HTMLElement);
  const back = screen.getByText("放回列表");
  expect(back).toBeTruthy();
  fireEvent.click(back);
  expect(hub.addWorkspace).toHaveBeenCalledWith("/gone");
});

it("an ordinary project still reads remove, not restore", () => {
  const ws: TreeWorkspace = { root: "/here", name: "here", remembered: true, sessions: [], total: 0 };
  const { container } = render(tree(ws));
  fireEvent.click(container.querySelector(".wsmore") as HTMLElement);
  expect(screen.getByText("移除")).toBeTruthy();
  expect(screen.queryByText("放回列表")).toBeNull();
});
