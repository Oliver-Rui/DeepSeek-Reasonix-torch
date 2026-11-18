import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { t } from "../i18n";
import type { DispatchBoardRow } from "../port/dispatch";
import type { HubPort } from "../port/hub";
import { useDispatchBoard } from "./useDispatchBoard";
import { useEscape } from "./dismiss";
import { listenAction } from "./listen";
import { StudioIcon } from "./StudioIcon";

const STATE_KEY: Record<DispatchBoardRow["state"], string> = {
  waiting: "chat.dispatch_board_state_waiting",
  working: "chat.dispatch_board_state_working",
  replied: "chat.dispatch_board_state_replied",
  declined: "chat.dispatch_board_state_declined",
};

/** A window-level read of every task one session handed to another, in this
 *  project and across projects. Both ends are named and carry their state, so
 *  a lead can see who owns what without opening each conversation. */
export function DispatchBoard({ hub, onOpen }: { hub: HubPort; onOpen: (path: string) => void }) {
  const [open, setOpen] = useState(false);
  const [seedPath, setSeedPath] = useState("");
  const card = useRef<HTMLDivElement>(null);
  const { rows, failed } = useDispatchBoard(hub, open ? 3000 : 0);
  useEscape(open, () => setOpen(false));

  useEffect(() => listenAction(window, "click", { action: "session.board", listener: (() => {
    setSeedPath("");
    setOpen(true);
  }) as EventListener }), []);
  useEffect(() => {
    if (!open) return;
    const outside = (event: MouseEvent) => {
      if (!card.current?.contains(event.target as Node) && !(event.target as HTMLElement)?.closest?.('[data-action="session.board"]')) setOpen(false);
    };
    window.addEventListener("mousedown", outside);
    return () => window.removeEventListener("mousedown", outside);
  }, [open]);

  if (!open) return null;
  const here = rows.filter((r) => !r.crossProject && (!seedPath || r.dispatcher.sessionPath === seedPath || r.assignee.sessionPath === seedPath));
  const across = rows.filter((r) => r.crossProject && (!seedPath || r.dispatcher.sessionPath === seedPath || r.assignee.sessionPath === seedPath));
  const row = (r: DispatchBoardRow, i: number) => {
    const state = t(STATE_KEY[r.state]);
    const go = (path?: string) => { if (path) { setOpen(false); onOpen(path); } };
    return (
      <div className="boardrow" key={`${r.dispatcher.sessionPath}-${r.assignee.sessionPath}-${i}`}>
        <button type="button" className="boardparty" title={r.dispatcher.sessionPath} onClick={() => go(r.dispatcher.sessionPath)}>
          <span className="boardwho">{r.dispatcher.title || r.role || t("chat.dispatch_thread_unnamed")}</span>
          <small>{r.dispatcher.project || ""}</small>
        </button>
        <span className="boardarrow" aria-hidden="true">→</span>
        <button type="button" className="boardparty" title={r.assignee.sessionPath} onClick={() => go(r.assignee.sessionPath)}>
          <span className="boardwho">{r.assignee.title || r.role || t("chat.dispatch_thread_unnamed")}</span>
          <small>{r.assignee.project || ""}</small>
        </button>
        {r.role && <span className="boardrole" title={t("chat.dispatch_task")}>{r.role}</span>}
        <span className="peerthread" data-state={r.state} title={state} aria-label={state}><i className="peerthread-lamp" aria-hidden="true" /></span>
        <span className="boardstate">{state}</span>
      </div>
    );
  };
  const section = (title: string, list: DispatchBoardRow[]) => list.length === 0 ? null : (
    <section className="boardgroup">
      <h3>{title}</h3>
      <div role="table">
        <div className="boardhead" role="row">
          <span role="columnheader">{t("chat.dispatch_board_dispatcher")}</span>
          <span role="columnheader">{t("chat.dispatch_board_assignee")}</span>
          <span role="columnheader">{t("chat.dispatch_task")}</span>
          <span role="columnheader">{t("chat.find_scope_reply")}</span>
        </div>
        {list.map(row)}
      </div>
    </section>
  );
  return createPortal(
    <div className="dispatchboard-mask">
      <div className="dispatchboard" ref={card} role="dialog" aria-label={t("chat.dispatch_board_title")}>
        <header>
          <b>{t("chat.dispatch_board_title")}</b>
          <button type="button" className="workspace-palette-close" data-action="session.board" aria-label={t("common.close")} onClick={() => setOpen(false)}><StudioIcon name="close" /></button>
        </header>
        <div className="dispatchboard-body">
          {failed && <p className="boardhint">{t("remote.could_not_read_the_state")}</p>}
          {rows.length === 0 && !failed && <p className="boardhint">{t("chat.dispatch_board_empty")}</p>}
          {section(t("chat.dispatch_board_this_project"), here)}
          {section(t("chat.dispatch_board_cross_project"), across)}
        </div>
      </div>
    </div>,
    document.body,
  );
}
