import { useCallback, useEffect, useRef, useState } from "react";
import { t } from "../i18n";
import type { DispatchBoardRow } from "../port/dispatch";
import type { HubPort } from "../port/hub";
import { StudioIcon } from "./StudioIcon";
import { ageWord } from "./age";
import { useStatusPoll } from "./useStatusPoll";

const POLL_MS = 10_000;

const STATE_KEY: Record<DispatchBoardRow["state"], string> = {
  waiting: "chat.dispatch_board_state_waiting",
  working: "chat.dispatch_board_state_working",
  replied: "chat.dispatch_board_state_replied",
  declined: "chat.dispatch_board_state_declined",
};

function shortProject(path?: string): string {
  const trimmed = (path ?? "").replace(/[\\/]+$/, "");
  if (!trimmed) return "";
  const at = Math.max(trimmed.lastIndexOf("/"), trimmed.lastIndexOf("\\"));
  return at >= 0 ? trimmed.slice(at + 1) : trimmed;
}

function projectLabel(row: DispatchBoardRow): string {
  const from = shortProject(row.dispatcher.project);
  const to = shortProject(row.assignee.project);
  if (row.crossProject && from && to) return `${from} → ${to}`;
  return to || from;
}

function projectTitle(row: DispatchBoardRow): string {
  return [row.dispatcher.project, row.assignee.project].filter(Boolean).join(" → ");
}

function boardTime(since?: string): { text: string; title: string } | null {
  if (!since) return null;
  const at = Date.parse(since);
  if (!Number.isFinite(at)) return null;
  const text = ageWord(at, Date.now());
  return text ? { text, title: new Date(at).toLocaleString() } : null;
}

/** The sessions this conversation dispatched tasks to, as one quiet chip in
 *  the meter rail beside 对话轮次. Hovering or focusing it opens a small card —
 *  anchored above the chip, never a full-screen layer — listing every dispatch
 *  on this machine with both ends named and the state lamp, grouped into this
 *  project and cross-project. */
export function DispatchThreadsBar({ hub, onOpen, active = true }: { hub?: Pick<HubPort, "dispatchBoard">; onOpen?: (path: string) => void; active?: boolean }) {
  const [rows, setRows] = useState<DispatchBoardRow[]>([]);
  const alive = useRef(true);
  const refresh = useCallback(async () => {
    if (!hub) return;
    if (typeof document !== "undefined" && document.hidden) return;
    try {
      const next = await hub.dispatchBoard();
      if (alive.current) setRows(next);
    } catch { /* the chip keeps its last view */ }
  }, [hub]);
  useEffect(() => {
    alive.current = true;
    return () => { alive.current = false; };
  }, []);
  useStatusPoll(active, refresh, POLL_MS);

  // The board is the authority for what is still live: the hub drops archived
  // and removed ends there, so a chip driven by it disappears with them.
  if (rows.length === 0) return null;
  const count = rows.length;
  const renderRow = (r: DispatchBoardRow, i: number) => {
    const stateKey = (STATE_KEY as Record<string, string>)[r.state];
    const stateText = stateKey ? t(stateKey) : r.state;
    const dispatcherName = r.dispatcher.title || t("chat.dispatch_thread_unnamed");
    const assigneeName = r.assignee.title || r.role || t("chat.dispatch_thread_unnamed");
    const dispatcherHint = r.dispatcher.title || r.dispatcher.sessionPath || dispatcherName;
    const assigneeHint = r.assignee.title || r.assignee.sessionPath || assigneeName;
    const project = projectLabel(r);
    const when = boardTime(r.since);
    const go = (path?: string) => { if (path) onOpen?.(path); };
    return (
      <div className="peerrow" key={`${r.dispatcher.sessionPath}-${r.assignee.sessionPath}-${i}`}>
        <button type="button" className="peername" title={dispatcherHint} onClick={() => go(r.dispatcher.sessionPath)}>{dispatcherName}</button>
        <span className="boardarrow" aria-hidden="true">→</span>
        <button type="button" className="peerstate-name" title={assigneeHint} onClick={() => go(r.assignee.sessionPath)}>{assigneeName}</button>
        <span className="peerrow-meta">
          {r.role && <span className="peerrole" title={r.role}>{r.role}</span>}
          {project && <span className="peerproject" title={projectTitle(r) || project}>{project}</span>}
          <span className="peerthread" data-state={r.state} title={stateText} aria-label={stateText}><i className="peerthread-lamp" aria-hidden="true" /></span>
          <span className="peerstate">{stateText}</span>
          {when && <time className="peertime" dateTime={r.since} title={when.title}>{when.text}</time>}
        </span>
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
