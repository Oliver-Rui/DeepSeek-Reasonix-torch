// @vitest-environment jsdom
import { useState } from "react";
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, createEvent, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { Workspaces } from "./Workspaces";
import type { TreeWorkspace } from "../port/hub";

afterEach(cleanup);

// The head writes the same folded set a row click writes: the harness keeps
// that set the way the window does, so both gestures land in one place.
function draw() {
  const tree: TreeWorkspace[] = [
    { root: "/a", name: "a", remembered: true, sessions: [{ path: "/a/1.jsonl", name: "one", title: "one" }] },
    { root: "/b", name: "b", remembered: true, sessions: [{ path: "/b/2.jsonl", name: "two", title: "two" }] },
    { root: "/auto", name: "auto", remembered: false, auto: true, sessions: [{ path: "/auto/3.jsonl", name: "three", title: "three" }] },
  ];
  const Harness = () => {
    const [folded, setFolded] = useState(new Set<string>());
    const fold = (roots: string[], shut: boolean) => setFolded((prev) => {
      const next = new Set(prev);
      for (const root of roots) { if (shut) next.add(root); else next.delete(root); }
      return next;
    });
    return <Workspaces hub={{} as never} tree={tree} treeRead runtimes={[]} active=""
      folded={folded} onFold={(root, shut) => fold([root], shut)}
      reload={async () => {}} onOpen={async () => {}} onFocus={() => {}} onClose={async () => {}}
      liveIds={() => []} runs={{}} onRename={() => {}} onError={() => {}}
      adder={{ add: () => {}, close: () => {}, at: null } as never} />;
  };
  render(<Harness />);
}

const head = (group: "projects" | "tasks") =>
  document.querySelector<HTMLButtonElement>(`[data-action="workspace.fold-all"][data-value="${group}"]`)!;

const row = (name: string) =>
  [...document.querySelectorAll<HTMLElement>(".wsrow")].find((el) => el.querySelector(".wsname")?.textContent === name)!;

it("folds the whole projects run from the 项目 head and leaves the task group alone", async () => {
  draw();
  const projects = head("projects");
  expect(projects.getAttribute("aria-expanded")).toBe("true");
  expect(projects.getAttribute("aria-label")).toBe("收起所有项目");
  await userEvent.click(projects);
  // The folder rows fold away with their conversations; the task run stays.
  expect(screen.queryByRole("treeitem", { name: "one" })).toBeNull();
  expect(screen.queryByRole("treeitem", { name: "two" })).toBeNull();
  expect(screen.getByRole("treeitem", { name: "three" })).toBeTruthy();
  expect(row("a")).toBeUndefined();
  expect(projects.getAttribute("aria-expanded")).toBe("false");
  expect(projects.getAttribute("aria-label")).toBe("展开所有项目");
  await userEvent.click(projects);
  expect(screen.getByRole("treeitem", { name: "one" })).toBeTruthy();
  expect(screen.getByRole("treeitem", { name: "three" })).toBeTruthy();
});

it("folds the task group from the 任务 head and leaves the projects alone", async () => {
  draw();
  const tasks = head("tasks");
  expect(tasks.getAttribute("aria-expanded")).toBe("true");
  expect(tasks.getAttribute("aria-label")).toBe("收起所有任务");
  await userEvent.click(tasks);
  expect(screen.queryByRole("treeitem", { name: "three" })).toBeNull();
  expect(screen.getByRole("treeitem", { name: "one" })).toBeTruthy();
  expect(tasks.getAttribute("aria-expanded")).toBe("false");
  expect(tasks.getAttribute("aria-label")).toBe("展开所有任务");
  await userEvent.click(tasks);
  expect(screen.getByRole("treeitem", { name: "three" })).toBeTruthy();
});

it("hangs task conversations directly under the 任务 head with no folder row", () => {
  draw();
  expect([...document.querySelectorAll(".wsrow")].some((el) => el.querySelector(".wsname")?.textContent === "auto")).toBe(false);
  expect(screen.getByRole("treeitem", { name: "three" })).toBeTruthy();
});

