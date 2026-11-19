import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { t } from "../i18n";
import type { DispatchThread, HubPort } from "../port/hub";
import { ageWord } from "./age";

const OPEN_DELAY_MS = 320;
const CLOSE_DELAY_MS = 140;

/** What a session row's hover card needs about the session it belongs to. The
 *  chat links are not here: they belong to the row being hovered, not to the
 *  session looking at it, so the card asks the hub for them itself. */
export interface HoverFacts {
  path: string;
  title?: string;
  project?: string;
  root?: string;
  updatedAt?: number;
}

/** The card a sidebar row opens on hover, worded and shaped like the shipped
 *  product's: the title, the project, the folder, when it last moved, then one
 *  line per chat link — 等待「X」回信 for a task handed out, 处理「X」的消息 for one
 *  received — each opening the other session. It is anchored beside the row and
 *  never covers the conversation.
 *
 *  The links are the hovered session's own record, read once when the card
 *  opens. The catalog behind the read is shared and cached, so a hover costs a
 *  map lookup rather than a directory walk. */
export function SessionHoverCard({ hub, facts, anchor, onOpen, onEnter, onLeave }: {
  hub?: Pick<HubPort, "dispatchThreads">;
  facts: HoverFacts;
  anchor: DOMRect;
  onOpen: (path: string) => void;
  onEnter: () => void;
  onLeave: () => void;
}) {
  const card = useRef<HTMLDivElement>(null);
  const [at, setAt] = useState({ left: 0, top: 0 });
  const [threads, setThreads] = useState<DispatchThread[]>([]);

  useEffect(() => {
    const width = 288;
    const height = card.current?.offsetHeight ?? 160;
    // The window applies its own scale as zoom on the root element, and a fixed
    // box's top/left are multiplied by it. The anchor is already measured in
    // scaled viewport pixels, so the offset has to be divided back out or the
    // card lands a fifth of the page below the row it describes. Every fixed
    // surface in this tree divides by --zoom for the same reason; this reads the
    // live value so a change of scale is followed without a reload.
    const zoom = Number.parseFloat(getComputedStyle(document.documentElement).zoom) || 1;
    const left = Math.min(anchor.right + 10, window.innerWidth - width - 12) / zoom;
    const top = Math.max(12, Math.min(anchor.top, window.innerHeight - height - 12)) / zoom;
    setAt({ left, top });
  }, [anchor]);

  useEffect(() => {
    if (!hub || !facts.path) return;
    let alive = true;
    Promise.resolve(hub.dispatchThreads(facts.path))
      .then((got) => { if (alive) setThreads(got ?? []); })
      .catch(() => { if (alive) setThreads([]); });
    return () => { alive = false; };
  }, [hub, facts.path]);

  const stamp = facts.updatedAt ? ageWord(facts.updatedAt, Date.now()) : "";

  // The card is positioned against the viewport, so it has to be a child of the
  // document body. Rendered where the row lives it would sit inside `.app`,
  // whose overflow:clip makes it the containing block for a fixed child — the
  // card then lands at the row's offset inside that box rather than at the row,
  // and a scrolled sidebar pushes it further off. A portal keeps the card on the
  // row it describes whatever the sidebar is doing.
  const body = typeof document === "undefined" ? null : document.body;
  if (!body) return null;
  return createPortal(
    <div className="hovercard" role="dialog" aria-label={facts.title || t("chat.dispatch_thread_unnamed")}
      ref={card} style={{ left: at.left, top: at.top }}
      onMouseEnter={onEnter} onMouseLeave={onLeave}>
      <b className="hovercard-title">{facts.title || t("chat.dispatch_thread_unnamed")}</b>
      {facts.project && <div className="hovercard-line"><span>{t("common.project")}</span>{facts.project}</div>}
      {facts.root && <div className="hovercard-line hovercard-path" title={facts.root}>{facts.root}</div>}
      {stamp && <div className="hovercard-line"><span>{t("chat.hover_updated_at")}</span>{stamp}</div>}
      {threads.length > 0 && (
        <div className="hovercard-links">
          {threads.map((th, i) => {
            const peer = th.targetTitle?.trim() || th.role?.trim() || t("chat.dispatch_thread_unnamed");
            const outbound = th.state === "waiting" || th.state === "working";
            const text = outbound
              ? t("chat.peer_link_waiting", { peer })
              : t("chat.peer_link_incoming", { peer });
            const go = th.sessionPath;
            return go ? (
              <button key={go || i} type="button" className="hovercard-link" data-action="peer.source-open" data-session-path={go}
                title={t("chat.peer_source_open", { title: peer })} onClick={() => onOpen(go)}>
                <span className="hovercard-dot" data-direction={outbound ? "outbound" : "inbound"} aria-hidden="true" />
                <span>{text}</span>
              </button>
            ) : (
              <div key={i} className="hovercard-link">
                <span className="hovercard-dot" aria-hidden="true" /><span>{text}</span>
              </div>
            );
          })}
        </div>
      )}
    </div>,
    body,
  );
}

/** Hover state for one row: the timer that opens the card, and the facts it
 *  reads. Kept in a hook so the row stays a row and the sidebar keeps one
 *  owner for "which card is open". */
export function useHoverCard() {
  const [open, setOpen] = useState<{ key: string; facts: HoverFacts; anchor: DOMRect } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const leave = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const clear = () => {
    clearTimeout(timer.current);
    clearTimeout(leave.current);
  };

  const enter = (key: string, facts: HoverFacts, el: HTMLElement) => {
    clear();
    timer.current = setTimeout(() => setOpen({ key, facts, anchor: el.getBoundingClientRect() }), OPEN_DELAY_MS);
  };

  const exit = () => {
    clearTimeout(timer.current);
    leave.current = setTimeout(() => setOpen(null), CLOSE_DELAY_MS);
  };

  const keep = () => {
    clearTimeout(leave.current);
  };

  useEffect(() => clear, []);

  return { open, enter, exit, keep, close: () => setOpen(null) };
}

export { OPEN_DELAY_MS, CLOSE_DELAY_MS };
