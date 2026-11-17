// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import "./testkit";
import { SessionDispatch } from "./SessionDispatch";
import type { DispatchResult, DispatchTarget } from "../port/dispatch";

HTMLDialogElement.prototype.showModal = function () { this.open = true; };
HTMLDialogElement.prototype.close = function () { this.open = false; };
afterEach(cleanup);

const from = { path: "/w/s/a.jsonl", title: "alpha" };
const projects = [{ root: "/p", name: "p" }, { root: "/q", name: "q" }];
const sessions = [{ path: "/p/1.jsonl", title: "one", project: "p" }, { path: "/q/2.jsonl", title: "two", project: "q" }];

function draw(dispatch = vi.fn(async (_from: string, _targets: DispatchTarget[]): Promise<DispatchResult[]> => [{ ok: true }, { ok: true }])) {
  const onClose = vi.fn();
  const onSent = vi.fn();
  render(<SessionDispatch from={from} projects={projects} sessions={sessions}
    hub={{ dispatch } as never} onClose={onClose} onSent={onSent} />);
  return { dispatch, onClose, onSent };
}

const text = () => screen.getByPlaceholderText("把这项任务交给下面的会话…") as HTMLTextAreaElement;

it("starts with one target row and a disabled send until there is a task", () => {
  const { dispatch } = draw();
  expect(screen.getByRole("button", { name: "派发任务" })).toBeInstanceOf(HTMLButtonElement);
  expect((screen.getByRole("button", { name: "派发任务" }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.change(text(), { target: { value: "ship the report" } });
  expect((screen.getByRole("button", { name: "派发任务" }) as HTMLButtonElement).disabled).toBe(false);
  expect(dispatch).not.toHaveBeenCalled();
});

it("sends one task to each target with its own role and mode", async () => {
  const { dispatch, onSent } = draw();
  fireEvent.change(text(), { target: { value: "ship the report" } });
  // the first row: new session in project q, with a role and instructions
  fireEvent.change(screen.getByLabelText("项目"), { target: { value: "/q" } });
  fireEvent.change(screen.getByLabelText("角色"), { target: { value: "美术" } });
  fireEvent.change(screen.getByLabelText("角色要求"), { target: { value: "负责海报" } });
  fireEvent.click(screen.getByRole("button", { name: "再加一个目标" }));
  fireEvent.change(screen.getAllByLabelText("目标方式")[1], { target: { value: "existing" } });
  fireEvent.change(screen.getByLabelText("已有会话"), { target: { value: "/p/1.jsonl" } });
  fireEvent.click(screen.getByRole("button", { name: "派发任务" }));
  await waitFor(() => expect(dispatch).toHaveBeenCalledOnce());
  expect(dispatch.mock.calls[0][0]).toBe(from.path);
  expect(dispatch.mock.calls[0][1]).toEqual([
    { mode: "new", sessionPath: undefined, project: "/q", role: "美术", instructions: "负责海报", text: "ship the report" },
    { mode: "existing", sessionPath: "/p/1.jsonl", project: undefined, role: "", instructions: undefined, text: "ship the report" },
  ]);
  await waitFor(() => expect(onSent).not.toHaveBeenCalled());
  fireEvent.click(screen.getByRole("button", { name: "完成" }));
  expect(onSent).toHaveBeenCalledOnce();
});

it("skips incomplete rows and shows per-target failures", async () => {
  const dispatch = vi.fn(async (_from: string, targets: DispatchTarget[]): Promise<DispatchResult[]> => targets.map(() => ({ ok: false, refusal: { code: "session.dispatch_self", message: "不能派发给自己" } })));
  draw(dispatch);
  fireEvent.change(text(), { target: { value: "hello" } });
  fireEvent.change(screen.getByLabelText("项目"), { target: { value: "/p" } });
  fireEvent.click(screen.getByRole("button", { name: "再加一个目标" }));
  // the second row is left blank: it must not be sent
  fireEvent.click(screen.getByRole("button", { name: "派发任务" }));
  await waitFor(() => expect(dispatch).toHaveBeenCalledOnce());
  expect(dispatch.mock.calls[0][1]).toHaveLength(1);
  expect(screen.getByText("会话不能把任务派发给自己")).toBeTruthy();
  expect((screen.getByRole("button", { name: "派发任务" }) as HTMLButtonElement)).toBeTruthy();
});

it("never sends twice while a dispatch is in flight", async () => {
  let release!: (value: { ok: boolean }[]) => void;
  const dispatch = vi.fn(() => new Promise<{ ok: boolean }[]>((resolve) => { release = resolve; }));
  draw(dispatch);
  fireEvent.change(text(), { target: { value: "hello" } });
  fireEvent.change(screen.getByLabelText("项目"), { target: { value: "/p" } });
  const send = () => fireEvent.click(screen.getByRole("button", { name: "派发任务" }));
  send();
  send();
  expect(dispatch).toHaveBeenCalledTimes(1);
  release([{ ok: true }]);
  await waitFor(() => expect(screen.getByText("已派出")).toBeTruthy());
});