it("folds a run whose folder holds no session at all", () => {
  const tree: TreeWorkspace[] = [{ root: "/empty", name: "empty", remembered: true, sessions: [] }];
  render(<Workspaces hub={{} as never} tree={tree} treeRead runtimes={[]} active="" folded={new Set()}
    onFold={() => {}} reload={async () => {}} onOpen={async () => {}}
    onFocus={() => {}} onClose={async () => {}} liveIds={() => []} runs={{}} onRename={() => {}}
    onError={() => {}} adder={{ add: () => {}, close: () => {}, at: null } as never} />);
  const projects = head("projects");
  expect(projects.disabled).toBe(false);
  expect(projects.getAttribute("aria-expanded")).toBe("true");
});

// The empty column used to answer with the lone 任务 bar alone; now both runs
// read as runs, each label standing and each saying what it holds.
const labels = () => [...document.querySelectorAll(".wsgroup span")].map((el) => el.textContent);
const notes = () => [...document.querySelectorAll(".ws-none")].map((el) => el.textContent);

function drawTree(tree: TreeWorkspace[]) {
  render(<Workspaces hub={{} as never} tree={tree} treeRead runtimes={[]} active="" folded={new Set()}
    onFold={() => {}} reload={async () => {}} onOpen={async () => {}}
    onFocus={() => {}} onClose={async () => {}} liveIds={() => []} runs={{}} onRename={() => {}}
    onError={() => {}} adder={{ add: () => {}, close: () => {}, at: null } as never} />);
}

it("stands both labels with their empty notes when only the window's own folder exists", () => {
  drawTree([{ root: "/auto", name: "auto", remembered: false, auto: true, sessions: [] }]);
  expect(labels()).toEqual(["项目", "任务"]);
  expect(notes()).toEqual(["暂无项目", "暂无任务"]);
  expect(document.querySelector('[data-action="workspace.add"]')).toBeNull();
});

it("offers the first project under 暂无项目 when no folder exists at all", () => {
  drawTree([]);
  expect(labels()).toEqual(["项目", "任务"]);
  expect(notes()).toEqual(["暂无项目", "暂无任务"]);
  expect(document.querySelector('[data-action="workspace.add"]')).toBeTruthy();
});

it("says 暂无任务 under the standing bar when only projects exist", () => {
  drawTree([{ root: "/a", name: "a", remembered: true, sessions: [{ path: "/a/1.jsonl", name: "one", title: "one" }] }]);
  expect(labels()).toEqual(["项目", "任务"]);
  expect(notes()).toEqual(["暂无任务"]);
});

it("waits for the counts before saying the task run is empty", () => {
  drawTree([{ root: "/auto", name: "auto", remembered: false, auto: true, sessions: [], countsComplete: false }]);
  expect(labels()).toEqual(["项目", "任务"]);
  expect(notes()).toEqual(["暂无项目"]);
});

it("folds the empty task run's note from its standing label", async () => {
  drawTree([{ root: "/auto", name: "auto", remembered: false, auto: true, sessions: [] }]);
  const tasks = head("tasks");
  expect(tasks.disabled).toBe(false);
  expect(tasks.getAttribute("aria-expanded")).toBe("true");
  await userEvent.click(tasks);
  expect(notes()).toEqual(["暂无项目"]);
  expect(tasks.getAttribute("aria-expanded")).toBe("false");
  await userEvent.click(tasks);
  expect(notes()).toEqual(["暂无项目", "暂无任务"]);
});

it("folds 暂无项目 and the first-project offer from the 项目 label", async () => {
  drawTree([]);
  const projects = head("projects");
  expect(projects.disabled).toBe(false);
  await userEvent.click(projects);
  expect(notes()).toEqual(["暂无任务"]);
  expect(document.querySelector('[data-action="workspace.add"]')).toBeNull();
  await userEvent.click(projects);
  expect(document.querySelector('[data-action="workspace.add"]')).toBeTruthy();
  expect(notes()).toEqual(["暂无项目", "暂无任务"]);
});

const both = () => document.querySelector<HTMLButtonElement>('[data-action="workspace.fold-all"][data-value="both"]')!;

it("shows each label's arrow folded to the right and open pointing down", () => {
  draw();
  const projectsChev = head("projects").querySelector(".wschev")!;
  expect(projectsChev.getAttribute("data-open")).toBe("");
  expect(projectsChev.getAttribute("data-leaf")).toBeNull();
  const tasksChev = head("tasks").querySelector(".wschev")!;
  expect(tasksChev.getAttribute("data-open")).toBe("");
});

