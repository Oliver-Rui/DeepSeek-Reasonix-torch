import { useCallback, useEffect, useRef, useState } from "react";
import { t } from "../i18n";
import type { PeerThread } from "../port/boundary";
import { ageWord } from "./age";

const OPEN_DELAY_MS = 320;
const CLOSE_DELAY_MS = 140;

/** What a session row's hover card needs about one session: the facts the row
 *  itself cannot show, plus the chat links the row is in the middle of. */
export interface HoverFacts {
  title?: string;
  project?: string;
  root?: string;
  updatedAt?: number;
  /** The other sessions this one is talking to, as the peer ledger folds them. */
  threads?: PeerThread[];
}

/** The card a sidebar row opens on hover, worded and shaped like the shipped
 *  product's: the title, the project, the folder, when it last moved, then one
 *  line per chat link — 等待「X」回信 for a task handed out, 处理「X」的消息 for one
 *  received — each opening the other session. It is anchored beside the row and
 *  never covers the conversation. */
export function SessionHoverCard({ facts, anchor, onOpen, onEnter, onLeave }: {
  facts: HoverFacts;
  anchor: DOMRect;
  onOpen: (path: string) => void;
  onEnter: () => void;
  onLeave: () => void;
}) {
  const card = useRef<HTMLDivElement>(null);
  const [at, setAt] = useState({ left: 0, top: 0 });

  useEffect(() => {
    const width = 288;
    const height = card.current?.offsetHeight ?? 160;
    const left = Math.min(anchor.right + 10, window.innerWidth - width - 12);
    const top = Math.max(12, Math.min(anchor.top, window.innerHeight - height - 12));
    setAt({ left, top });
  }, [anchor]);

  const stamp = facts.updatedAt ? ageWord(facts.updatedAt, Date.now()) : "";
  const threads = facts.threads ?? [];

  return (
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
            const peer = th.title?.trim() || t("chat.dispatch_thread_unnamed");
            const text = th.state === "waiting" || th.state === "working"
              ? t("chat.peer_link_waiting", { peer })
              : t("chat.peer_link_incoming", { peer });
            const go = th.sessionPath;
            return go ? (
              <button key={th.key ?? i} type="button" className="hovercard-link" data-action="peer.source-open" data-session-path={go}
                title={t("chat.peer_source_open", { title: peer })} onClick={() => onOpen(go)}>
                <span className="hovercard-dot" data-direction={th.state === "waiting" || th.state === "working" ? "outbound" : "inbound"} aria-hidden="true" />
                <span>{text}</span>
              </button>
            ) : (
              <div key={th.key ?? i} className="hovercard-link">
                <span className="hovercard-dot" aria-hidden="true" /><span>{text}</span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

/** Hover state for one row: the timer that opens the card, and the facts it
 *  reads. Kept in a hook so the row stays a row and the sidebar keeps one
 *  owner for "which card is open". */
export function useHoverCard() {
  const [open, setOpen] = useState<{ key: string; facts: HoverFacts; anchor: DOMRect } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const leave = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const clear = useCallback(() => {
    clearTimeout(timer.current);
    clearTimeout(leave.current);
  }, []);

  const enter = useCallback((key: string, facts: HoverFacts, el: HTMLElement) => {
    clear();
    timer.current = setTimeout(() => setOpen({ key, facts, anchor: el.getBoundingClientRect() }), OPEN_DELAY_MS);
  }, [clear]);

  const exit = useCallback(() => {
    clearTimeout(timer.current);
    leave.current = setTimeout(() => setOpen(null), CLOSE_DELAY_MS);
  }, []);

  const keep = useCallback(() => {
    clearTimeout(leave.current);
  }, []);

  useEffect(() => clear, [clear]);

  return { open, enter, exit, keep, close: () => setOpen(null) };
}

export { OPEN_DELAY_MS, CLOSE_DELAY_MS };
