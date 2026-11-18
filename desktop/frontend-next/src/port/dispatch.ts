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

/** One end of a session-collaboration board row: a session that handed a task
 *  out or the one that received it. */
export interface DispatchBoardParty {
  key?: string;
  sessionPath?: string;
  title?: string;
  project?: string;
}

/** One task on the global board, from both ends' point of view. No task text:
 *  the kernel audit keeps only a digest, never the words. */
export interface DispatchBoardRow {
  dispatcher: DispatchBoardParty;
  assignee: DispatchBoardParty;
  role?: string;
  state: "waiting" | "working" | "replied" | "declined";
  since?: string;
  crossProject: boolean;
}