it("shows the arrow open on a run the label can fold", () => {
  drawTree([{ root: "/empty", name: "empty", remembered: true, sessions: [] }]);
  const chev = head("projects").querySelector(".wschev")!;
  expect(chev.getAttribute("data-open")).toBe("");
  expect(chev.getAttribute("data-leaf")).toBeNull();
});

it("folds both runs at once from the 任务列表 head", async () => {
  draw();
  const bar = both();
  expect(bar.getAttribute("aria-expanded")).toBe("true");
  await userEvent.click(bar);
  expect(screen.queryByRole("treeitem", { name: "one" })).toBeNull();
  expect(screen.queryByRole("treeitem", { name: "three" })).toBeNull();
  expect(notes()).toEqual([]);
  expect(bar.getAttribute("aria-expanded")).toBe("false");
  expect(head("projects").getAttribute("aria-expanded")).toBe("false");
  expect(head("tasks").getAttribute("aria-expanded")).toBe("false");
  expect(head("projects").querySelector(".wschev")!.getAttribute("data-open")).toBeNull();
  await userEvent.click(bar);
  expect(screen.getByRole("treeitem", { name: "one" })).toBeTruthy();
  expect(screen.getByRole("treeitem", { name: "three" })).toBeTruthy();
  expect(bar.getAttribute("aria-expanded")).toBe("true");
});

it("folds both empty runs' notes from the 任务列表 head", async () => {
  drawTree([]);
  await userEvent.click(both());
  expect(notes()).toEqual([]);
  expect(document.querySelector('[data-action="workspace.add"]')).toBeNull();
  await userEvent.click(both());
  expect(notes()).toEqual(["暂无项目", "暂无任务"]);
  expect(document.querySelector('[data-action="workspace.add"]')).toBeTruthy();
});

const GROUP_KIND = "application/x-reasonix-group";
const dragTransfer = () => {
  const transfer = {
    data: "",
    types: [GROUP_KIND],
    effectAllowed: "",
    dropEffect: "",
    setData(_kind: string, value: string) { this.data = value; },
    getData(_kind: string) { return this.data; },
  };
  return transfer;
};

