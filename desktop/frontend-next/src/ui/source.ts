import { t } from "../i18n";
import type { PeerSource, Via } from "../port/wire";
import type { DeviceSelf } from "../port/share";

/** Where a message came from, said only when it was not this screen: another
 *  device by its number, or the window when this page is a paired device. A
 *  screen's own messages carry nothing, as they always have. */
export function messageSource(via: Via | undefined, viewer: DeviceSelf | null): string {
  if (via) return viewer?.id === via.device ? "" : t("chat.from_device_n", { n: via.ordinal });
  return viewer ? t("chat.from_the_computer") : "";
}

/** The other session a message came from, worded the way the shipped product
 *  words it: 来自「X」 for a task handed over, 「X」的回信 for a member's answer.
 *  The host wrote this label at delivery, so nothing here reads the body. */
export function peerSourceLabel(peer: PeerSource | undefined): string {
  if (!peer?.sessionId) return "";
  const title = peer.title?.trim() || t("chat.dispatch_thread_unnamed");
  return peer.reply ? t("chat.peer_source_reply", { title }) : t("chat.peer_source_from", { title });
}

/** Who answered a question this screen did not, with what they chose. */
export function answerSource(by: Via | "window" | undefined, viewer: DeviceSelf | null, said: string | undefined): string {
  const choice = (said ?? "").trim();
  if (by && by !== "window" && viewer?.id !== by.device) {
    return choice ? t("chat.answered_on_device_n_said", { n: by.ordinal, said: choice }) : t("chat.answered_on_device_n", { n: by.ordinal });
  }
  if (by === "window" && viewer) return choice ? t("chat.answered_on_the_computer_said", { said: choice }) : t("chat.answered_on_the_computer");
  return "";
}

/** Who settled a prompt this screen did not answer. */
export function decisionSource(by: Via | "window" | undefined, viewer: DeviceSelf | null): string {
  if (!by) return "";
  if (by === "window") return viewer ? t("chat.settled_on_the_computer") : "";
  return viewer?.id === by.device ? "" : t("chat.settled_on_device_n", { n: by.ordinal });
}
