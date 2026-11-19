import { t } from "../../i18n";
import type { PeerSource } from "../../port/wire";
import { StudioIcon } from "../StudioIcon";

/** The label on a message another session relayed. It names the sender the way
 *  the shipped product does — 来自「X」 for a task handed over, 「X」的回信 for a
 *  member's answer — and opens that conversation when the window can still
 *  reach it. The words come from the host's PeerSource, never from the body. */
export function PeerSourceChip({ peer, label, onOpen }: {
  peer: PeerSource;
  label: string;
  onOpen?: (path: string) => void;
}) {
  const path = peer.path?.trim();
  const canOpen = !!path && !!onOpen;
  const title = canOpen ? t("chat.peer_source_open", { title: peer.title ?? "" }) : t("chat.peer_source_gone");
  const body = (
    <>
      <StudioIcon name="send" aria-hidden="true" />
      <span>{label}</span>
    </>
  );
  if (!canOpen) return <span className="peersrc" title={title}>{body}</span>;
  return (
    <button type="button" className="peersrc" data-action="peer.source-open" data-session-path={path}
      title={title} aria-label={title} onClick={() => onOpen?.(path)}>
      {body}
    </button>
  );
}