it("swaps the two runs when one label is dragged onto the other", async () => {
  draw();
  const transfer = dragTransfer();
  // The drag rides the whole label row: the carried state and the line sit
  // on the row, not on the fold button itself.
  const lineOf = (button: HTMLButtonElement) => button.closest(".wsgroupline")!;
  const tasks = head("tasks");
  const projects = head("projects");
  fireEvent.dragStart(tasks, { dataTransfer: transfer });
  await waitFor(() => expect(lineOf(tasks).getAttribute("data-dragging")).toBe(""));
  fireEvent.dragOver(projects, { dataTransfer: transfer });
  // The carried run sits below the target, so it lands above it: the line
  // rides the target's top edge, where the run will be.
  await waitFor(() => expect(lineOf(projects).getAttribute("data-drop")).toBe("before"));
  fireEvent.drop(projects, { dataTransfer: transfer });
  expect([...document.querySelectorAll(".wsgroup span")].map((el) => el.textContent)).toEqual(["任务", "项目"]);
  expect(localStorage.getItem("rx-rail-group-order")).toBe("tasks-first");
  // The run the drag carried no longer fades, and no head reads as a target.
  expect(lineOf(tasks).getAttribute("data-dragging")).toBeNull();
  expect(lineOf(projects).getAttribute("data-drop")).toBeNull();
  // The first run is now the tasks run: its conversations lead the tree.
  expect(screen.getAllByRole("treeitem", { name: "three" })[0]!.compareDocumentPosition(screen.getAllByRole("treeitem", { name: "one" })[0]!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  // Dragging back restores the standing order: now the carried run sits
  // above its target, so its line rides the target's bottom edge.
  fireEvent.dragStart(projects, { dataTransfer: transfer });
  fireEvent.dragOver(tasks, { dataTransfer: transfer });
  await waitFor(() => expect(lineOf(tasks).getAttribute("data-drop")).toBe("before"));
  fireEvent.drop(tasks, { dataTransfer: transfer });
  expect([...document.querySelectorAll(".wsgroup span")].map((el) => el.textContent)).toEqual(["项目", "任务"]);
  expect(localStorage.getItem("rx-rail-group-order")).toBe("projects-first");
  localStorage.removeItem("rx-rail-group-order");
});

it("drags the standing labels of an empty tree the same way", () => {
  drawTree([]);
  const transfer = dragTransfer();
  const tasks = head("tasks");
  const projects = head("projects");
  fireEvent.dragStart(tasks, { dataTransfer: transfer });
  fireEvent.drop(projects, { dataTransfer: transfer });
  expect([...document.querySelectorAll(".wsgroup span")].map((el) => el.textContent)).toEqual(["任务", "项目"]);
  expect(notes()).toEqual(["暂无任务", "暂无项目"]);
  localStorage.removeItem("rx-rail-group-order");
});

// A conversation row carries its own kind, so the two drags never cross: the
// label drag swaps runs, the row drag names a slot the container computes.
const SESSION_KIND = "application/x-reasonix-session";
const sessionTransfer = () => {
  const transfer = {
    data: "",
    types: [SESSION_KIND],
    effectAllowed: "",
    dropEffect: "",
    setData(_kind: string, value: string) { this.data = value; },
    getData(_kind: string) { return this.data; },
  };
  return transfer;
};

function drawMoving(hub: unknown, reload: () => Promise<void> = async () => {}) {
  const tree: TreeWorkspace[] = [
    { root: "/a", name: "a", remembered: true, sessions: [
      { path: "/a/1.jsonl", name: "one", title: "one" },
      { path: "/a/2.jsonl", name: "two", title: "two" },
      { path: "/a/3.jsonl", name: "three", title: "three" },
    ] },
    { root: "/b", name: "b", remembered: true, sessions: [{ path: "/b/9.jsonl", name: "nine", title: "nine" }] },
  ];
  render(<Workspaces hub={hub as never} tree={tree} treeRead runtimes={[]} active="" folded={new Set()}
    onFold={() => {}} reload={reload} onOpen={async () => {}}
    onFocus={() => {}} onClose={async () => {}} liveIds={() => []} runs={{}} onRename={() => {}}
    onError={() => {}} adder={{ add: () => {}, close: () => {}, at: null } as never} />);
  // jsdom lays nothing out, so the run's rows are given their boxes: three
  // rows of 30px stacked the way the sidebar draws them.
  for (const [name, top] of [["one", 100], ["two", 130], ["three", 160]] as const) {
    const el = screen.getByRole("treeitem", { name });
    el.getBoundingClientRect = () =>
      ({ top, bottom: top + 30, height: 30, left: 0, right: 200, width: 200, x: 0, y: top, toJSON: () => ({}) } as DOMRect);
  }
}

// The drop events are received by the run's list container, not the row: the
// container reads every row's box and names the slot the pointer sits in.
function pointer(type: "dragOver" | "drop", el: HTMLElement, dt: object, y: number) {
  const ev = createEvent[type](el, { dataTransfer: dt });
  Object.defineProperty(ev, "clientY", { value: y });
  fireEvent(el, ev);
  return ev;
}

const kids = () => document.querySelector<HTMLElement>('[data-reorder-list="/a"]')!;

it("carries a conversation to the slot the pointer named, in one kernel call", async () => {
  let reloaded = 0;
  const placeSession = vi.fn(async () => {});
  drawMoving({ placeSession }, () => { reloaded++; return Promise.resolve(); });
  const transfer = sessionTransfer();
  const one = screen.getByRole("treeitem", { name: "one" });
  const three = screen.getByRole("treeitem", { name: "three" });
  fireEvent.dragStart(one, { dataTransfer: transfer });
  await waitFor(() => expect(one.getAttribute("data-dragging")).toBe(""));
  expect(kids().getAttribute("data-session-drag")).toBe("");
  pointer("dragOver", kids(), transfer, 185); // below three's middle: the run's end
  await waitFor(() => expect(three.getAttribute("data-drop")).toBe("after"));
  pointer("drop", kids(), transfer, 185);
  fireEvent.dragEnd(one, { dataTransfer: transfer });
  // One gesture, one place, one reload — not a walk of kernel swaps.
  await waitFor(() => expect(placeSession).toHaveBeenCalledExactlyOnceWith("/a/1.jsonl", ""));
  expect(reloaded).toBe(1);
  expect(one.getAttribute("data-dragging")).toBeNull();
  expect(three.getAttribute("data-drop")).toBeNull();
});

it("lands a conversation above the target the pointer named", async () => {
  const placeSession = vi.fn(async () => {});
  drawMoving({ placeSession });
  const transfer = sessionTransfer();
  const one = screen.getByRole("treeitem", { name: "one" });
  fireEvent.dragStart(screen.getByRole("treeitem", { name: "three" }), { dataTransfer: transfer });
  expect(pointer("dragOver", kids(), transfer, 105).defaultPrevented).toBe(true); // above one's middle
  await waitFor(() => expect(one.getAttribute("data-drop")).toBe("before"));
  pointer("drop", kids(), transfer, 105);
  await waitFor(() => expect(placeSession).toHaveBeenCalledExactlyOnceWith("/a/3.jsonl", "/a/1.jsonl"));
});

it("the pointer's half names the slot, and the slot the row already fills paints no line", async () => {
  const placeSession = vi.fn(async () => {});
  drawMoving({ placeSession });
  const transfer = sessionTransfer();
  const one = screen.getByRole("treeitem", { name: "one" });
  const three = screen.getByRole("treeitem", { name: "three" });
  fireEvent.dragStart(one, { dataTransfer: transfer });
  pointer("dragOver", kids(), transfer, 135); // two's upper half is the gap one sits in
  await waitFor(() => expect(document.querySelectorAll('[data-drop]')).toHaveLength(0));
  pointer("dragOver", kids(), transfer, 165); // two's lower half: the slot below two
  await waitFor(() => expect(three.getAttribute("data-drop")).toBe("before"));
  pointer("drop", kids(), transfer, 165);
  await waitFor(() => expect(placeSession).toHaveBeenCalledExactlyOnceWith("/a/1.jsonl", "/a/3.jsonl"));
});

it("a row dropped into its own slot still asks nothing", async () => {
  const placeSession = vi.fn(async () => {});
  drawMoving({ placeSession });
  const transfer = sessionTransfer();
  const two = screen.getByRole("treeitem", { name: "two" });
  fireEvent.dragStart(two, { dataTransfer: transfer });
  pointer("dragOver", kids(), transfer, 135);
  pointer("drop", kids(), transfer, 135);
  fireEvent.dragEnd(two, { dataTransfer: transfer });
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(placeSession).not.toHaveBeenCalled();
});

it("ignores a conversation dropped into another project's list", async () => {
  const placeSession = vi.fn(async () => {});
  drawMoving({ placeSession });
  const transfer = sessionTransfer();
  const one = screen.getByRole("treeitem", { name: "one" });
  const nine = screen.getByRole("treeitem", { name: "nine" });
  fireEvent.dragStart(one, { dataTransfer: transfer });
  const ev = fireEvent.dragOver(nine, { dataTransfer: transfer });
  expect(ev).toBe(true); // never offered a slot: the other run's list says nothing
  fireEvent.drop(nine, { dataTransfer: transfer });
  expect(placeSession).not.toHaveBeenCalled();
  expect(nine.getAttribute("data-drop")).toBeNull();
});

it("takes the run's head: the pointer above the first conversation lands before it", async () => {
  // The band over the project row is the run's own head: in the browser the
  // drag there targets the list container, so the test hands it the same
  // pointer height and lets the container name the slot.
  const placeSession = vi.fn(async () => {});
  drawMoving({ placeSession });
  const transfer = sessionTransfer();
  const one = screen.getByRole("treeitem", { name: "one" });
  fireEvent.dragStart(screen.getByRole("treeitem", { name: "three" }), { dataTransfer: transfer });
  expect(pointer("dragOver", kids(), transfer, 20).defaultPrevented).toBe(true);
  await waitFor(() => expect(one.getAttribute("data-drop")).toBe("before"));
  pointer("drop", kids(), transfer, 20);
  await waitFor(() => expect(placeSession).toHaveBeenCalledExactlyOnceWith("/a/3.jsonl", "/a/1.jsonl"));
});

it("takes the run's foot: the pointer under the last conversation lands at the end", async () => {
  const placeSession = vi.fn(async () => {});
  drawMoving({ placeSession });
  const transfer = sessionTransfer();
  const one = screen.getByRole("treeitem", { name: "one" });
  const three = screen.getByRole("treeitem", { name: "three" });
  fireEvent.dragStart(one, { dataTransfer: transfer });
  expect(pointer("dragOver", kids(), transfer, 200).defaultPrevented).toBe(true);
  await waitFor(() => expect(three.getAttribute("data-drop")).toBe("after"));
  pointer("drop", kids(), transfer, 200);
  await waitFor(() => expect(placeSession).toHaveBeenCalledExactlyOnceWith("/a/1.jsonl", ""));
});
