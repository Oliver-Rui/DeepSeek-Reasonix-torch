import type { DragEvent } from "react";
import { t } from "../i18n";
import { WsMark } from "./WsMark";

const WORDS = {
  projects: {
    expand: "session.expand_all_projects",
    collapse: "session.collapse_all_projects",
    searching: "session.projects_expand_automatically_while_searching",
  },
  tasks: {
    expand: "session.expand_all_tasks",
    collapse: "session.collapse_all_tasks",
    searching: "session.tasks_expand_automatically_while_searching",
  },
} as const;

/** The label doubles as the handle that drags its run above the other one, so
 *  the drag handlers ride on whatever shape the label takes. */
export interface GroupDragProps {
  draggable?: boolean;
  onDragStart?: (e: DragEvent) => void;
  onDragOver?: (e: DragEvent) => void;
  onDragLeave?: (e: DragEvent) => void;
  onDrop?: (e: DragEvent) => void;
  onDragEnd?: (e: DragEvent) => void;
}

/** The tree's two standing labels: folders the person chose (项目), and the
 *  folder the window itself opened in (任务). Each label carries its run's
 *  fold-all, writing the same folded set the rows' own chevrons read, so one
 *  click rests the whole group and every chevron agrees with the label. The
 *  arrow rides beside the label's words: folded it points ahead, open it
 *  points down, and a run with nothing to fold shows no arrow at all. */
export function WorkspaceGroupHead({ group, folded, canFold, searching, onToggle, dragProps, dropping, dragging }: {
  group: "projects" | "tasks";
  folded: boolean;
  canFold?: boolean;
  searching?: boolean;
  onToggle?: () => void;
  dragProps?: GroupDragProps;
  dropping?: boolean;
  dragging?: boolean;
}) {
  const label = group === "projects" ? t("common.project") : t("session.tasks");
  const drag = {
    draggable: dragProps?.draggable,
    onDragStart: dragProps?.onDragStart,
    onDragOver: dragProps?.onDragOver,
    onDragLeave: dragProps?.onDragLeave,
    onDrop: dragProps?.onDrop,
    onDragEnd: dragProps?.onDragEnd,
    "data-drop": dropping ? "over" : undefined,
    "data-dragging": dragging ? "" : undefined,
  };
  if (!onToggle) {
    return <div className="wsgroup" {...drag}><span>{label}</span></div>;
  }
  const words = WORDS[group];
  const said = t(searching ? words.searching : folded ? words.expand : words.collapse);
  return (
    <button type="button" className="wsgroup" data-action="workspace.fold-all" data-value={group}
      disabled={!canFold || searching} aria-expanded={canFold ? !folded : undefined}
      title={said} aria-label={said} onClick={onToggle} {...drag}>
      <span>{label}</span>
      <WsMark open={canFold ? !folded : undefined} />
    </button>
  );
}
