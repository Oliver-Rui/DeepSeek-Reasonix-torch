import { t } from "../i18n";
import { UnreadDot } from "./UnreadMark";
import { ageWord } from "./age";

export { ageWord };

export function SessionAge({ at, unread, dot = true }: { at?: number; unread?: boolean; dot?: boolean }) {
  const date = at ? new Date(at) : null;
  const time = date && Number.isFinite(date.valueOf())
    ? <time className="sessage ds-meta" dateTime={date.toISOString()} title={date.toLocaleString()}>{ageWord(at!, Date.now())}</time>
    : null;
  return <>{unread && dot && <UnreadDot />}{time}</>;
}

export function SessionTail({ run, at, unread, dot }: { run?: string; at?: number; unread?: boolean; dot?: boolean }) {
  if (run === "running" || run === "halt") return <span className="sessword ds-meta" data-run={run}>{t(run === "running" ? "common.running" : "session.needs_you")}</span>;
  return <SessionAge at={at} unread={unread} dot={dot} />;
}
