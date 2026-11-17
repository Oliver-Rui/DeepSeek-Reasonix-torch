/** One line of a session's dispatch record: the role the target was opened
 *  under and the session the task landed on, as the hover card reads them.
 *  State, time and the target's title come from the durable audit replay; the
 *  richer fields are optional so an older hub's answer still reads. */
export interface DispatchThread {
  key?: string;
  role?: string;
  sessionPath?: string;
  targetTitle?: string;
  targetProject?: string;
  state?: "waiting" | "working" | "replied" | "declined";
  since?: string | number;
}

/** One target of a one-to-many dispatch: an existing conversation, or a new
 *  one opened in a project under the given role. */
export interface DispatchTarget {
  mode: "existing" | "new";
  sessionPath?: string;
  project?: string;
  role: string;
  instructions?: string;
  text: string;
}

export interface DispatchResult {
  ok: boolean;
  refusal?: { code: string; message: string };
}
