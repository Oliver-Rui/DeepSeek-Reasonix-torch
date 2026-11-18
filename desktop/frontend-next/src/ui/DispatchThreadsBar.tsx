import { useCallback, useEffect, useRef, useState } from "react";
import { t } from "../i18n";
import type { AgentPort } from "../port/port";
import type { DispatchBoardRow } from "../port/dispatch";
import type { HubPort } from "../port/hub";
import { usePeers } from "./usePeers";
import { StudioIcon } from "./StudioIcon";

const POLL_MS = 3000;

const STATE_KEY: Record<DispatchBoardRow["state"], string> = {
  waiting: "chat.dispatch_board_state_waiting",
  working: "chat.dispatch_board_state_working",
  replied: "chat.dispatch_board_state_replied",
  declined: "chat.dispatch_board_state_declined",
};

/** The sessions this conversation dispatched tasks to, as one quiet chip in
 *  the meter rail beside 对话轮次. Hovering or focusing it opens a small card —
 *  anchored above the chip, never a full-screen layer — listing every dispatch
 *  on this machine with both ends named and the state lamp, grouped into this
 *  project and cross-project. */
export function DispatchThreadsBar({ port, hub, onOpen }: { port: AgentPort; hub?: Pick<HubPort, "dispatchBoard">; onOpen?: (path: string) => void }) {
  const { state } = usePeers(port, POLL_MS);
  const threads = state.enabled ? (state.threads ?? []) : [];
  const [rows, setRows] = useState<DispatchBoardRow[]>([]);
  const alive = useRef(true);
  const refresh = useCallback(async () => {
    if (!hub) return;
    try { setRows(await hub.dispatchBoard()); } catch { /* the chip keeps its last view */ }
  }, [hub]);
  useEffect(() => {
    alive.current = true;
    void refresh();
    const timer = setInterval(() => { if (typeof document === "undefined" || !document.hidden) void refresh(); }, POLL_MS);
    return () => { alive.current = false; clearInterval(timer); };
  }, [refresh]);

  if (threads.length === 0 && rows.length === 0) return null;
  const count = Math.max(threads.length, rows.length);
  const renderRow = (r: DispatchBoardRow, i: number) => {
    const stateText = t(STATE_KEY[r.state]);
    const go = (path?: string) => { if (path) onOpen?.(path); };
    return (
      <div className="peerrow" key={`${r.dispatcher.sessionPath}-${r.assignee.sessionPath}-${i}`}>
        <button type="button" className="peername" title={r.dispatcher.sessionPath} onClick={() => go(r.dispatcher.sessionPath)}>{r.dispatcher.title || r.role || t("chat.dispatch_thread_unnamed")}</button>
        <span className="boardarrow" aria-hidden="true">→</span>
        <button type="button" className="peerstate-name" title={r.assignee.sessionPath} onClick={() => go(r.assignee.sessionPath)}>{r.assignee.title || r.role || t("chat.dispatch_thread_unnamed")}</button>
        {r.role && <span className="peerrole" title={r.role}>{r.role}</span>}
        <span className="peerthread" data-state={r.state} title={stateText} aria-label={stateText}><i className="peerthread-lamp" aria-hidden="true" /></span>
        <span className="peerstate">{stateText}</span>
      </div>
    );
  };
  const here = rows.filter((r) => !r.crossProject);
  const across = rows.filter((r) => r.crossProject);
  const group = (title: string, list: DispatchBoardRow[]) => list.length === 0 ? null : (
    <div className="peergroup"><h4>{title}</h4>{list.map(renderRow)}</div>
  );
  return (
    <span className="studio-peers-anchor">
      <button type="button" className="studio-meter-peers" data-action="metrics.details" data-value="peers" aria-haspopup="dialog"
        aria-label={t("chat.dispatch_threads_title")} title={t("chat.dispatch_threads_count", { n: count })}>
        <StudioIcon name="send" /><span>{t("chat.dispatch_threads_title")}</span><b>{count}</b>
      </button>
      <div className="studio-peers-detail" role="dialog" aria-label={t("chat.dispatch_board_title")}>
        <header><b>{t("chat.dispatch_board_title")}</b><small>{t("chat.dispatch_threads_count", { n: count })}</small></header>
        {rows.length === 0 && <p className="peerhint">{t("chat.dispatch_board_empty")}</p>}
        {group(t("chat.dispatch_board_this_project"), here)}
        {group(t("chat.dispatch_board_cross_project"), across)}
      </div>
    </span>
  );
}
